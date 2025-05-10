package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/daemon/network"
	"github.com/google/uuid"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"
)

type Response struct {
	Result string `json:"result"`
	Error  string `json:"error,omitempty"`
}

type Executor struct {
	maxRunedContainers int32
	runedContainers    atomic.Int32
}

func New() *Executor {
	e := &Executor{
		maxRunedContainers: 0,
		runedContainers:    atomic.Int32{},
	}

	maxRunedContainers := os.Getenv("MAX_PARALLEL_CONTAINERS")
	if maxRunedContainers == "" {
		e.maxRunedContainers = 100
		slog.Warn("MAX_PARALLEL_CONTAINERS not set, using default value of 100")
	} else {
		parsed, err := strconv.Atoi(maxRunedContainers)
		if err == nil {
			e.maxRunedContainers = int32(parsed)
		} else {
			slog.Warn("Failed to parse MAX_PARALLEL_CONTAINERS, using default value of 100")
		}
	}

	return e
}

func (c *Executor) ExecuteCode(ctx context.Context, language, code string) (*Response, error) {
	for c.runedContainers.Load() >= c.maxRunedContainers {
		time.Sleep(100 * time.Millisecond)
	}

	c.runedContainers.Add(1)
	defer c.runedContainers.Add(-1)

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %v", err)
	}
	defer cli.Close()

	// Create a temporary directory for the code
	tempDir, err := os.MkdirTemp("", "code-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set directory permissions to 0755 (world-readable/executable)
	if err := os.Chmod(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to set temp dir permissions: %v", err)
	}

	containerCtx := createContainerContext(language, code)
	if containerCtx == nil {
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	// Write wrapper script to a file
	wrapperPath := filepath.Join(tempDir, containerCtx.wrapperFile)
	if err := os.WriteFile(wrapperPath, []byte(containerCtx.wrapperContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write wrapper file: %v", err)
	}
	if err := os.Chmod(wrapperPath, 0644); err != nil {
		return nil, fmt.Errorf("failed to set wrapper file permissions: %v", err)
	}

	// ensure containerImage is pulled
	if _, fetchErr := cli.ImageInspect(ctx, containerCtx.containerImage); fetchErr != nil {
		pulled, pullErr := cli.ImagePull(
			ctx,
			containerCtx.containerImage,
			image.PullOptions{},
		)
		if pullErr != nil {
			return nil, fmt.Errorf("failed to pull containerImage: %v", pullErr)
		}
		defer pulled.Close()
		// Read and discard the pull response to ensure completion
		if _, pullErr := io.Copy(io.Discard, pulled); pullErr != nil {
			return nil, fmt.Errorf(
				"failed to read pull response for image %s: %v",
				containerCtx.containerImage,
				pullErr,
			)
		}
	}

	for i, m := range containerCtx.mounts {
		containerCtx.mounts[i].Source = tempDir + m.Source
	}

	// Create and configure the container
	containerConfig := &container.Config{
		Image:      containerCtx.containerImage,
		Cmd:        containerCtx.cmd,
		WorkingDir: "/code",
		Tty:        false,
	}

	networkMode := container.NetworkMode(network.DefaultNetwork)
	if os.Getenv("APP_ENV") == "production" {
		networkMode = container.NetworkMode("container:executron-warp")
	}

	pidsLimit := int64(100)
	hostConfig := &container.HostConfig{
		Mounts: containerCtx.mounts,
		Resources: container.Resources{
			Memory:    128 * 1024 * 1024, // 128 MB
			NanoCPUs:  1000000000,        // 1 CPU
			PidsLimit: &pidsLimit,        // Limit to 100 PIDs
		},
		NetworkMode:    networkMode, // Disable networking
		ReadonlyRootfs: true,        // Read-only root filesystem
		AutoRemove:     true,        // Remove container after execution
		SecurityOpt:    []string{"no-new-privileges"},
		CapDrop:        []string{"ALL"}, // Drop all capabilities
		//Runtime:        "runsc",
	}

	// Create a unique container name
	containerName := "code-exec-" + uuid.New().String()

	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, containerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %v", err)
	}

	// Start the container

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("failed to start container: %v", err)
	}

	// Wait for the container to finish with a timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	statusCh, errCh := cli.ContainerWait(timeoutCtx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		return nil, fmt.Errorf("container wait error: %v", err)
	case status := <-statusCh:
		if status.StatusCode != 0 {
			// Get container logs for error details
			logs, logErr := getContainerLogs(ctx, cli, resp.ID)
			if logErr != nil {
				return nil, fmt.Errorf(
					"container failed with status %d, log error: %v",
					status.StatusCode,
					logErr,
				)
			}
			return &Response{Error: logs}, nil
		}
	case <-timeoutCtx.Done():
		// Stop the container on timeout
		cli.ContainerStop(ctx, resp.ID, container.StopOptions{})
		return nil, fmt.Errorf("execution timed out")
	}

	// Get container logs (stdout/stderr)
	logs, err := getContainerLogs(ctx, cli, resp.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container logs: %v", err)
	}

	// Parse the JSON output
	var output struct {
		Result string `json:"result"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal([]byte(logs), &output); err != nil {
		return nil, fmt.Errorf("failed to parse output: %v", err)
	}

	return &Response{Result: output.Result, Error: output.Error}, nil
}
