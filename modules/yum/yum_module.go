package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"encoding/json"
)

type YumModule struct {
	Name    string
	Version string
	State   string
	Repo    string
}

func (ym *YumModule) IsInstalled() bool {
	// can also use 'yum list installed <package>-<version>'
	cmd := exec.Command("rpm", "-qa", ym.Name)
	output, err := cmd.Output()
	if err != nil {
		panic(err.Error())
	}

	for _, pkg := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(pkg, ym.Name) {
			if ym.Version != "" {
				if strings.HasSuffix(pkg, ym.Version) {
					return true
				}
			} else {
				return true
			}
		}
	}

	return false
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

	yumParams := YumModule{}

	// basically unmarshall but can take in a io.Reader
	dec := json.NewDecoder(os.Stdin)
	if err := dec.Decode(&yumParams); err != nil {
		panic(err.Error())
	}

	// checks if there's a specific repo needed
	if yumParams.Repo != "" {
		yumParams.Repo = "--enablerepo=" + yumParams.Repo
	}

	// if no state is present assume it's present
	if yumParams.State == "" {
		yumParams.State = "present"
	}

	if yumParams.State != "present" && yumParams.State != "absent" {
		panic("Valid states are 'present' or 'absent'")
	}

	if yumParams.Version == "latest" {
		yumParams.Version = ""
	}

	fullPkgName := yumParams.Name
	if yumParams.Version != "" {
		fullPkgName += "-" + yumParams.Version
	}

	installed := yumParams.IsInstalled()

	if installed && yumParams.State == "present" {
		result["msg"] = fmt.Sprintf("Package %s already present", fullPkgName)
		result["status"] = "ok"
	} else if !installed && yumParams.State == "absent" {
		result["msg"] = fmt.Sprintf("Package %s already absent", fullPkgName)
		result["status"] = "ok"
	} else {
		subCommand := yumSubcommands[yumParams.State]
		cmd := exec.Command("yum", subCommand, "-y", fullPkgName, yumParams.Repo)

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			panic(fmt.Sprintf("Error occurred while installing/removing package %s - %s", fullPkgName, stderr.String()))
		}

		result["status"] = "changed"
		result["msg"] = fmt.Sprintf("State of package %s changed - %s", fullPkgName, stdout.String())
		result["output"] = map[string]string{
			"stdout": stdout.String(),
			"stderr": stderr.String(),
		}
	}
}
