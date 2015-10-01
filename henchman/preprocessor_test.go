package henchman

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func writeTempFile(buf []byte, fname string) string {
	fpath := path.Join("/tmp", fname)
	ioutil.WriteFile(fpath, buf, 0644)
	return fpath
}

func rmTempFile(fpath string) {
	os.Remove(fpath)
}

func TestPreprocessPlanValid(t *testing.T) {
	inv, _ := loadValidInventory()
	buf, err := ioutil.ReadFile("test/validPlan.yaml")
	if err != nil {
		t.Errorf("Could not read validPlan.yaml")
	}
	plan, err := PreprocessPlan(buf, inv)
	if err != nil {
		t.Fatalf("This plan couldn't be processed - %s\n", err.Error())
	}
	if len(plan.Tasks) != 2 {
		t.Errorf("Expected 2 tasks. Found %d tasks instead\n", len(plan.Tasks))
	}
}

func TestPreprocessPlanValidWithHosts(t *testing.T) {
	inv, _ := loadValidInventory()
	buf, err := ioutil.ReadFile("test/validPlanWithHosts.yaml")
	if err != nil {
		t.Errorf("Could not read validPlanWithHosts.yaml")
	}
	plan, err := PreprocessPlan(buf, inv)
	if err != nil {
		t.Fatalf("This plan couldn't be processed - %s\n", err.Error())
	}
	if len(plan.Tasks) != 4 {
		t.Errorf("Expected 4 tasks. Found %d tasks instead\n", len(plan.Tasks))
	}
	// NOTE: The inner hosts are ignored and the top level is taken
	if plan.Inventory.Count() != 2 {
		t.Errorf("Expected 2 machines. Got %d instead\n", plan.Inventory.Count())
	}
}

func TestPreprocessIncludeTasks(t *testing.T) {
	inv, _ := loadValidInventory()
	buf, err := ioutil.ReadFile("test/planWithIncludes.yaml")
	if err != nil {
		t.Errorf("Could not read planWithIncludes.yaml")
	}

	plan, err := PreprocessPlan(buf, inv)
	if err != nil {
		t.Fatalf("This plan shouldn't be having an error - %s\n", err.Error())
	}
	if len(plan.Tasks) != 3 {
		t.Fatalf("Expected 3 tasks. Found %d instead\n", len(plan.Tasks))
	}
	task1 := plan.Tasks[0].Name
	task2 := plan.Tasks[1].Name
	if task1 != "task1" {
		t.Errorf("Task name should have been task1. Got %s\n", task1)
	}
	if task2 != "included_task1" {
		t.Errorf("Task name should have been included_task1. Got %s\n", task2)
	}
}

func TestPreprocessIncludeTasksWithHosts(t *testing.T) {
	inv, _ := loadValidInventory()
	buf, err := ioutil.ReadFile("test/planWithIncludes.yaml")
	if err != nil {
		t.Errorf("Could not read planWithIncludes.yaml")
	}

	plan, err := PreprocessPlan(buf, inv)
	if err != nil {
		t.Fatalf("This plan shouldn't be having an error - %s\n", err.Error())
	}
	if len(plan.Tasks) != 3 {
		t.Fatalf("Expected 3 tasks. Found %d instead\n", len(plan.Tasks))
	}
	task1 := plan.Tasks[0].Name
	task2 := plan.Tasks[1].Name
	if task1 != "task1" {
		t.Errorf("Task name should have been task1. Got %s\n", task1)
	}
	if task2 != "included_task1" {
		t.Errorf("Task name should have been included_task1. Got %s\n", task2)
	}
}

func TestPreprocessNestedIncludeTasks(t *testing.T) {
	inv, _ := loadValidInventory()
	buf, err := ioutil.ReadFile("test/planWithNestedIncludes.yaml")
	if err != nil {
		t.Errorf("Could not read planWithNestedIncludes.yaml")
	}
	plan, err := PreprocessPlan(buf, inv)
	if err != nil {
		t.Fatalf("This plan shouldn't be having an error - %s\n", err.Error())
	}
	if len(plan.Tasks) != 4 {
		t.Fatalf("Expected 4 tasks. Found %d instead\n", len(plan.Tasks))
	}

	task1 := plan.Tasks[0].Name
	task2 := plan.Tasks[1].Name
	if task1 != "task1" {
		t.Errorf("Task name should have been task1. Got %s\n", task1)
	}
	if task2 != "included_task1" {
		t.Errorf("Task name should have been included_task1. Got %s\n", task2)
	}
}

