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
	//mod, err := setupTestShellModule()
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

func setupTestShellModule() (*Module, error) {
	writeTempFile([]byte("ls -al"), "shell")
	defer rmTempFile("/tmp/shell")
	return NewModule("shell", "foo=bar")
}

func TestNonexistentModuleResolve(t *testing.T) {
	//ModuleSearchPath = append(ModuleSearchPath, "/tmp")
	mod, err := setupTestShellModule()
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

func TestModuleDefaultExecOrder(t *testing.T) {
	mod, err := setupTestShellModule()
	if err != nil {
		t.Fatalf("There shouldn't have been any error. Got %s\n", err.Error())
	}

	if mod == nil {
		t.Errorf("Module shouldn't be nil")
	}

	execOrder, err := mod.ExecOrder()
	if err != nil {
		t.Fatalf("There shouldn't have been any error. Got %s\n", err.Error())
	}
	if execOrder[0] != "create_dir" ||
		execOrder[1] != "put_module" ||
		execOrder[2] != "exec_module" {
		t.Errorf("Exec Order sequence is wrong for a default module. Expected [create_dir put_module exec_module] but instead got ", execOrder)
	}
}

func TestModuleCopyExecOrder(t *testing.T) {
	writeTempFile([]byte("ls -al"), "copy")
	defer rmTempFile("/tmp/copy")
	mod, err := NewModule("copy", "src=foo dest=bar")

	if err != nil {
		t.Fatalf("There shouldn't have been any error. Got %s\n", err.Error())
	}

	if mod == nil {
		t.Errorf("Module shouldn't be nil")
	}
	execOrder, err := mod.ExecOrder()
	if err != nil {
		t.Fatalf("There shouldn't have been any error. Got %s\n", err.Error())
	}
	if execOrder[0] != "create_dir" ||
		execOrder[1] != "put_module" ||
		execOrder[2] != "copy_src" {
		t.Errorf("Exec Order sequence is wrong for a copy module. Expected [create_dir put_module copy_src] but instead got ", execOrder)
	}

}
