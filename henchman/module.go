package henchman

import (
	"errors"
	"strings"
)

type Module struct {
	Name   string
	Params map[string]string
}

// Split args from the cli that are of the form,
// "a=x b=y c=z" as a map of form { "a": "b", "b": "y", "c": "z" }
// These plan arguments override the variables that may be defined
// as part of the plan file.
func parseModuleArgs(args string) (map[string]string, error) {
	extraArgs := make(map[string]string)
	if args == "" {
		return extraArgs, nil
	}
	for _, a := range strings.Split(args, " ") {
		kv := strings.Split(a, "=")
		if len(kv) != 2 {
			return nil, errors.New("Invalid module parameters")
		}
		extraArgs[kv[0]] = kv[1]
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
