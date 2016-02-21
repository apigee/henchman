package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"encoding/json"
)

type ShellModule struct {
	Cmd   string
	Chdir string
	Env   string
	Shell string
}

var result map[string]interface{} = map[string]interface{}{}

func main() {
	// recover code
	// also does the printout of result
	defer func() {
		if r := recover(); r != nil {
			result["status"] = "error"
			result["msg"] = fmt.Sprintf("Command exec'ed with errors.  Error - %s", r)
		}

		output, err := json.Marshal(result)
		if err != nil {
			panic(err)
		}

		result["msg"] = result["msg"].(string) + fmt.Sprintf(" - %v", len(output))

		output, err = json.Marshal(result)
		if err != nil {
			panic(err)
		}
		fmt.Print(string(output))
	}()

	shellParams := ShellModule{}

	// basically unmarshall but can take in a io.Reader
	dec := json.NewDecoder(os.Stdin)
	if err := dec.Decode(&shellParams); err != nil {
		panic(err.Error())
	}

	if shellParams.Cmd == "" {
		panic("Required parameter 'cmd' not found")
	}

	if shellParams.Shell == "" {
		shellParams.Shell = "sh"
	}
	if err := setEnv(shellParams.Env); err != nil {
		panic("While setting env vars, " + err.Error())
	}

	var cmd *exec.Cmd
	cmd = exec.Command("/bin/"+shellParams.Shell, "-c", shellParams.Cmd)
	cmd.Dir = shellParams.Chdir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		result["status"] = "failure"
	} else {
		result["status"] = "changed"
	}

	result["msg"] = "executed command"
	result["output"] = map[string]string{
		"stdout": stdout.String(),
		"stderr": stderr.String(),
	}
}

// setEnv expects a string of "key=val key=val key=val" and adds them to the current env
func setEnv(envStr string) error {
	envList := strings.Split(envStr, " ")
	for _, envKeyVal := range envList {
		if strings.ContainsAny(envKeyVal, "=") {
			keyVal := strings.Split(envKeyVal, "=")
			if err := os.Setenv(keyVal[0], keyVal[1]); err != nil {
				return err
			}
		}
	}

	return nil
}
