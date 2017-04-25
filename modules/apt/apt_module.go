package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"encoding/json"
	"bytes"
)

const aptPackageStatePresent = "present"
const aptPackageStateAbsent = "absent"

type AptPackage struct {
	Name    string
	Version string
	State   string
}

func (pkg *AptPackage) getFullPackageName() string {
	fullPkgName := pkg.Name
	if len(pkg.Version) > 0 {
		fullPkgName += "=" + pkg.Version
	}

	return fullPkgName
}

func (pkg *AptPackage) execCmd(command string) (string, string, error) {
	environment := os.Environ()
	shell := "/bin/sh"
	
	cmd := exec.Command(shell, "-c", command)
	cmd.Stderr = nil
	cmd.Env = environment
	
	var outbuf, errbuf bytes.Buffer
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf
	
	err := cmd.Run()
	
	return outbuf.String(), errbuf.String(), err
}

func (pkg *AptPackage) IsInstalled() bool {
	stdout, stderr, err := pkg.execCmd(fmt.Sprintf("apt list --installed %s", pkg.Name))

	if err != nil {
		panic(err.Error() + "\n\nSTDOUT:\n" + stdout + "\n\nSTDERR:\n" + stderr)
	}
	
	outputArr := strings.Split(stdout, "\n");
	
	for k := range outputArr {
		line := outputArr[k]
		if strings.Contains(line, pkg.Name) {
			if len(pkg.Version) > 0 {
				if strings.Contains(line, pkg.Version) {
					return true
				}
				return false
			}
			return true
		}
	}
	
	return false
}

func (pkg *AptPackage) Install() {
	fullPkgName := pkg.getFullPackageName()
	stdout, stderr, err :=  pkg.execCmd(fmt.Sprintf("apt install -y %s", fullPkgName))

	if err != nil {
		panic(fmt.Sprintf("Error occurred while installing package '%s': %s", fullPkgName, stderr))
	}
	
	result["status"] = "changed"
	result["msg"] = fmt.Sprintf("State of package '%s' changed", fullPkgName)
	result["output"] = stdout
}

func (pkg *AptPackage) Remove() {
	fullPkgName := pkg.getFullPackageName()
	stdout, stderr, err :=  pkg.execCmd(fmt.Sprintf("apt remove -y %s", fullPkgName))

	if err != nil {
		panic(fmt.Sprintf("Error occurred while removing package '%s': %s", fullPkgName, stderr))
	}

	result["status"] = "changed"
	result["msg"] = fmt.Sprintf("State of package '%s' changed", fullPkgName)
	result["output"] = stdout
}

var result map[string]interface{} = map[string]interface{}{}

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

	aptPackage := AptPackage{}

	// basically unmarshall but can take in a io.Reader
	dec := json.NewDecoder(os.Stdin)
	if err := dec.Decode(&aptPackage); err != nil {
		panic(err.Error())
	}

	// if no state is present assume it's present
	if aptPackage.State == "" {
		aptPackage.State = aptPackageStatePresent
	}

	if aptPackage.State != aptPackageStatePresent && aptPackage.State != aptPackageStateAbsent {
		panic("Valid states are 'present' or 'absent'")
	}
	
	if strings.Contains(aptPackage.Name, "=") {
		fullPkgNameParts := strings.Split(aptPackage.Name, "=")
		aptPackage.Name = fullPkgNameParts[0]
		aptPackage.Version = fullPkgNameParts[1]
	}

	if aptPackage.Version == "latest" {
		aptPackage.Version = ""
	}

	fullPkgName := aptPackage.getFullPackageName()

	installed := aptPackage.IsInstalled()

	if installed && aptPackage.State == aptPackageStatePresent {
		result["msg"] = fmt.Sprintf("Package %s already %s", fullPkgName, aptPackageStatePresent)
		result["status"] = "ok"
		result["output"] = ""
	} else if !installed && aptPackage.State == aptPackageStateAbsent {
		result["msg"] = fmt.Sprintf("Package %s already %s", fullPkgName, aptPackageStateAbsent)
		result["status"] = "ok"
		result["output"] = ""
	} else {
		if aptPackageStatePresent == aptPackage.State {
			aptPackage.Install()
		} else {
			aptPackage.Remove()
		}
	}
}
