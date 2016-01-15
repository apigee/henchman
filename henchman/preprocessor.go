package henchman

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

type VarsProxy struct {
	Vars VarsMap
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
		return HenchErr(err,
			map[string]interface{}{
				"solution": "Check for Yaml formatting errors.  Usually indentation with tabs",
			}, "")
	}

	vp.Vars = make(VarsMap)
	for field, val := range vMap {
		switch field {
		case "include":
			vp.Vars["include"], found = val.([]interface{})
			if !found {
				return HenchErr(ErrWrongType(field, val, "[]interface{}"), map[string]interface{}{
					"solution": "Make sure the field is of proper type",
				}, "")
			}

			numInclude++
			if numInclude > 1 {
				return HenchErr(fmt.Errorf("Can only have one include statement at Vars level."),
					map[string]interface{}{
						"solution": "remove extra include statements",
					}, "")
			}
		default:
			if isKeyword(field) {
				return HenchErr(ErrKeyword(field), nil, "While unmarshalling vars")
			}
			vp.Vars[field] = val
		}
	}

	return nil
}

// Task is for the general Task format.  Refer to task.go
// Vars are kept in scope for each Task.  So there is a Vars
// field for each task
// Include is the file name for the included Tasks list
type TaskProxy struct {
	Task        `yaml:",inline"`
	SudoState   string
	DebugState  string
	Include     string
	IncludeVars VarsMap `yaml:"vars"`
}

