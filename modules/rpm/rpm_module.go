package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"encoding/json"
)

type RpmModule struct {
	Name        string
	State       string
	Url         string
	Replacepkgs string
}

var result map[string]interface{} = map[string]interface{}{}
var yumSubcommands map[string]string = map[string]string{
	"present": "install",
	"absent":  "remove",
}

func main() {
	// recover code
	// also does the printout of result
	defer func() {
		if r := recover(); r != nil {
			result["status"] = "error"
			result["msg"] = r
		}

		output, err := json.Marshal(result)
		if err != nil {
			panic(err)
		}
		fmt.Print(string(output))
	}()

	rpmParams := RpmModule{}

	// basically unmarshall but can take in a io.Reader
	dec := json.NewDecoder(os.Stdin)
	if err := dec.Decode(&rpmParams); err != nil {
		panic(err.Error())
	}

	if rpmParams.Url == "" {
		panic("Missing required param 'url'")
	}
	rpmParams.Url = fmt.Sprintf("'%s'", rpmParams.Url)

	msg := ""
	var cmds []string
	if rpmParams.State == "present" || rpmParams.State == "" {
		cmds = []string{"-ivh", rpmParams.Url}
	} else if rpmParams.State == "absent" {
		cmds = []string{"-e", rpmParams.Url}
	} else {
		panic(fmt.Sprintf("State should be either 'present' or 'absent'. Got %s", rpmParams.State))
	}

	for _, x := range []string{"yes", "true", "True"} {
		if rpmParams.Replacepkgs == x {
			cmds = append(cmds, "--replacepkgs")
			break
		}
	}

	cmd := exec.Command("rpm", cmds...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		panic(fmt.Sprintf("Error occurred while installing/removing package %s - %s", rpmParams.Url, stderr.String()))
	}

	result["status"] = "changed"
	result["msg"] = msg
	result["output"] = map[string]string{
		"stdout": stdout.String(),
		"stderr": stderr.String(),
	}
}
