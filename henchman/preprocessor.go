package henchman

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strings"
)

type PlanProxy struct {
	Name            string       `yaml:"name"`
	Sudo            bool         `yaml:"sudo"`
	TaskProxies     []*TaskProxy `yaml:"tasks"`
	VarsProxy       *VarsProxy   `yaml:"vars"`
	InventoryGroups []string     `yaml:"hosts"`
}

// Task is for the general Task format.  Refer to task.go
// Vars are kept in scope for each Task.  So there is a Vars
// field for each task
// Include is the file name for the included Tasks list
type TaskProxy struct {
	Task    `yaml:",inline"`
	Include string
}

type VarsProxy struct {
	Vars VarsMap
}

// source values will override dest values override is true
// else dest values will not be overridden
func mergeMap(src map[interface{}]interface{}, dst map[interface{}]interface{}, override bool) {
	for variable, value := range src {
		if override == true {
			dst[variable] = value
		} else if _, present := dst[variable]; !present {
			dst[variable] = value
		}
	}
}

// Custom unmarshaller which account for multiple include statements and include types
// NOTE: Cannot account for double includes because unmarshal(&vMap) already does
//       under the hood unmarshaling and does what any map would do, which is override
//       repeating key values
func (vp *VarsProxy) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var vMap map[string]interface{}
	var found bool
	numInclude := 0

	err := unmarshal(&vMap)
	if err != nil {
		return err
	}

	vp.Vars = make(VarsMap)
	for field, val := range vMap {
		switch field {
		case "include":
			vp.Vars["include"], found = val.([]interface{})
			if !found {
				return ErrWrongType(field, val, "[]interface{}")
			}

			numInclude++
			if numInclude > 1 {
				return fmt.Errorf("Can only have one include statement at Vars level.")
			}
		default:
			vp.Vars[field] = val
		}
	}

	return nil
}

// Custom unmarshaller which accounts for module names
func (tp *TaskProxy) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var tmap map[string]interface{}
	var found bool
	numModule := 0

	err := unmarshal(&tmap)
	if err != nil {
		return err
	}

	for field, val := range tmap {
		switch field {
		case "name":
			tp.Name, found = val.(string)
			if !found {
				return ErrWrongType(field, val, "string")
			}
		case "sudo":
			tp.Sudo, found = val.(bool)
			if !found {
				return ErrWrongType(field, val, "bool")
			}
		case "ignore_errors":
			tp.IgnoreErrors, found = val.(bool)
			if !found {
				return ErrWrongType(field, val, "bool")
			}
		case "local":
			tp.Local, found = val.(bool)
			if !found {
				return ErrWrongType(field, val, "bool")
			}
		case "when":
			tp.When, found = val.(string)
			if !found {
				return ErrWrongType(field, val, "string")
			}
		case "register":
			tp.Register, found = val.(string)
			if !found {
				return ErrWrongType(field, val, "string")
			}
			if len(strings.Fields(tp.Register)) > 1 {
				return ErrNotValidVariable(tp.Register)
			}
			if isKeyword(tp.Register) {
				return ErrKeyword(tp.Register)
			}
		case "include":
			tp.Include, found = val.(string)
			if !found {
				return ErrWrongType(field, val, "string")
			}
		case "vars":
			tp.Vars, found = val.(map[interface{}]interface{})
			if !found {
				return ErrWrongType(field, val, "map[interface{}]interface{}")
			}
		default:
			// We have a module
			params, found := val.(string)
			if !found {
				return ErrWrongType(field, val, "string")
			}

			if numModule > 0 {
				return fmt.Errorf("\"%v\" is an extra Module.  Can only have one module per task.", field)
			}

			tp.Module, err = NewModule(field, params)
			if err != nil {
				return fmt.Errorf("Module %v: %s", field, err.Error())
			}
			numModule++
		}
	}

	return nil
}

// Checks the a slice of TaskProxy ptrs passed in by a Plan and determines
// whether if it's an include value or a normal task.  If it's a normal task
// it appends it as a standard task, otherwise it recursively expands the include
// statement
func PreprocessTasks(taskSection []*TaskProxy, planVars VarsMap, sudo bool) ([]*Task, error) {
	tasksList, err := parseTaskProxies(taskSection, planVars, "", sudo)
	if err != nil {
		return nil, err
	}

	return tasksList, nil
}

