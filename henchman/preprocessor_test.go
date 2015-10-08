package henchman

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTempFile(buf []byte, fname string) string {
	fpath := path.Join("/tmp", fname)
	ioutil.WriteFile(fpath, buf, 0644)
	return fpath
}

func rmTempFile(fpath string) {
	os.Remove(fpath)
}

func TestPreprocessInventoryAtHostLevel(t *testing.T) {
	inv, _ := loadValidInventory()
	buf, err := ioutil.ReadFile("test/plan/inventoryAtHostLevel.yaml")
	require.NoError(t, err)

	plan, err := PreprocessPlan(buf, inv)
	require.NoError(t, err)

	assert.Equal(t, 4, len(plan.Tasks), "Wrong number of tasks.")
	// NOTE: The inner hosts are ignored and the top level is taken
	assert.Equal(t, 2, plan.Inventory.Count(), "Wrong number of machines")
}

func TestPreprocessIncludeAtTaskLevel(t *testing.T) {
	inv, _ := loadValidInventory()
	buf, err := ioutil.ReadFile("test/plan/includeAtTaskLevel.yaml")
	require.NoError(t, err)

	plan, err := PreprocessPlan(buf, inv)
	require.NoError(t, err)

	assert.Equal(t, 3, len(plan.Tasks), "Wrong number of tasks.")
	assert.Equal(t, "task1", plan.Tasks[0].Name, "Wrong first task.")
	assert.Equal(t, "included_task1", plan.Tasks[1].Name, "Wrong second task.")
}

func TestPreprocessNestedIncludeAtTaskLevel(t *testing.T) {
	inv, _ := loadValidInventory()
	buf, err := ioutil.ReadFile("test/plan/nestedIncludeAtTaskLevel.yaml")
	require.NoError(t, err)

	plan, err := PreprocessPlan(buf, inv)
	require.NoError(t, err)

	assert.Equal(t, 4, len(plan.Tasks), "Wrong number of tasks.")
	assert.Equal(t, "task1", plan.Tasks[0].Name, "Wrong first task.")
	assert.Equal(t, "included_task1", plan.Tasks[2].Name, "Wrong second task.")
}

func TestPreprocessIncludeAndVarsAtTaskLevel(t *testing.T) {
	inv, _ := loadValidInventory()
	buf, err := ioutil.ReadFile("test/plan/includeAndVarsAtTaskLevel.yaml")
	require.NoError(t, err)

	plan, err := PreprocessPlan(buf, inv)
	require.NoError(t, err)

	assert.Equal(t, 6, len(plan.Tasks), "Wrong number of tasks.")
	assert.Equal(t, "bar", plan.Tasks[0].Vars["foo"], "Wrong key in Task Vars")
	assert.Equal(t, "nope", plan.Tasks[1].Vars["foo"], "Wrong key in Task Vars")
	assert.Equal(t, "thumb", plan.Tasks[2].Vars["foo"], "Wrong key in Task Vars")
	assert.Equal(t, "nope", plan.Tasks[4].Vars["foo"], "Wrong key in Task Vars")
	assert.Equal(t, "bar", plan.Tasks[5].Vars["foo"], "Wrong key in Task Vars")
}

func TestPreprocessIncludeAtVarsLevel(t *testing.T) {
	inv, _ := loadValidInventory()
	buf, err := ioutil.ReadFile("test/plan/includeAtVarsLevel.yaml")
	require.NoError(t, err)

	plan, err := PreprocessPlan(buf, inv)
	require.NoError(t, err)

	require.Equal(t, 6, len(plan.Vars), "Wrong number of vars.")

	for key, val := range plan.Vars {
		switch key {
		case "fun":
			assert.Equal(t, "times", val.(string), fmt.Sprintf("Wrong value for key %v", key))
		case "hello":
			assert.Equal(t, "world", val.(string), fmt.Sprintf("Wrong value for key %v", key))
		case "foo":
			assert.Equal(t, "scar", val.(string), fmt.Sprintf("Wrong value for key %v", key))
		case "spam":
			assert.Equal(t, "eggs", val.(string), fmt.Sprintf("Wrong value for key %v", key))
		case "goodbye":
			assert.Equal(t, "moon", val.(string), fmt.Sprintf("Wrong value for key %v", key))
		}
	}
}

