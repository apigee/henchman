package henchman

import (
	"os"
	"path"
	"testing"
)

func moduleTestSetup(modName string) (module *Module) {
	moduleContent := `
#!/usr/bin/env sh
ls -al $1
`
	writeTempFile([]byte(moduleContent), modName)

	mod, _ := NewModule(modName, "")
	return mod
}

func moduleTestTeardown(mod *Module) {
	os.Remove(path.Join("/tmp", mod.Name))
}

func TestValidModule(t *testing.T) {
	name := "shell"
	args := "cmd=\"ls -al\" foo=bar baz=☃"
	mod, err := NewModule(name, args)
	if err != nil {
		t.Fatalf("Error when creating the module - %s\n", err.Error())
	}
	if mod.Name != name {
		t.Errorf("Mod name should have been %s. Got %s instead\n", name, mod.Name)
	}
	if mod.Params["cmd"] != "ls -al" {
		t.Errorf("Mod params wasn't initialized properly")
	}
	if mod.Params["foo"] != "bar" {
		t.Errorf("Expected value for foo to be bar. Got %s instead\n", mod.Params["foo"])
	}
	if mod.Params["baz"] != "☃" {
		t.Errorf("Expected snowman. Got %s instead\n", mod.Params["baz"])
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

func TestInvalidArgsModule2(t *testing.T) {
	name := "invalid"
	args := "foo bar=baz"
	_, err := NewModule(name, args)
	if err == nil {
		t.Errorf("Module arg parsing should have failed")
	}
}

func TestModuleResolve(t *testing.T) {
	origSearchPath := ModuleSearchPath
	ModuleSearchPath = append(ModuleSearchPath, "/tmp")
	defer func() {
		ModuleSearchPath = origSearchPath
	}()
	writeTempFile([]byte("ls -al"), "shell")
	defer rmTempFile("/tmp/shell")
	mod, err := NewModule("shell", "foo=bar")
	if err != nil {
		t.Fatalf("There shouldn't have been any error. Got %s\n", err.Error())
	}
	if mod == nil {
		t.Errorf("Module shouldn't be nil")
	}
	fullPath, err := mod.Resolve()
	if err != nil {
		t.Fatalf("Error when resolving module path - %s\n", err.Error())
	}
	if fullPath != "/tmp/shell" {
		t.Errorf("Got incorrect fullPath - %s\n", fullPath)
	}
}

func TestNonexistentModuleResolve(t *testing.T) {
	//ModuleSearchPath = append(ModuleSearchPath, "/tmp")
	writeTempFile([]byte("ls -al"), "shell")
	defer rmTempFile("/tmp/shell")
	mod, err := NewModule("shell", "foo=bar")
	if err != nil {
		t.Fatalf("There shouldn't have been any error. Got %s\n", err.Error())
	}
	if mod == nil {
		t.Errorf("Module shouldn't be nil")
	}
	fullPath, err := mod.Resolve()
	if err == nil {
		t.Error("Module path resolution should have failed")
	}
	if fullPath != "" {
		t.Error("Fullpath should have been empty")
	}
}