// Custom unmarshaller which accounts for module names
func (tp *TaskProxy) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var tmap map[string]interface{}
	var found bool
	numModule := 0

	err := unmarshal(&tmap)
	if err != nil {
		return HenchErr(err,
			map[string]interface{}{
				"solution": "Check for Yaml formatting errors.  Usually indentation with tabs",
			}, "")
	}

	// Sets tp.Name for use when returning errors
	if val, ok := tmap["name"]; ok {
		tp.Name, found = val.(string)
		if !found {
			return HenchErr(ErrWrongType("name", val, "string"), map[string]interface{}{
				"task":     tp.Name,
				"solution": "Make sure the field is of proper type",
			}, "")
		}
	}
	for field, val := range tmap {
		switch field {
		case "name":
			// holder so switch won't think it's a module
			break
		case "retry":
			tp.Retry, found = val.(int)
			if !found {
				return HenchErr(ErrWrongType(field, val, "int"), map[string]interface{}{
					"task":     tp.Name,
					"solution": "Make sure the field is of proper type",
				}, "")
			}
		case "sudo":
			tp.Sudo, found = val.(bool)
			if !found {
				return HenchErr(ErrWrongType(field, val, "bool"), map[string]interface{}{
					"task":     tp.Name,
					"solution": "Make sure the field is of proper type",
				}, "")
			}
			tp.SudoState = strconv.FormatBool(tp.Sudo)
		case "debug":
			tp.Debug, found = val.(bool)
			if !found {
				return HenchErr(ErrWrongType(field, val, "bool"), map[string]interface{}{
					"task":     tp.Name,
					"solution": "Make sure the field is of proper type",
				}, "")
			}
			tp.DebugState = strconv.FormatBool(tp.Debug)
		case "ignore_errors":
			tp.IgnoreErrors, found = val.(bool)
			if !found {
				return HenchErr(ErrWrongType(field, val, "bool"), map[string]interface{}{
					"task":     tp.Name,
					"solution": "Make sure the field is of proper type",
				}, "")
			}
		case "with_items":
			switch val.(type) {
			case string, []interface{}:
				tp.WithItems = val
			default:
				return HenchErr(ErrWrongType(field, val, "string or []string or []map[inteface{}]interface{}"), map[string]interface{}{
					"task":     tp.Name,
					"solution": "Make sure the field is of proper type",
				}, "")
			}
		case "local":
			tp.Local, found = val.(bool)
			if !found {
				return HenchErr(ErrWrongType(field, val, "bool"), map[string]interface{}{
					"task":     tp.Name,
					"solution": "Make sure the field is of proper type",
				}, "")
			}
		case "when":
			tp.When, found = val.(string)
			if !found {
				return HenchErr(ErrWrongType(field, val, "string"), map[string]interface{}{
					"task":     tp.Name,
					"solution": "Make sure the field is of proper type",
				}, "")
			}
			if strings.Contains(tp.When, "{{") || strings.Contains(tp.When, "}}") {
				return HenchErr(fmt.Errorf("When field should not include {{ or }}"), map[string]interface{}{
					"task":     tp.Name,
					"solution": "Remove {{ or }} from the when field",
				}, "")
			}
		case "register":
			tp.Register, found = val.(string)
			if !found {
				return HenchErr(ErrWrongType(field, val, "string"), map[string]interface{}{
					"task":     tp.Name,
					"solution": "Make sure the field is of proper type",
				}, "")
			}
			if len(strings.Fields(tp.Register)) > 1 {
				return HenchErr(ErrNotValidVariable(tp.Register), map[string]interface{}{
					"task":     tp.Name,
					"solution": "Registers must be a single word w/o spaces",
				}, "")
			}
			if isKeyword(tp.Register) {
				return HenchErr(ErrKeyword(tp.Register), map[string]interface{}{
					"task":     tp.Name,
					"field":    "register",
					"solution": "Avoid the key words: vars, item",
				}, "")
			}
		case "include":
			tp.Include, found = val.(string)
			if !found {
				return HenchErr(ErrWrongType(field, val, "string"), map[string]interface{}{
					"task":     tp.Name,
					"solution": "Make sure the field is of proper type",
				}, "")
			}
		case "vars":
			tp.IncludeVars, found = val.(map[interface{}]interface{})
			if !found {
				return HenchErr(ErrWrongType(field, val, "map[interface{}]interface{}"), map[string]interface{}{
					"task":     tp.Name,
					"solution": "Make sure the field is of proper type",
				}, "")
			}
		default:
			// We have a module
			params, found := val.(string)
			if !found {
				return HenchErr(ErrWrongType(field, val, "string"), map[string]interface{}{
					"task":     tp.Name,
					"solution": "Make sure the field is of proper type",
				}, "")
			}
			if numModule > 0 {
				return HenchErr(fmt.Errorf("'%v' is an extra Module.", field), map[string]interface{}{
					"task":     tp.Name,
					"solution": "There can only be one module per task",
				}, "")
			}

			tp.Module, err = NewModule(field, params)
			if err != nil {
				return HenchErr(err, map[string]interface{}{
					"task": tp.Name,
				}, fmt.Sprintf("Module '%v'", field))
			}
			numModule++
		}
	}

	return nil
}

type PlanProxy struct {
	Name            string       `yaml:"name"`
	Sudo            bool         `yaml:"sudo"`
	Debug           bool         `yaml:"debug"`
	TaskProxies     []*TaskProxy `yaml:"tasks"`
	VarsProxy       *VarsProxy   `yaml:"vars"`
	InventoryGroups []string     `yaml:"hosts"`
}

// Checks the a slice of TaskProxy ptrs passed in by a Plan and determines
// whether if it's an include value or a normal task.  If it's a normal task
// it appends it as a standard task, otherwise it recursively expands the include
// statement
// NOTE: making this a function of PlanProxy in case we want to have more plan level variables
//       in the future.  Removes the need to pass each variable as a parameter.
func (px PlanProxy) PreprocessTasks() ([]*Task, error) {
	tasksList, err := parseTaskProxies(&px, nil, "")
	if err != nil {
		return nil, err
	}

	return tasksList, nil
}

