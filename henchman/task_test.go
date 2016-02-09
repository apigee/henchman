package henchman

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskRunForSingleMachine(t *testing.T) {
	origSearchPath := ModuleSearchPath
	modDir := createTempDir("henchman")
	ModuleSearchPath = append(ModuleSearchPath, modDir)
	defer func() {
		ModuleSearchPath = origSearchPath
	}()
	defer os.RemoveAll(modDir)

	shellPath := path.Join(modDir, "shell")
	err := os.Mkdir(shellPath, 0755)
	require.NoError(t, err)

	err = ioutil.WriteFile(path.Join(shellPath, "shell"), []byte("ls -al"), 0644)
	mod, err := NewModule("shell", "foo=bar")

	task := Task{}
	task.Name = "test"
	task.Module = mod

	testTransport := TestTransport{}

	localhost := Machine{}
	localhost.Hostname = "localhost"
	localhost.Transport = &testTransport

	regMap := make(RegMap)

	_, err = task.Run(&localhost, task.Vars, regMap)
	require.NoError(t, err, "There shouldn't have been any errors")
}

func TestTaskRenderAndProcessWhen(t *testing.T) {
	buf, err := ioutil.ReadFile("test/plan/planWithPongo2.yaml")
	require.NoError(t, err, "Could not read planWithPongo2.yaml")

	plan, err := PreprocessPlan(buf, &Inventory{})
	require.NoError(t, err, "This plan shouldn't be having an error")

	testTransport := TestTransport{}
	localhost := Machine{}
	localhost.Hostname = "localhost"
	localhost.Transport = &testTransport

	regMap := make(RegMap)
	regMap["cmd"] = "touch"
	regMap["name"] = "Task 2"

	renderedTasks := []*Task{}
	for _, task := range plan.Tasks {
		MergeMap(plan.Vars, task.Vars, false)
		renderedTask, err := task.Render(task.Vars, regMap)
		require.NoError(t, err)
		renderedTasks = append(renderedTasks, renderedTask)
	}

	task := renderedTasks[0]
	assert.Equal(t, "iptables with abcd1234", task.Name, "Expected iptables at abcd1234")
	assert.Equal(t, "iptables", task.Module.Params["key"], "Expected key to be abcd1234")

	task = renderedTasks[1]
	assert.Equal(t, "Task 2 is valid", task.Name, "Task Name should have rendered properly")
	assert.Equal(t, "touch", task.Module.Params["cmd"], "Module param should have rendered properly")
	assert.Equal(t, "True", task.When, "When should be true in string form")

	proceed, err := task.ProcessWhen()
	require.NoError(t, err, "When should evaluate properly")
	assert.Equal(t, true, proceed, "When should evaluate to true")

	task = renderedTasks[2]
	assert.Equal(t, "iTask 1", task.Name, "Task Name should have rendered properly")
	assert.Equal(t, "True", task.When, "When should evaluate properly")

	task = renderedTasks[3]
	assert.Equal(t, "iTask 2", task.Name, "Task Name should have rendered properly")
	assert.Equal(t, "False", task.When, "When should evaluate properly")

	// tests for ProcessWhen()
	task = renderedTasks[4]
	proceed, err = task.ProcessWhen()
	require.Error(t, err, "When should only have a true or false string")
	assert.Equal(t, false, proceed, "When should evaluate to false")

	task = renderedTasks[5]
	proceed, err = task.ProcessWhen()
	require.NoError(t, err, "When should only have a true or false string")
	assert.Equal(t, false, proceed, "When should evaluate to false")
}

func TestProccessWithItems(t *testing.T) {
	inv, _ := loadValidInventory()
	buf, err := ioutil.ReadFile("test/plan/withItemsAtTaskLevelWithHenchmanitem.yaml")
	require.NoError(t, err)

	invGroups, err := GetInventoryGroups(buf)
	inventory := inv.GetInventoryForGroups(invGroups)
	inventory.SetGlobalVarsFromInventoryGroups(inv.Groups)
	assert.Equal(t, []string{"localhost"}, invGroups, "inventory Groups invGroups do not match expected output")

	plan, err := PreprocessPlan(buf, &inventory)
	require.NoError(t, err)
	subTasks, err := plan.Tasks[0].ProcessWithItems(make(VarsMap), make(RegMap))
	require.NoError(t, err)
	assert.Equal(t, "Task 1 test1", subTasks[0].Name)
	assert.Equal(t, "Task 1 test2", subTasks[1].Name)
	assert.Equal(t, "Task 1 test3", subTasks[2].Name)
}

/*
func TestSandboxRendering(t *testing.T) {
	inv, _ := loadValidInventory()
	buf, err := ioutil.ReadFile("test/plan/sandboxPongo2.yaml")
	require.NoError(t, err, "Could not read sandboxPongo2.yaml")

	plan, err := PreprocessPlan(buf, inv)
	require.NoError(t, err)

	regMap := make(RegMap)
	err = plan.Tasks[0].Render(plan.Vars, regMap)
	require.NoError(t, err)
	fmt.Println(plan.Tasks[0].Name)
}
*/
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
