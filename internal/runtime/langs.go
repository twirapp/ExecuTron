package runtime

import "github.com/docker/docker/api/types/mount"

type containerContext struct {
	mounts         []mount.Mount
	cmd            []string
	containerImage string
	wrapperFile    string
	userCodeFile   string
	wrapperContent string
}

func createContainerContext(language string) *containerContext {
	var ctx *containerContext
	switch language {
	case "javascript":
		ctx = createJsContext()
	case "python":
		ctx = createPyContext()
	}

	if ctx == nil {
		return nil
	}

	return ctx
}

func createJsContext() *containerContext {
	wrapperFile := "wrapper.mjs"
	userCodeFile := "user_code.mjs"
	cmd := []string{"node", "/code/wrapper.mjs"}
	containerImage := "twirapp/executron:node-latest"
	wrapperContent := `
import { readFileSync } from 'fs';
import vm from 'node:vm'
import _ from 'lodash';

try {
	const code = readFileSync('/code/user_code.mjs', 'utf8');
	const result = await eval('(async () => { ' + code + ' })()');
	console.log(JSON.stringify({ result: result.toString() }));
} catch (e) {
	console.log(JSON.stringify({ error: e.message }));
}
`

	mounts := []mount.Mount{
		{
			Type:     mount.TypeBind,
			Source:   "/user_code.mjs",
			Target:   "/code/user_code.mjs",
			ReadOnly: true,
			BindOptions: &mount.BindOptions{
				Propagation: mount.PropagationRPrivate, // Ensure private mount propagation
			},
		},
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
		userCodeFile:   userCodeFile,
		wrapperContent: wrapperContent,
	}
}

func createPyContext() *containerContext {
	return nil
}