// handles the include statements
func preprocessTasksHelper(taskFileName string, prevVars VarsMap, prevWhen string, px *PlanProxy) ([]*Task, error) {
	buf, err := ioutil.ReadFile(taskFileName)
	if err != nil {
		return nil, HenchErr(err, map[string]interface{}{
			"solution": "Verify the included task file exists",
		}, "")
	}

	var tmpPx PlanProxy
	err = yaml.Unmarshal(buf, &tmpPx)
	if err != nil {
		return nil, HenchErr(err, map[string]interface{}{
			"task_file": taskFileName,
			"solution":  "Check for Yaml formatting errors.  Usually indentation with tabs",
		}, "Unmarshalling Included Task")
	}

	// since it's using the included task file proxy
	// gotta assign the original plan level variables
	tmpPx.Sudo = px.Sudo
	tmpPx.Debug = px.Debug
	return parseTaskProxies(&tmpPx, prevVars, prevWhen)
}

// Majority of the work done here.  Assigns and creates the Task list
func parseTaskProxies(px *PlanProxy, prevVars VarsMap, prevWhen string) ([]*Task, error) {
	var tasks []*Task
	for _, tp := range px.TaskProxies {
		task := Task{}
		// links when paramter
		// put out here b/c every task can have a when
		if tp.When != "" && prevWhen != "" {
			tp.When = tp.When + " && " + prevWhen
		} else if prevWhen != "" {
			tp.When = prevWhen
		}

		// recursive case
		if tp.Include != "" {
			// links previous vars
			if prevVars != nil {
				// If this is an nested included task without vars, create a blank map for it
				// Use test includeAtTaskWithTemplateVars as a ref
				if tp.IncludeVars == nil {
					tp.IncludeVars = make(VarsMap)
				}
				MergeMap(prevVars, tp.IncludeVars, false)
			}

			// recursion here
			includedTasks, err := preprocessTasksHelper(tp.Include, tp.IncludeVars, tp.When, px)
			if err != nil {
				return nil, HenchErr(err, map[string]interface{}{
					"task":          tp.Name,
					"while_parsing": "include",
				}, "While checking Include")
			}

			tasks = append(tasks, includedTasks...)
		} else {
			// Checking for name because module is not a pointer
			if tp.Module.Name == "" {
				return nil, HenchErr(fmt.Errorf("This task doesn't have a valid module"), map[string]interface{}{
					"task":     tp.Name,
					"solution": "Each task needs to have exactly one module",
				}, "")
			}
			err := populateTaskFromTaskProxy(&task, tp, prevVars, px)
			if err != nil {
				return nil, err
			}
			//			if tp.WithItems != nil {
			//				for index, item := range tp.WithItems {
			//					t := task
			//					// This is to make sure we have different copies.
			//					// This helps while replacing values
			//					t.Module.Params = nil
			//					t.Module.Params = make(map[string]string)
			//					for k, v := range task.Module.Params {
			//						t.Module.Params[k] = v
			//					}
			//					if strings.Contains(t.Name, "henchman_item") {
			//						t.Name = strings.Replace(t.Name, "henchman_item", item, -1)
			//					} else {
			//						s := strconv.Itoa(index)
			//						t.Name = strings.Replace(t.Name, t.Name, t.Name+s, -1)
			//					}
			//					for k, v := range t.Module.Params {
			//						if strings.Contains(v, "henchman_item") {
			//							v = strings.Replace(v, "henchman_item", item, -1)
			//							t.Module.Params[k] = v
			//						}
			//					}
			//					tasks = append(tasks, &t)
			//				}
			//			} else {
			tasks = append(tasks, &task)
			//			}
		}
	}

	return tasks, nil
}
func populateTaskFromTaskProxy(task *Task, tp *TaskProxy, prevVars VarsMap, px *PlanProxy) error {
	task.Name = tp.Name
	task.Module = tp.Module
	task.IgnoreErrors = tp.IgnoreErrors
	task.Local = tp.Local
	task.Register = tp.Register
	task.Retry = tp.Retry
	task.When = tp.When
	task.WithItems = tp.WithItems

	// NOTE: plan.Vars is merged in plan.execute(...) before Render
	// Creates an empty map for later use if there are no included vars
	if prevVars == nil {
		task.Vars = make(VarsMap)
	} else {
		task.Vars = prevVars
	}

	// takes plan sudo, changes it to task Sudo if there is one
	// does the same for Debug
	task.Sudo = px.Sudo
	if tp.SudoState != "" {
		var err error
		task.Sudo, err = strconv.ParseBool(tp.SudoState)
		if err != nil {
			return HenchErr(err, map[string]interface{}{
				"task":          tp.Name,
				"while_parsing": "sudo",
				"solution":      "Verify the 'sudo' field is a boolean",
			}, "")
		}
	}

	task.Debug = px.Debug
	if tp.DebugState != "" {
		var err error
		task.Debug, err = strconv.ParseBool(tp.DebugState)
		if err != nil {
			return HenchErr(err, map[string]interface{}{
				"task":          tp.Name,
				"while_parsing": "debug",
				"solution":      "Verify the 'debug' field is a boolean",
			}, "")
		}
	}
	return nil

}

