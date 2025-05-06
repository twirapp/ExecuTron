package runtime

import (
	"bytes"
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// getContainerLogs retrieves the stdout and stderr logs from the container.
func getContainerLogs(ctx context.Context, cli *client.Client, containerID string) (string, error) {
	logs, err := cli.ContainerLogs(
		ctx,
		containerID,
		container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to get logs: %v", err)
	}
	defer logs.Close()

	var buf bytes.Buffer
	_, err = stdcopy.StdCopy(&buf, &buf, logs)
	if err != nil {
		return "", fmt.Errorf("failed to read logs: %v", err)
	}

	return buf.String(), nil
}
