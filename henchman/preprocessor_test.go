package henchman

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreprocessInventoryAtHostLevel(t *testing.T) {
	inv, _ := loadValidInventory()
	buf, err := ioutil.ReadFile("test/plan/inventoryAtHostLevel.yaml")
	require.NoError(t, err)

	tc := make(TransportConfig)
	tc["hostname"] = "foo"
	tc["username"] = "foobar"
	tc["password"] = "bar"

	invGroups, err := GetInventoryGroups(buf)
	inventory := inv.GetInventoryForGroups(invGroups)
	plan, err := PreprocessPlan(buf, inventory)

	require.NoError(t, err)
	assert.Equal(t, "Sample plan", plan.Name, "plan name wasn't unmarshalled properly")
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

	assert.Equal(t, "Plan with single Include", plan.Name, "plan name wasn't unmarshalled properly")
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

	assert.Equal(t, "Sample plan", plan.Name, "plan name wasn't unmarshalled properly")
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

	assert.Equal(t, "Plan With Tasks and Vars", plan.Name, "plan name wasn't unmarshalled properly")
	assert.Equal(t, 6, len(plan.Tasks), "Wrong number of tasks.")
	assert.Empty(t, plan.Tasks[0].Vars, "Isn't part of included task, should not have variables binded")
	assert.Equal(t, "nope", plan.Tasks[1].Vars["foo"], "Wrong key in Task Vars")
	assert.Equal(t, "thumb", plan.Tasks[2].Vars["foo"], "Wrong key in Task Vars")
	assert.Equal(t, "baz", plan.Tasks[3].Vars["bar"], "Wrong key in Task Vars")
	assert.Equal(t, "nope", plan.Tasks[4].Vars["foo"], "Wrong key in Task Vars")
	assert.Empty(t, plan.Tasks[5].Vars, "Isn't part of included task, should not have variables binded")
}

func TestPreprocessIncludeAtVarsLevel(t *testing.T) {
	inv, _ := loadValidInventory()
	buf, err := ioutil.ReadFile("test/plan/includeAtVarsLevel.yaml")
	require.NoError(t, err)

	plan, err := PreprocessPlan(buf, inv)
	require.NoError(t, err)

	require.Equal(t, 5, len(plan.Vars), "Wrong number of vars.")

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
	assert.Equal(t, "Sample plan", plan.Name, "Plan name wasn't unmarshalled")
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
	assert.Equal(t, "Plan with Task Includes And When", plan.Name, "Plan name wasn't unmarshalled")
}

func TestPreprocessWithSudoAtThePlanLevel(t *testing.T) {
	inv, _ := loadValidInventory()
	buf, err := ioutil.ReadFile("test/plan/sudoAtPlanLevel.yaml")
	require.NoError(t, err)

	plan, err := PreprocessPlan(buf, inv)
	require.NoError(t, err)

	assert.Equal(t, 2, len(plan.Tasks), "Wrong number of tasks.")
	assert.Equal(t, "Sample plan", plan.Name, "Plan name wasn't unmarshalled")
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
	assert.Equal(t, "Sample plan", plan.Name, "Plan name wasn't unmarshalled")
	for _, task := range plan.Tasks {
		if task.Name == "First task" {
			assert.True(t, task.Sudo, "First task should have sudo priviledges")
		}

		if task.Name == "Second task" {
			assert.False(t, task.Sudo, "Second task should not have sudo priviledges")
		}
	}
}

func TestPreprocessWithSudoOverrideAtTheTaskLevel(t *testing.T) {
	inv, _ := loadValidInventory()
	buf, err := ioutil.ReadFile("test/plan/sudoOverrideAtTaskLevel.yaml")
	require.NoError(t, err)

	plan, err := PreprocessPlan(buf, inv)
	require.NoError(t, err)

	assert.Equal(t, 2, len(plan.Tasks), "Wrong number of tasks.")
	assert.Equal(t, "Sample plan", plan.Name, "Plan name wasn't unmarshalled")
	for _, task := range plan.Tasks {
		if task.Name == "First task" {
			assert.True(t, task.Sudo, "First task should have sudo priviledges")
		}

		if task.Name == "Second task" {
			assert.False(t, task.Sudo, "Second task should not have sudo priviledges")
		}
	}
}

func TestPreprocessWithIgnoreErrorsAtTheTaskLevel(t *testing.T) {
	inv, _ := loadValidInventory()
	buf, err := ioutil.ReadFile("test/plan/ignoreErrsAtTaskLevel.yaml")
	require.NoError(t, err)

	plan, err := PreprocessPlan(buf, inv)
	require.NoError(t, err)

	assert.Equal(t, 2, len(plan.Tasks), "Wrong number of tasks.")
	assert.Equal(t, "Sample plan", plan.Name, "Plan name wasn't unmarshalled")
	for _, task := range plan.Tasks {
		if task.Name == "First task" {
			assert.True(t, task.IgnoreErrors, "First task should ignore errors")
		}

		if task.Name == "Second task" {
			assert.False(t, task.IgnoreErrors, "Second task should not ignore errors")
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
	assert.Equal(t, "Sample plan", plan.Name, "Plan name wasn't unmarshalled")
	for _, task := range plan.Tasks {
		if task.Name == "included_task1" {
			assert.True(t, task.Sudo, "First task should have sudo priviledges")
		}

		if task.Name == "included_task2" {
			assert.False(t, task.Sudo, "Second task should not have sudo priviledges")
		}
	}
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

// Table driven test for Invalids
func TestInvalid(t *testing.T) {
	var tests = []struct {
		fName string
	}{
		{"test/plan/invalid/invalidIncludeFormatAtVarsLevel.yaml"},
		{"test/plan/invalid/invalidRegisterKeyword.yaml"},
		{"test/plan/invalid/invalidRegisterVariable.yaml"},
		{"test/plan/invalid/invalidPongo2AtWhen.yaml"},
	}
	inv, _ := loadValidInventory()
	for _, test := range tests {
		buf, err := ioutil.ReadFile(test.fName)
		require.NoError(t, err)

		_, err = PreprocessPlan(buf, inv)
		require.Error(t, err, fmt.Sprintf("Expected error in %v", test.fName))
	}
}