// Processes plan level vars with includes
// All plan level vars will be in the vars map
// And any repeat vars in the includes will be a FCFS priority
// NOTE: if the user has multiple include blocks it'll grab the one closest to
//       the bottom
func (px PlanProxy) PreprocessVars() (VarsMap, error) {
	newVars := px.VarsProxy.Vars

	// parses include statements in vars
	if fileList, present := px.VarsProxy.Vars["include"]; present {
		for _, fName := range fileList.([]interface{}) {
			tempVars, err := preprocessVarsHelper(fName)
			if err != nil {
				return nil, HenchErr(err, map[string]interface{}{"while_parsing": "include"}, "While checking includes")
			}
			MergeMap(tempVars, newVars, false)
		}
	}
	delete(newVars, "include")

	return newVars, nil
}

// Reads and extracts the vars fields from include statements
func preprocessVarsHelper(fName interface{}) (VarsMap, error) {
	newFName, found := fName.(string)
	if !found {
		return nil, HenchErr(ErrWrongType("Include", fName, "string"), map[string]interface{}{
			"solution": "Make sure it's not a map",
		}, "")
	}

	buf, err := ioutil.ReadFile(newFName)
	if err != nil {
		return nil, HenchErr(err, map[string]interface{}{
			"file":     newFName,
			"solution": "verify if file exists",
		}, "")
	}

	var px PlanProxy
	err = yaml.Unmarshal(buf, &px)
	if err != nil {
		return nil, HenchErr(err, map[string]interface{}{
			"vars_file": fName,
			"solution":  "Check for Yaml formatting errors.  Usually indentation with tabs",
		}, "Unmarshalling Included Vars")
	}

	return px.VarsProxy.Vars, nil
}

// Calls the other Preprocessing Compenents
func PreprocessPlan(buf []byte, inv *Inventory) (*Plan, error) {
	px, err := newPlanProxy(buf)
	if err != nil {
		return nil, HenchErr(err, nil, "")
	}

	plan := Plan{}
	plan.Inventory = *inv
	plan.Name = px.Name

	//common vars processing
	vars := make(VarsMap)
	if px.VarsProxy != nil {
		vars, err = px.PreprocessVars()
		if err != nil {
			return nil, HenchErr(err, map[string]interface{}{
				"plan":             plan.Name,
				"while_processing": "vars",
			}, "Error processing vars")
		}
	}

	// sets and merges vars for plans, and inv globals
	plan.Vars = vars
	MergeMap(inv.GlobalVars, plan.Vars, false)

	tasks, err := px.PreprocessTasks()
	if err != nil {
		return nil, HenchErr(err, map[string]interface{}{
			"plan":             plan.Name,
			"while_processing": "tasks",
		}, "Error processing tasks")
	}
	// set task local parameter if inventory consists only of localhost
	if _, present := plan.Inventory.Groups["localhost"]; present {
		if len(plan.Inventory.Groups) == 1 {
			for _, task := range tasks {
				task.Local = true
			}
		}
	}
	plan.Tasks = tasks

	return &plan, nil
}

// Creates new plan proxy
func newPlanProxy(buf []byte) (PlanProxy, error) {
	var px PlanProxy
	err := yaml.Unmarshal(buf, &px)
	if err != nil {
		return px, err
	}
	return px, nil
}
