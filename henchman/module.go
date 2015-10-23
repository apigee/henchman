package henchman

import (
	//"errors"
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
)

var ModuleSearchPath = []string{
	"modules",
}

// FIXME: Have custom error types when parsing modules
type Module struct {
	Name   string
	Params map[string]string
}

func getRemainingToken(str []byte, sep byte) ([]byte, error) {
	readbuffer := bytes.NewBuffer(str)
	reader := bufio.NewReader(readbuffer)
	remainingToken, err := reader.ReadBytes(sep)
	return remainingToken, err
}

// Split args from the cli that are of the form,
// "a=x b=y c=z" as a map of form { "a": "b", "b": "y", "c": "z" }
// These plan arguments override the variables that may be defined
// as part of the plan file.
func parseModuleArgs(args string) (map[string]string, error) {
	extraArgs := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(args))

	split := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		advance, nextToken, err := bufio.ScanWords(data, atEOF)
		tokenParts := strings.Split(string(nextToken), "=")
		seps := []byte{'"', '\''}
		for _, sep := range seps {
			if len(tokenParts) > 1 && tokenParts[1][0] == sep && tokenParts[1][len(tokenParts[1])-1] != sep {
				//get the remaining token
				remainingToken, err := getRemainingToken(data[(advance-1):], sep)
				if err == nil {
					token = append(nextToken, remainingToken...)
					break
				}
			} else {
				token = nextToken
			}
		}
		return
	}

	scanner.Split(split)
	// Validate the input
	for scanner.Scan() {
		text := scanner.Text()
		if extraArgsHasText(extraArgs, text) {
			continue
		} else if strings.Contains(text, "=") {
			splitValues := strings.Split(text, "=")
			//this may happen for cases where '=' is in the string
			if len(splitValues) > 2 {
				buffer := bytes.NewBufferString(splitValues[1])
				for i := 2; i < len(splitValues); i++ {
					buffer.WriteString("=")
					buffer.WriteString(splitValues[i])
				}
				splitValues[1] = buffer.String()
			}
			extraArgs[splitValues[0]] = splitValues[1]
		} else {
			// this check takes care of 2nd part of " def'" part of 'abc def'
			return nil, errors.New("Module args are invalid")
		}
	}
	// remove all quotes. Value for the respective key
	// should not have quotes
	extraArgs = stripQuotes(extraArgs)

	if err := scanner.Err(); err != nil {
		fmt.Printf("Invalid input: %s", err)
		return extraArgs, err
	}
	return extraArgs, nil
}

func stripQuotes(args map[string]string) map[string]string {
	removeQuotes := func(r rune) rune {
		if r == '"' || r == '\'' {
			return -1
		}
		return r
	}
	for k, v := range args {
		args[k] = strings.Map(removeQuotes, v)
	}
	return args
}

func extraArgsHasText(extraArgs map[string]string, text string) bool {
	for _, v := range extraArgs {
		if strings.Contains(v, text) {
			return true
		}
	}
	return false
}

func NewModule(name string, params string) (*Module, error) {
	module := Module{}
	module.Name = name
	paramTable, err := parseModuleArgs(params)
	if err != nil {
		return nil, err
	}
	module.Params = paramTable
	return &module, nil
}

func (module *Module) Resolve() (modulePath string, err error) {
	for _, dir := range ModuleSearchPath {
		fullPath := path.Join(dir, module.Name)
		finfo, err := os.Stat(fullPath)
		if finfo != nil {
			// checks if the module is a standalone script
			// else checks the module dir if there is a file named exec
			if !finfo.IsDir() {
				return fullPath, err
			} else {
				fullPath = path.Join(fullPath, "exec")
				finfo, err = os.Stat(fullPath)
				if finfo != nil && !finfo.IsDir() {
					return fullPath, err
				}
			}
		}
	}
	return "", fmt.Errorf("Module %s couldn't be resolved", module.Name)
}

func (module *Module) ExecOrder() ([]string, error) {
	execOrder := map[string][]string{"default": []string{"create_dir", "put_module", "exec_module"},
		"copy": []string{"create_dir", "put_module", "put_file", "copy_remote", "exec_module"},
		"template": []string{"create_dir", "put_module", "process_template", "put_file", "copy_remote",
			"reset_src", "exec_module"}}

	var defaultOrder []string
	for moduleType, order := range execOrder {
		if moduleType == module.Name {
			return order, nil
		}
		if moduleType == "default" {
			defaultOrder = order
		}
	}
	//default
	return defaultOrder, nil
}
