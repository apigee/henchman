package henchman

import (
	"errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
)

type PlanProxy struct {
	Plan        Plan
	TaskProxies []*TaskProxy `yaml:"tasks"`
	VarsProxy   TaskVars     `yaml:"vars"`
}

// Task is for the general Task format.  Refer to task.go
// Vars are kept in scope for each Task.  So there is a Vars
// field for each task
// Include is the file name for the included Tasks list
type TaskProxy struct {
	Task    `yaml:",inline"`
	Include string
}

// source values will override dest values override is true
// else dest values will not be overridden
func mergeMap(src TaskVars, dst TaskVars, override bool) {
	for variable, value := range src {
		if override == true {
			dst[variable] = value
		} else if _, present := dst[variable]; !present {
			dst[variable] = value
		}
	}
}

func (tp *TaskProxy) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var tmap map[string]interface{}
	err := unmarshal(&tmap)
	if err != nil {
		return err
	}
	for field, val := range tmap {
		// FIXME: Also do a type assertion later on.
		switch field {
		case "name":
			tp.Name = val.(string)
		case "ignore_errors":
			tp.IgnoreErrors = val.(bool)
		case "local":
			tp.Local = val.(bool)
		case "when":
			tp.When = val.(string)
		case "register":
			tp.Register = val.(string)
		case "include":
			tp.Include = val.(string)
		case "vars":
			tp.Vars = val.(map[interface{}]interface{})
		default:
			// We have a module
			module, err := NewModule(field, val.(string))
			if err != nil {
				return err
			}
			tp.Module = module
		}
	}
	return nil
}

// Checks the a slice of TaskProxy ptrs passed in by a Plan and determines
// whether if it's an include value or a normal task.  If it's a normal task
// it appends it as a standard task, otherwise it recursively expands the include
// statement
func PreprocessTasks(taskSection []*TaskProxy, planVars TaskVars) ([]*Task, error) {
	return parseTaskProxies(taskSection, planVars)
}

func preprocessTasksHelper(buf []byte, prevVars TaskVars) ([]*Task, error) {
	var px PlanProxy
	err := yaml.Unmarshal(buf, &px)
	if err != nil {
		return nil, err
	}

	return parseTaskProxies(px.TaskProxies, prevVars)
}

func parseTaskProxies(taskProxies []*TaskProxy, prevVars TaskVars) ([]*Task, error) {
	var tasks []*Task
	for _, tp := range taskProxies {
		task := Task{}
		if tp.Include != "" {
			// FIXME: resolve if templating is found
			// things need to be rendered when it's done
			buf, err := ioutil.ReadFile(tp.Include)
			if err != nil {
				return nil, err
			}

			if tp.Vars == nil {
				tp.Vars = make(TaskVars)
			}
			mergeMap(prevVars, tp.Vars, false)

			includedTasks, err := preprocessTasksHelper(buf, tp.Vars)
			if err != nil {
				return nil, err
			}

			tasks = append(tasks, includedTasks...)
		} else {
			if tp.Module == nil {
				return nil, errors.New("This task doesn't have a valid module")
			}
			task.Name = tp.Name
			task.Module = tp.Module
			task.IgnoreErrors = tp.IgnoreErrors
			task.Local = tp.Local
			task.Register = tp.Register
			task.When = tp.When
			task.Vars = prevVars

			tasks = append(tasks, &task)
		}
	}

	return tasks, nil
}

// Processes plan level vars with includes
// All plan level vars will be in the vars map
// And any repeat vars in the includes will be a FCFS priority
func PreprocessVars(vars TaskVars) (TaskVars, error) {
	newVars := vars
	if fileList, present := vars["include"]; present {
		for _, fName := range fileList.([]interface{}) {
			tempVars, err := preprocessVarsHelper(fName)
			if err != nil {
				return nil, err
			}
			mergeMap(tempVars, newVars, false)
		}
	}

	delete(newVars, "include")
	return newVars, nil
}

func preprocessVarsHelper(fName interface{}) (TaskVars, error) {
	newFName, ok := fName.(string)
	if !ok {
		log.Println("In an include in vars is not a valid string")
	}

	buf, err := ioutil.ReadFile(newFName)
	if err != nil {
		return nil, err
	}

	var px PlanProxy
	err = yaml.Unmarshal(buf, &px)
	if err != nil {
		return nil, err
	}

	return px.VarsProxy, nil
}

// For Plan
func PreprocessPlan(buf []byte) (*Plan, error) {
	var px PlanProxy
	err := yaml.Unmarshal(buf, &px)
	if err != nil {
		return nil, err
	}

	plan := Plan{}

	plan.Name = px.Plan.Name
	plan.Hosts = px.Plan.Hosts

	vars := make(TaskVars)
	if px.VarsProxy != nil {
		vars, err = PreprocessVars(px.VarsProxy)
		if err != nil {
			log.Printf("Error processing vars\n")
			return nil, err
		}
	}
	plan.Vars = vars

	tasks, err := PreprocessTasks(px.TaskProxies, plan.Vars)
	if err != nil {
		log.Printf("Error processing tasks\n")
		return nil, err
	}
	plan.Tasks = tasks

	return &plan, err
}
