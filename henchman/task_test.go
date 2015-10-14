package henchman

import (
	//"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskRunForSingleMachine(t *testing.T) {
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

	regMap := make(RegMap)

	_, err := task.Run(&localhost, task.Vars, regMap)
	require.NoError(t, err, "There shouldn't have been any errors")
}

func TestTaskRenderAndProcessWhen(t *testing.T) {
	buf, err := ioutil.ReadFile("test/plan/planWithPongo2.yaml")
	require.NoError(t, err, "Could not read planWithPongo2.yaml")

	plan, err := PreprocessPlan(buf, Inventory{})
	require.NoError(t, err, "This plan shouldn't be having an error")

	testTransport := TestTransport{}
	localhost := Machine{}
	localhost.Hostname = "localhost"
	localhost.Transport = &testTransport

	regMap := make(RegMap)
	regMap["cmd"] = "touch"
	regMap["name"] = "Task 2"

	for _, task := range plan.Tasks {
		err = task.Render(task.Vars, regMap)
		require.NoError(t, err)
	}

	task := plan.Tasks[0]
	assert.Equal(t, "iptables with abcd1234", task.Name, "Expected iptables at abcd1234")
	assert.Equal(t, "iptables", task.Module.Params["key"], "Expected key to be abcd1234")

	task = plan.Tasks[1]
	assert.Equal(t, "Task 2 is valid", task.Name, "Task Name should have rendered properly")
	assert.Equal(t, "touch", task.Module.Params["cmd"], "Module param should have rendered properly")
	assert.Equal(t, "True", task.When, "When should be true in string form")

	proceed, err := task.ProcessWhen()
	require.NoError(t, err, "When should evaluate properly")
	assert.Equal(t, true, proceed, "When should evaluate to true")

	task = plan.Tasks[2]
	assert.Equal(t, "iTask 1", task.Name, "Task Name should have rendered properly")
	assert.Equal(t, "True", task.When, "When should evaluate properly")

	task = plan.Tasks[3]
	assert.Equal(t, "iTask 2", task.Name, "Task Name should have rendered properly")
	assert.Equal(t, "False", task.When, "When should evaluate properly")

	// tests for ProcessWhen()
	task = plan.Tasks[4]
	proceed, err = task.ProcessWhen()
	require.Error(t, err, "When should only have a true or false string")
	assert.Equal(t, false, proceed, "When should evaluate to false")

	task = plan.Tasks[5]
	proceed, err = task.ProcessWhen()
	require.NoError(t, err, "When should only have a true or false string")
	assert.Equal(t, false, proceed, "When should evaluate to false")
}

//func TestingInvalidRendering(t *testing.T) {
//	buf, err := ioutil.ReadFile("test/plan/invalid/invalidPongo2.yaml")
//	require.NoError(t, err, "Could not read invalidPongo2.yaml")
//	//	Test is no longer valid after latest refactoring
//	//	plan, err := PreprocessPlan(buf, nil)
//	//	require.NoError(t, err, "This plan shouldn't be having an error")
//
//	regMap := make(RegMap)
//	plan.Tasks[0].Render(regMap)
//	require.Error(t, err, "task.Name should not render properly, nested {{ }} are not allowed")
//
//	plan.Tasks[1].Render(regMap)
//	require.Error(t, err, "task.Name should not render properly, every \"{{\" needs a closing \"}}\"")
//
//	plan.Tasks[2].Render(regMap)
//	require.NoError(t, err, "Having }} in a variable is legal")
//}
