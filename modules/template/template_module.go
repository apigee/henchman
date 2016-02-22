package main

import (
	_ "bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"encoding/json"
)

type TemplateModule struct {
	Owner    string
	Group    string
	Mode     string
	Dest     string
	RmtSrc   string
	Override string
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

	templateParams := TemplateModule{}

	// basically unmarshall but can take in a io.Reader
	dec := json.NewDecoder(os.Stdin)
	if err := dec.Decode(&templateParams); err != nil {
		panic(err.Error())
	}

	if templateParams.Dest == "" {
		panic("Required parameter 'dest' not found")
	}

	// Override param is passed in as a string and bools tend to default to false
	override := true
	str := strings.ToLower(templateParams.Override)
	if str == "false" {
		override = false
	} else if str != "" && str != "true" {
		panic("override param must be true or false")
	}

	// Creates all necessary nested directories
	_, err := os.Stat(templateParams.Dest)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(templateParams.Dest, 0755); err != nil {
			panic(fmt.Sprintf("Error creating directories - %s", err.Error()))
		}
	}

	if override {
		// Removes the last file/folder
		if err := os.RemoveAll(templateParams.Dest); err != nil {
			panic(fmt.Sprintf("Error removing endpoint for override - %s", err.Error()))
		}
	} else {
		// extends dest
		templateParams.Dest = filepath.Join(templateParams.Dest, filepath.Base(templateParams.RmtSrc))
	}

	// moves the file/folder to the destination
	if err := os.Rename(templateParams.RmtSrc, templateParams.Dest); err != nil {
		panic(fmt.Sprintf("Error moving file/folder - %s", err.Error()))
	}

	// using chown command os.Chown(...) only takes in ints for now GO 1.5
	var cmd *exec.Cmd
	cmdList := []string{"-R", templateParams.Owner + ":" + templateParams.Group, templateParams.Dest}
	cmd = exec.Command("/bin/chown", cmdList...)

	if output, err := cmd.CombinedOutput(); err != nil {
		panic(fmt.Sprintf("Error chown file/folder - %s", string(output)))
	}

	if templateParams.Mode != "" {
		i, err := strconv.ParseInt(templateParams.Mode, 8, 32)
		if err != nil {
			panic(fmt.Sprintf("Error retrieving mode - %s", err.Error()))
		}
		if i < 0 {
			panic("Error mode must be an unsigned integer")
		}

		if err := os.Chmod(templateParams.Dest, os.FileMode(i)); err != nil {
			panic(fmt.Sprintf("Error chmod file/folder - %s", err.Error()))
		}
	}

	result["status"] = "changed"
	result["msg"] = fmt.Sprintf("State of '%s' changed", templateParams.Dest)
}
