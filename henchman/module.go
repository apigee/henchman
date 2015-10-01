package henchman

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"text/scanner"

	"gopkg.in/yaml.v2"
)

var ModuleSearchPath = []string{
	"modules",
}

const MODULE_TYPE_FILE = "moduleTypes.yml"

// FIXME: Have custom error types when parsing modules
type Module struct {
	Name   string
	Params map[string]string
}

type ModuleTypes struct {
	ModuleTypeOrders []ModuleTypeOrder `yaml:"modules"`
}

type ModuleTypeOrder struct {
	Type  string
	Order []string
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

func getModulePath() (string, error) {
	curDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	moduleTypePath := path.Join(curDir, MODULE_TYPE_FILE)
	if _, err := os.Stat(moduleTypePath); err != nil {
		moduleTypePath = path.Join(curDir, "henchman", MODULE_TYPE_FILE)
		if _, err := os.Stat(moduleTypePath); err != nil {
			return "", err
		}
	}
	return moduleTypePath, nil
}

func (module *Module) ExecOrder() ([]string, error) {
	moduleTypePath, err := getModulePath()
	log.Println(moduleTypePath)
	yamlFile, err := ioutil.ReadFile(moduleTypePath)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var moduleTypes ModuleTypes

	err = yaml.Unmarshal(yamlFile, &moduleTypes)
	if err != nil {
		return nil, err
	}
	var defaultOrder []string
	for _, moduleTypeOrder := range moduleTypes.ModuleTypeOrders {
		if moduleTypeOrder.Type == module.Name {
			return moduleTypeOrder.Order, nil
		}
		if moduleTypeOrder.Type == "default" {
			defaultOrder = moduleTypeOrder.Order
		}
	}
	//default
	return defaultOrder, nil
}
