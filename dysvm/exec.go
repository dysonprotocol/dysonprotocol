package dysvm

import (
	"fmt"

	"dysonprotocol.com/dysvm/internal/data"

	"github.com/kluctl/go-embed-python/embed_util"
	"github.com/kluctl/go-embed-python/python"
)

func Exec(msgJSON, scriptJSON, attachedMsgResultsJSON, headerInfoJSON, port string) (string, error) {
	var lib *embed_util.EmbeddedFiles
	ep, err := python.NewEmbeddedPython("dyslang")
	if err != nil {
		return "", err
	}

	lib, err = embed_util.NewEmbeddedFiles(data.Data, "dyslang-libs")

	if err != nil {
		return "", err
	}
	// TODO Make this an environment variable or config
	ep.AddPythonPath("./dysvm/internal/py-dyslang")
	ep.AddPythonPath(lib.GetExtractedPath())

	cmd, err := ep.PythonCmd("-m", "dyslang", "exec_script", msgJSON, scriptJSON, attachedMsgResultsJSON, headerInfoJSON, port)

	if err != nil {
		return "", err
	}

	out, runErr := cmd.CombinedOutput()

	return string(out), runErr

}

func Wsgi(port, scriptJSON, blockInfoJSON, httpreq string) (string, error) {
	var lib *embed_util.EmbeddedFiles
	ep, err := python.NewEmbeddedPython("dyslang")
	if err != nil {
		return "", err
	}

	lib, err = embed_util.NewEmbeddedFiles(data.Data, "dyslang-libs")

	if err != nil {
		return "", err
	}
	// TODO Make this an environment variable or config
	ep.AddPythonPath("./dysvm/internal/py-dyslang")
	ep.AddPythonPath(lib.GetExtractedPath())

	cmd, err := ep.PythonCmd("-m", "dyslang", "run_wsgi", port, scriptJSON, blockInfoJSON, httpreq)

	if err != nil {
		return "", err
	}

	out, runErr := cmd.CombinedOutput()

	fmt.Println("Command output: ", string(out))
	fmt.Println("Command error: ", runErr)

	return string(out), runErr

}
