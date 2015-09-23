package henchman

import (
	"gopkg.in/yaml.v2"
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
	buf, err := ioutil.ReadFile("test/validPlan.yaml")
	if err != nil {
		t.Errorf("Could not read validPlan.yaml")
	}

	plan, err := PreprocessPlan(buf, nil)
	if err != nil {
		t.Fatalf("This plan couldn't be processed - %s\n", err.Error())
	}
	if len(plan.Tasks) != 2 {
		t.Errorf("Expected 2 tasks. Found %d tasks instead\n", len(plan.Tasks))
	}
}

func TestPreprocessIncludeTasks(t *testing.T) {
	buf, err := ioutil.ReadFile("test/planWithIncludes.yaml")
	if err != nil {
		t.Errorf("Could not read planWithIncludes.yaml")
	}

	plan, err := PreprocessPlan(buf, nil)
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
	buf, err := ioutil.ReadFile("test/planWithNestedIncludes.yaml")
	if err != nil {
		t.Errorf("Could not read planWithNestedIncludes.yaml")
	}

	plan, err := PreprocessPlan(buf, nil)
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
	buf, err := ioutil.ReadFile("test/planWithTasksAndVars.yaml")
	if err != nil {
		t.Errorf("Could not read planWithTasksAndVars.yaml")
	}

	plan, err := PreprocessPlan(buf, nil)
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
	buf, err := ioutil.ReadFile("test/planWithIncludesInVars.yaml")
	if err != nil {
		t.Errorf("Could not read planWithIncludesInVars.yaml")
	}

	plan, err := PreprocessPlan(buf, nil)
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

// NOTE: This assumes hosts files are in a YAML format
//       This will change.
func TestPreprocessHosts(t *testing.T) {
	buf, err := ioutil.ReadFile("test/planWithHosts.yaml")
	if err != nil {
		t.Errorf("Could not read planWithHosts.yaml")
	}

	inv := make(Inventory)
	invBuf, err := ioutil.ReadFile("test/hosts")
	if err != nil {
		t.Errorf("Could not read hosts")
	}

	err = yaml.Unmarshal(invBuf, &inv)
	if err != nil {
		t.Errorf("Error unmarshalling hosts")
	}

	plan, err := PreprocessPlan(buf, inv)
	if err != nil {
		t.Fatalf("This plan couldn't be processed - %s\n", err.Error())
	}

	if len(plan.Hosts) != 6 {
		t.Fatalf("Expected 6 hosts.  Found %v\n", len(plan.Hosts))
	}
	if plan.Hosts[0] != "127.0.0.1" {
		t.Fatalf("Expected 127.0.0.1. Found %v\n", plan.Hosts[0])
	}
	if plan.Hosts[1] != "127.0.0.2" {
		t.Fatalf("Expected 127.0.0.2. Found %v\n", plan.Hosts[1])
	}
	if plan.Hosts[2] != "123.456.789" {
		t.Fatalf("Expected 123.456.789. Found %v\n", plan.Hosts[2])
	}
	if plan.Hosts[3] != "000.000.000" {
		t.Fatalf("Expected 000.000.000. Found %v\n", plan.Hosts[3])
	}
	if plan.Hosts[4] != "127.0.0.3" {
		t.Fatalf("Expected 127.0.0.3. Found %v\n", plan.Hosts[4])
	}
	if plan.Hosts[5] != "127.0.0.4" {
		t.Fatalf("Expected 127.0.0.4. Found %v\n", plan.Hosts[5])
	}
}

func TestPreprocessTasksWithIncludesAndWhen(t *testing.T) {
	buf, err := ioutil.ReadFile("test/planWithTaskIncludesAndWhen.yaml")
	if err != nil {
		t.Errorf("Could not read planWithHosts.yaml")
	}

	plan, err := PreprocessPlan(buf, nil)
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
