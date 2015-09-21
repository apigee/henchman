package henchman

import (
	"testing"
)

func TestTaskRun(t *testing.T) {
	mod := moduleTestSetup("sample_module")
	defer moduleTestTeardown(mod)

	origSearchPath := ModuleSearchPath
	ModuleSearchPath = append(ModuleSearchPath, "/tmp")
	defer func() {
		ModuleSearchPath = origSearchPath
	}()

	task := Task{}
	task.Name = "test"
	task.Module = mod

	testTransport := TestTransport{}

	localhost := Machine{}
	localhost.Hostname = "localhost"
	localhost.Transport = &testTransport

	err := task.Run(&localhost)
	if err != nil {
		t.Errorf("There shouldn't have been any errors. Got : %s\n", err.Error())
	}
}