func TestPreprocessIncludeTasksWithVars(t *testing.T) {
	inv, _ := loadValidInventory()
	buf, err := ioutil.ReadFile("test/planWithTasksAndVars.yaml")
	if err != nil {
		t.Errorf("Could not read planWithTasksAndVars.yaml")
	}
	plan, err := PreprocessPlan(buf, inv)
	if err != nil {
		t.Fatalf("This plan shouldn't be having an error - %s\n", err.Error())
	}
	if len(plan.Tasks) != 6 {
		t.Fatalf("Expected 5 tasks. Found %d instead\n", len(plan.Tasks))
	}
	if plan.Tasks[0].Vars["foo"] != "bar" {
		t.Fatalf("Expected bar. Found %v instead\n", plan.Tasks[0].Vars["foo"])
	}
	if plan.Tasks[1].Vars["foo"] != "nope" {
		t.Fatalf("Expected nope. Found %v instead\n", plan.Tasks[1].Vars["foo"])
	}
	if plan.Tasks[2].Vars["foo"] != "thumb" {
		t.Fatalf("Expected thumb. Found %v instead\n", plan.Tasks[2].Vars["foo"])
	}
	if plan.Tasks[4].Vars["foo"] != "nope" {
		t.Fatalf("Expected nope. Found %v instead\n", plan.Tasks[3].Vars["foo"])
	}
	if plan.Tasks[5].Vars["foo"] != "bar" {
		t.Fatalf("Expected bar. Found %v instead\n", plan.Tasks[4].Vars["foo"])
	}
}

func TestPreprocessVarsWithIncludeNoOverride(t *testing.T) {
	inv, _ := loadValidInventory()
	buf, err := ioutil.ReadFile("test/planWithIncludesInVars.yaml")
	if err != nil {
		t.Errorf("Could not read planWithIncludesInVars.yaml")
	}

	plan, err := PreprocessPlan(buf, inv)
	if err != nil {
		t.Fatalf("This plan couldn't be processed - %s\n", err.Error())
	}
	if len(plan.Tasks) != 2 {
		t.Errorf("Expected 2 tasks. Found %d tasks instead\n", len(plan.Tasks))
	}

	if len(plan.Vars) != 5 {
		t.Errorf("Expected 5 vars.  Found %d vars instead\n", len(plan.Vars))
	}
	for key, val := range plan.Vars {
		switch key {
		case "fun":
			if val.(string) != "times" {
				t.Fatalf("For key fun, expected \"times\".  Received %v\n", val)
			}
		case "hello":
			if val.(string) != "world" {
				t.Fatalf("For key hello, expected \"world\".  Received %v\n", val)
			}
		case "foo":
			if val.(string) != "scar" {
				t.Fatalf("For key foo, expected \"scar\".  Received %v\n", val)
			}
		case "spam":
			if val.(string) != "eggs" {
				t.Fatalf("For key spam, expected \"eggs\".  Received %v\n", val)
			}
		case "goodbye":
			if val.(string) != "moon" {
				t.Fatalf("For key goodbye, expected \"times\".  Received %v\n", val)
			}
		}
	}
}

