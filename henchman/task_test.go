package henchman

import (
	"fmt"
	"io/ioutil"
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

	_, err := task.Run(&localhost)
	if err != nil {
		t.Errorf("There shouldn't have been any errors. Got : %s\n", err.Error())
	}
}

func TestTaskRender(t *testing.T) {
	buf, err := ioutil.ReadFile("test/plan/planWithPongo2.yaml")
	if err != nil {
		t.Errorf("Could not read planWithPongo2.yaml")
	}

	plan, err := PreprocessPlan(buf, nil)
	if err != nil {
		t.Fatalf("This plan shouldn't be having an error - %s\n", err.Error())
	}

	testTransport := TestTransport{}
	localhost := Machine{}
	localhost.Hostname = "localhost"
	localhost.Transport = &testTransport

	err = plan.Tasks[0].Render(&localhost)
	if err != nil {
		t.Fatalf("There shouldn't have been any errors. Got : %s\n", err.Error())
	}

	if plan.Tasks[0].Name != "iptables at localhost" {
		t.Errorf("Expected iptables at localhost.  Received %v\n", plan.Tasks[0].Name)
	}

	fmt.Println(plan.Tasks[0].Module)
}
