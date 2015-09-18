package henchman

import (
	"testing"
)

func TestValidModule(t *testing.T) {
	name := "shell"
	args := "cmd=ls"
	mod, err := NewModule(name, args)
	if err != nil {
		t.Fatalf("Error when creating the module - %s\n", err.Error())
	}
	if mod.Name != name {
		t.Errorf("Mod name should have been %s. Got %s instead\n", name, mod.Name)
	}
	if mod.Params["cmd"] != "ls" {
		t.Errorf("Mod params wasn't initialized properly")
	}
}

func TestInvalidArgsModule(t *testing.T) {
	name := "invalid"
	args := "foo"
	_, err := NewModule(name, args)
	if err == nil {
		t.Errorf("Module arg parsing should have failed")
	}
}
