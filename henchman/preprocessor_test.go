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

/*
func TestPreprocessPlanValid(t *testing.T) {
	plan_string := `---
name: "Sample plan"
hosts:
  - "127.0.0.1:22"
  - 192.168.1.2
tasks:
  - name: Sample task that does nothing
    action: cmd="ls"
  - name: Second task
    action: cmd="echo"
    ignore_errors: true
`
	plan, err := PreprocessPlan([]byte(plan_string))
	if err != nil {
		t.Fatalf("This plan couldn't be processed - %s\n", err.Error())
	}
	if len(plan.Tasks) != 2 {
		t.Errorf("Expected 2 tasks. Found %d tasks instead\n", len(plan.Tasks))
	}
}

func TestPreprocessIncludeTasks(t *testing.T) {
	include_file := `
name: "To be include"
tasks:
    - name: "included_task1"
      action: bar=baz
    - name: "included_task2"
      action: foo=bar
`
	plan_file := `
name: "Sample plan"
hosts:
  - "127.0.0.1:22"
  - 192.168.1.2
tasks:
  - name: task1
    action: cmd="ls-al"
  - include: /tmp/included.yaml
`
	fpath := writeTempFile([]byte(include_file), "included.yaml")
	defer rmTempFile(fpath)
	plan, err := PreprocessPlan([]byte(plan_file))
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
	nested_include_file := `
name: "Nested Included"
tasks:
    - name: "nested_task1"
      yum: "pkg=bar"
    - name: "nested_task2"
      action: foo=baz
`
	include_file := `
name: "Included"
tasks:
    - name: "included_task1"
      shell: cmd=foo user=root
    - include: /tmp/nested.yaml
`
	plan_file := `
name: "Sample plan"
hosts:
  - "127.0.0.1:22"
  - 192.168.1.2
tasks:
  - name: task1
    action: cmd=ls user=foo
  - include: /tmp/included.yaml
`
	fpath := writeTempFile([]byte(include_file), "included.yaml")
	nested_path := writeTempFile([]byte(nested_include_file), "nested.yaml")
	defer rmTempFile(fpath)
	defer rmTempFile(nested_path)
	plan, err := PreprocessPlan([]byte(plan_file))
	if err != nil {
		t.Fatalf("This plan shouldn't be having an error - %s\n", err.Error())
	}
	if len(plan.Tasks) != 4 {
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

func TestPreprocessIncludeTasksWithVars(t *testing.T) {
	nested_include_file := `
name: "Nested Included"
tasks:
    - name: "nested_task1"
      action: monkey="bar"
`
	include_file := `
name: "To be include"
tasks:
    - name: "included_task1"
      action: foo="bar"
    - include: /tmp/nested.yaml
      vars:
        foo: thumb
    - name: "included_task2"
      action: spiz="spaz"
`
	plan_file := `
name: "Sample plan"
vars:
  foo: bar
hosts:
  - "127.0.0.1:22"
  - 192.168.1.2
tasks:
  - name: task1
    action: ls=al
  - include: /tmp/included.yaml
    vars:
      foo: nope
      bar: baz
  - name: task2
    action: hey="yoooo"
`
	fpath := writeTempFile([]byte(include_file), "included.yaml")
	nested_path := writeTempFile([]byte(nested_include_file), "nested.yaml")
	defer rmTempFile(fpath)
	defer rmTempFile(nested_path)
	plan, err := PreprocessPlan([]byte(plan_file))
	if err != nil {
		t.Fatalf("This plan shouldn't be having an error - %s\n", err.Error())
	}
	if len(plan.Tasks) != 5 {
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
	if plan.Tasks[3].Vars["foo"] != "nope" {
		t.Fatalf("Expected nope. Found %v instead\n", plan.Tasks[3].Vars["foo"])
	}
	if plan.Tasks[4].Vars["foo"] != "bar" {
		t.Fatalf("Expected bar. Found %v instead\n", plan.Tasks[4].Vars["foo"])
	}
}
*/
func TestPreprocessVarsWithIncludeNoOverride(t *testing.T) {
	vars_file := `---
name: "Included Vars"
vars:
  goodbye: moon
  foo: bar
`
	vars2_file := `---
name: "Included Vars 2"
vars:
  goodbye: guacamole
  fun: times
`
	plan_string := `---
name: "Sample plan"
hosts:
  - "127.0.0.1:22"
  - 192.168.1.2
vars:
  hello: world
  foo: scar
  include:
   - /tmp/vars.yaml
   - /tmp/vars2.yaml
  spam: eggs
tasks:
  - name: Sample task that does nothing
    action: cmd="ls"
  - name: Second task
    action: cmd="echo"
    ignore_errors: true
`
	fpath := writeTempFile([]byte(vars_file), "vars.yaml")
	fpath2 := writeTempFile([]byte(vars2_file), "vars2.yaml")
	defer rmTempFile(fpath)
	defer rmTempFile(fpath2)
	plan, err := PreprocessPlan([]byte(plan_string))
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
