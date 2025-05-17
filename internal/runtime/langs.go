package runtime

import (
	"fmt"
	"github.com/docker/docker/api/types/mount"
	"strings"
)

type containerContext struct {
	mounts         []mount.Mount
	cmd            []string
	containerImage string
	wrapperFile    string
	wrapperContent string
}

func createContainerContext(language, code string) *containerContext {
	var ctx *containerContext
	switch language {
	case "javascript":
		ctx = createJsContext(code)
	case "python":
		ctx = createPyContext(code)
	}

	if ctx == nil {
		return nil
	}

	return ctx
}

func createJsContext(code string) *containerContext {
	wrapperFile := "wrapper.mjs"
	cmd := []string{"node", "/code/wrapper.mjs"}
	containerImage := "twirapp/executron:node-latest"
	wrapperContent := fmt.Sprintf(
		`
import { readFileSync } from 'fs';
import vm from 'node:vm'
import _ from 'lodash';

const consoleRegex = /console\.(log|debug|info|warn|error|table|trace|group|groupEnd|time|timeEnd)\s*\([^;]*\);?/g;

async function ExecutronExecuteJsCode() {
	%s
}

try {
	const result = await ExecutronExecuteJsCode()
	console.log(JSON.stringify({ result: result.toString().replace(consoleRegex, '') }));
} catch (e) {
	console.log(JSON.stringify({ error: e.message }));
}
`, code,
	)

	mounts := []mount.Mount{
		{
			Type:     mount.TypeBind,
			Source:   "/wrapper.mjs",
			Target:   "/code/wrapper.mjs",
			ReadOnly: true,
			BindOptions: &mount.BindOptions{
				Propagation: mount.PropagationRPrivate, // Ensure private mount propagation
			},
		},
	}

	return &containerContext{
		mounts:         mounts,
		cmd:            cmd,
		containerImage: containerImage,
		wrapperFile:    wrapperFile,
		wrapperContent: wrapperContent,
	}
}

func createPyContext(code string) *containerContext {
	var codeLines []string
	for _, line := range strings.Split(code, "\n") {
		codeLines = append(codeLines, fmt.Sprintf("    %s", line))
	}

	wrapperFile := "wrapper.py"
	cmd := []string{"python", "/code/wrapper.py"}
	containerImage := "python:3.12-alpine"

	wrapperContent := fmt.Sprintf(
		`
def __code_wrapper():
    %s

import json, os, sys, io
from contextlib import redirect_stdout

sys.stdin = io.StringIO("")

try:
    with redirect_stdout(open(os.devnull, "w")):
        result = __code_wrapper()
except Exception as e:
    print(json.dumps({"error": str(e)}))
else:
    print(json.dumps({"result": str(result)}))
`, strings.Join(codeLines, "\n"),
	)

	mounts := []mount.Mount{
		{
			Type:     mount.TypeBind,
			Source:   "/wrapper.py",
			Target:   "/code/wrapper.py",
			ReadOnly: true,
			BindOptions: &mount.BindOptions{
				Propagation: mount.PropagationRPrivate, // Ensure private mount propagation
			},
		},
	}

	return &containerContext{
		mounts:         mounts,
		cmd:            cmd,
		containerImage: containerImage,
		wrapperFile:    wrapperFile,
		wrapperContent: wrapperContent,
	}
}