func TestPreprocessTasksWithIncludesAndWhen(t *testing.T) {
	inv, _ := loadValidInventory()
	buf, err := ioutil.ReadFile("test/planWithTaskIncludesAndWhen.yaml")
	if err != nil {
		t.Errorf("Could not read planWithHosts.yaml")
	}

	plan, err := PreprocessPlan(buf, inv)
	if plan.Tasks[0].When != "test == true" {
		t.Fatalf("Expected \"test == true\".  Received \"%v\"\n", plan.Tasks[0].When)
	}
	if plan.Tasks[1].When != "hello == world && test == false" {
		t.Fatalf("Expected \"hello == world && test == false\".  Received \"%v\"\n", plan.Tasks[1].When)
	}
	if plan.Tasks[2].When != "jolly == santa && goodbye == moon && test == false" {
		t.Fatalf("Expected \"jolly == santa && goodbye == moon && test == false\".  Received \"%v\"\n", plan.Tasks[2].When)
	}
	if plan.Tasks[3].When != "goodbye == moon && test == false" {
		t.Fatalf("Expected \"goodbye == moon && test == false\".  Received \"%v\"\n", plan.Tasks[3].When)
	}
}

func TestPreprocessWithSudoAtThePlanLevel(t *testing.T) {
	plan_string := `---
name: "Sample plan"
sudo: true
hosts:
  - "127.0.0.1:22"
  - 192.168.1.2
tasks:
  - name: Sample task that does nothing
    action: cmd="ls"
  - name: Another task
    action: cmd="test"
`
	plan, err := PreprocessPlan([]byte(plan_string), nil)
	if err != nil {
		t.Fatalf("This plan couldn't be processed - %s\n", err.Error())
	}
	if len(plan.Tasks) != 2 {
		t.Errorf("Expected 2 tasks. Found %d tasks instead\n", len(plan.Tasks))
	}
	for _, task := range plan.Tasks {
		if !task.Sudo {
			t.Errorf("This task should have sudo privilege")
		}
	}
}

func TestPreprocessWithSudoAtTheTaskLevel(t *testing.T) {
	inv, _ := loadValidInventory()
	plan_string := `---
name: "Sample plan"
tasks:
  - name: First task
    action: cmd="ls"
    sudo: true
  - name: Second task
    action: cmd="echo"
`
	plan, err := PreprocessPlan([]byte(plan_string), inv)
	if err != nil {
		t.Fatalf("This plan couldn't be processed - %s\n", err.Error())
	}
	if len(plan.Tasks) != 2 {
		t.Errorf("Expected 2 tasks. Found %d tasks instead\n", len(plan.Tasks))
	}
	for _, task := range plan.Tasks {
		if task.Name == "First task" && !task.Sudo {
			t.Errorf("The task %s should have sudo privilege", task.Name)

		}
		if task.Name == "Second task" && task.Sudo {
			t.Errorf("The task %s should have sudo privilege", task.Name)

		}
	}
}

func TestPreprocessWithSudoInTheIncludeTask(t *testing.T) {
	inv, _ := loadValidInventory()
	include_file := `
name: "To be include"
tasks:
    - name: "included_task1"
      action: bar=baz
      sudo: true
    - name: "included_task2"
      action: foo=bar
`
	plan_file := `
name: "Sample plan"
tasks:
  - name: task1
    action: cmd="ls -al"
  - include: /tmp/included.yaml
`
	fpath := writeTempFile([]byte(include_file), "included.yaml")
	defer rmTempFile(fpath)
	plan, err := PreprocessPlan([]byte(plan_file), inv)
	if err != nil {
		t.Fatalf("This plan shouldn't be having an error - %s\n", err.Error())
	}
	if len(plan.Tasks) != 3 {
		t.Fatalf("Expected 3 tasks. Found %d instead\n", len(plan.Tasks))
	}
	for _, task := range plan.Tasks {
		if task.Name == "included_task1" && !task.Sudo {
			t.Errorf("Expected nested_task1 task to have Sudo privileges")
		}
		if task.Name != "included_task1" && task.Sudo {
			t.Errorf("Expected task %s to not have Sudo privileges", task.Name)
		}
	}
}

/*
func TestWithErrors(t *testing.T) {
	plan_file :=
		`
---
name: "Fail plan"
tasks:
  - name: "Bad task"
    shell: okay=wut
    mod: wut=mutt
	- name: "Task Bad Indent"
	  shell: okay=wutt
`
	_, err := PreprocessPlan([]byte(plan_file), nil)
	if err != nil {
		t.Errorf(err.Error())
	}
}
*/
