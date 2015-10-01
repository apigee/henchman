package henchman

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"text/scanner"
)

var ModuleSearchPath = []string{
	"modules",
}

// FIXME: Have custom error types when parsing modules
type Module struct {
	Name   string
	Params map[string]string
}

// Split args from the cli that are of the form,
// "a=x b=y c=z" as a map of form { "a": "b", "b": "y", "c": "z" }
// These plan arguments override the variables that may be defined
// as part of the plan file.
func parseModuleArgs(args string) (map[string]string, error) {
	var s scanner.Scanner
	s.Init(strings.NewReader(args))
	var tok rune
	extraArgs := make(map[string]string)
	currentKey := ""
	for tok != scanner.EOF {
		tok = s.Scan()
		tokText := s.TokenText()
		if strings.TrimSpace(tokText) == "" {
			continue
		}
		if currentKey == "" {
			if s.Peek() == 61 { // Peek for '='
				tok = s.Scan()
				currentKey = tokText
			} else {
				return nil, errors.New(fmt.Sprintf("Expected '=' at position %v", s.Pos()))
			}
		} else {
			extraArgs[currentKey] = strings.Trim(tokText, "\"")
			currentKey = ""
		}
	}
	return extraArgs, nil
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

// Module not found
func (module *Module) Resolve() (modulePath string, err error) {
	for _, dir := range ModuleSearchPath {
		fullPath := path.Join(dir, module.Name)
		finfo, err := os.Stat(fullPath)
		if finfo != nil && !finfo.IsDir() {
			return fullPath, err
		}
	}
	return "", errors.New("Module couldn't be resolved")
}

func (module *Module) ExecOrder() ([]string, error) {
	execOrder := map[string][]string{"default": []string{"create_dir", "put_module", "exec_module"},
		"copy": []string{"create_dir", "put_module", "copy_src"}}

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