func preprocessTasksHelper(buf []byte, prevVars VarsMap, prevWhen string, sudo bool) ([]*Task, error) {
	var px PlanProxy
	err := yaml.Unmarshal(buf, &px)
	if err != nil {
		return nil, err
	}

	return parseTaskProxies(px.TaskProxies, prevVars, prevWhen, sudo)
}

func parseTaskProxies(taskProxies []*TaskProxy, prevVars VarsMap, prevWhen string, sudo bool) ([]*Task, error) {
	var tasks []*Task
	for _, tp := range taskProxies {
		task := Task{}
		// links when paramter
		// put out here b/c every task can have a when
		if tp.When != "" && prevWhen != "" {
			tp.When = tp.When + " && " + prevWhen
		} else if prevWhen != "" {
			tp.When = prevWhen
		}

		if tp.Include != "" {
			buf, err := ioutil.ReadFile(tp.Include)
			if err != nil {
				return nil, err
			}

			// links previous vars
			if tp.Vars == nil {
				tp.Vars = make(VarsMap)
			}
			mergeMap(prevVars, tp.Vars, false)
			includedTasks, err := preprocessTasksHelper(buf, tp.Vars, tp.When, sudo)
			if err != nil {
				return nil, err
			}

			tasks = append(tasks, includedTasks...)
		} else {
			if tp.Module == nil {
				return nil, fmt.Errorf("This task doesn't have a valid module")
			}
			task.Name = tp.Name
			task.Module = tp.Module
			task.IgnoreErrors = tp.IgnoreErrors
			task.Local = tp.Local
			task.Register = tp.Register
			task.When = tp.When
			// NOTE: assigns to prevVars not tp.Vars
			task.Vars = prevVars
			task.Sudo = sudo
			if tp.Sudo {
				task.Sudo = tp.Sudo
			}

			tasks = append(tasks, &task)
		}
	}

	return tasks, nil
}

// Processes plan level vars with includes
// All plan level vars will be in the vars map
// And any repeat vars in the includes will be a FCFS priority
// NOTE: if the user has multipl include blocks it'll grab the one closest to
//       the bottom

func PreprocessVars(vars VarsMap) (VarsMap, error) {
	newVars := vars

	// parses include statements in vars
	if fileList, present := vars["include"]; present {
		for _, fName := range fileList.([]interface{}) {
			tempVars, err := preprocessVarsHelper(fName)
			if err != nil {
				return nil, fmt.Errorf("While checking includes - %s", err.Error())
			}
			mergeMap(tempVars, newVars, false)
		}
	}
	delete(newVars, "include")
	return newVars, nil
}

func preprocessVarsHelper(fName interface{}) (VarsMap, error) {
	newFName, found := fName.(string)
	if !found {
		return nil, ErrWrongType("Include", fName, "string")
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

	return px.VarsProxy.Vars, nil
}

// Process hosts list.  Checks the host list to see if any of the
// hosts entries are valid sections and will extract it based on
func filterInventory(groups []string, fullInventory Inventory) Inventory {
	// FIXME: Support globbing in the groups
	// No groups? No problem. Just return the full inventory
	if len(groups) == 0 {
		return fullInventory
	} else {
		filtered := make(Inventory)
		for _, group := range groups {
			machines, present := fullInventory[group]
			if present {
				filtered[group] = machines
			}
		}
		return filtered
	}
}

// For Plan
// NOTE: inventory should always be initialized and passed in?
//       or should we just check to see if it's nil?
func PreprocessPlan(buf []byte, inv Inventory) (*Plan, error) {
	var px PlanProxy
	err := yaml.Unmarshal(buf, &px)
	if err != nil {
		return nil, fmt.Errorf("Error processing plan - %s", err.Error())
	}
	plan := Plan{}
	plan.Inventory = filterInventory(px.InventoryGroups, inv)

	vars := make(VarsMap)
	if px.VarsProxy != nil {
		vars, err = PreprocessVars(px.VarsProxy.Vars)
		if err != nil {
			return nil, fmt.Errorf("Error processing vars - %s", err.Error())
		}
	}
	vars["inventory"] = inv
	plan.Vars = vars

	tasks, err := PreprocessTasks(px.TaskProxies, plan.Vars, px.Sudo)
	if err != nil {
		return nil, fmt.Errorf("Error processing tasks - %s", err.Error())
	}
	plan.Tasks = tasks
	return &plan, err
}