func TestPreprocessIncludeAndWhenAtTaskLevel(t *testing.T) {
	inv, _ := loadValidInventory()
	buf, err := ioutil.ReadFile("test/plan/includeAndWhenAtTaskLevel.yaml")
	require.NoError(t, err)

	plan, err := PreprocessPlan(buf, inv)
	require.NoError(t, err)

	assert.Equal(t, "test == true", plan.Tasks[0].When, "task.When is wrong")
	assert.Equal(t, "hello == world && test == false", plan.Tasks[1].When, "task.When is wrong")
	assert.Equal(t, "jolly == santa && goodbye == moon && test == false", plan.Tasks[2].When, "task.When is wrong")
	assert.Equal(t, "goodbye == moon && test == false", plan.Tasks[3].When, "task.When is wrong")
}

func TestPreprocessWithSudoAtThePlanLevel(t *testing.T) {
	inv, _ := loadValidInventory()
	buf, err := ioutil.ReadFile("test/plan/sudoAtPlanLevel.yaml")
	require.NoError(t, err)

	plan, err := PreprocessPlan(buf, inv)
	require.NoError(t, err)

	assert.Equal(t, 2, len(plan.Tasks), "Wrong number of tasks.")

	for _, task := range plan.Tasks {
		assert.True(t, task.Sudo, "Sudo should be true")
	}
}

func TestPreprocessWithSudoAtTheTaskLevel(t *testing.T) {
	inv, _ := loadValidInventory()
	buf, err := ioutil.ReadFile("test/plan/sudoAtTaskLevel.yaml")
	require.NoError(t, err)

	plan, err := PreprocessPlan(buf, inv)
	require.NoError(t, err)

	assert.Equal(t, 2, len(plan.Tasks), "Wrong number of tasks.")

	for _, task := range plan.Tasks {
		if task.Name == "First task" {
			assert.True(t, task.Sudo, "First task should have sudo priviledges")
		}

		if task.Name == "Second task" {
			assert.False(t, task.Sudo, "Second task should not have sudo priviledges")
		}
	}
}

func TestPreprocessWithSudoInTheIncludeTask(t *testing.T) {
	inv, _ := loadValidInventory()
	buf, err := ioutil.ReadFile("test/plan/includeWithSudoAtTaskLevel.yaml")
	require.NoError(t, err)

	plan, err := PreprocessPlan(buf, inv)
	require.NoError(t, err)

	assert.Equal(t, 3, len(plan.Tasks), "Wrong number of tasks.")
	for _, task := range plan.Tasks {
		if task.Name == "included_task1" {
			assert.True(t, task.Sudo, "First task should have sudo priviledges")
		}

		if task.Name == "included_task2" {
			assert.False(t, task.Sudo, "Second task should not have sudo priviledges")
		}
	}
}

// create table driven tests for invalids
func TestInvalidIncludeFormatAtVarsLevel(t *testing.T) {
	inv, _ := loadValidInventory()
	buf, err := ioutil.ReadFile("test/plan/invalidIncludeFormatAtVarsLevel.yaml")
	require.NoError(t, err)

	_, err = PreprocessPlan(buf, inv)
	require.Error(t, err)
}

func TestPreprocessWithCommentsAtTheTaskLevelAndVarsLevel(t *testing.T) {
	inv, _ := loadValidInventory()
	buf, err := ioutil.ReadFile("test/plan/planWithComments.yaml")
	require.NoError(t, err)

	plan, err := PreprocessPlan(buf, inv)
	require.NoError(t, err)

	assert.Equal(t, 2, len(plan.Tasks), "Wrong number of tasks.")
	assert.Equal(t, "Second task", plan.Tasks[0].Name, "Task name is wrong. Expected task name was 'second task'")
	assert.Equal(t, "hello", plan.Vars["foo"], "Variable foo should have value hello")
	assert.Nil(t, plan.Vars["bar"], "Variable 'bar' should be commented")
}

func TestPreprocessWithInvalidTaskComments(t *testing.T) {
	inv, _ := loadValidInventory()
	buf, err := ioutil.ReadFile("test/plan/taskWithInvalidComments.yaml")
	require.NoError(t, err)

	_, err = PreprocessPlan(buf, inv)
	require.Error(t, err)

	buf, err = ioutil.ReadFile("test/plan/taskWithInvalidComments2.yaml")
	require.NoError(t, err)

	_, err = PreprocessPlan(buf, inv)
	require.Error(t, err)
}
