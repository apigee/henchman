package henchman

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/flosch/pongo2"
	"github.com/pborman/uuid"
)

var renderLock sync.Mutex

type Task struct {
	Id           string
	Debug        bool
	IgnoreErrors bool `yaml:"ignore_errors"`
	Local        bool
	Module       Module
	Name         string
	Register     string
	Retry        int
	Sudo         bool
	Vars         VarsMap
	When         string
	WithItems    interface{} `yaml:"with_items"`
}

type TaskResult struct {
	State  string      `json:"status"`
	Output interface{} `json:"output,omitempty"`
	Msg    string      `json:"msg"`
}

func setTaskResult(taskResult *TaskResult, buf *bytes.Buffer) error {
	resultStr := buf.String()
	resultInBytes := []byte(resultStr)
	//fmt.Printf("DEBUG: taskresult - %s - length - %d\n", resultStr, len(resultStr))
	err := json.Unmarshal(resultInBytes, &taskResult)
	if err != nil {
		return HenchErr(err, map[string]interface{}{
			"len":   len(resultInBytes),
			"input": ":" + resultStr + ":",
		}, "Task result json string")
	}
	return nil
}

// ProcessWithItems checks for the with_items files in a task.  If it's present it will generate a list of rendered tasks.
func (task *Task) ProcessWithItems(varsMap VarsMap, regMap RegMap) ([]*Task, error) {
	// NOTE: placing item variables into regMap since item is a keyword and
	var newTasks []*Task
	var itemList []interface{}
	if task.WithItems != nil {
		// {{ somelist }} case
		if reflect.TypeOf(task.WithItems).Name() == "string" {
			// some string parsing magic to simulate pongo2
			newWithItems := strings.Trim(task.WithItems.(string), "{{}}")
			newWithItems = strings.TrimSpace(newWithItems)
			newWithItems = strings.TrimPrefix(newWithItems, "vars.")

			var present bool
			itemList, present = varsMap[newWithItems].([]interface{})
			if !present {
				return nil, fmt.Errorf("The with_item rendered variable '%s' is not of type []interface{}", task.WithItems)
			}
		} else {
			itemList = task.WithItems.([]interface{})
		}

		newTasks = []*Task{}
		for _, v := range itemList {
			switch v.(type) {
			case string:
				renderedVal, err := renderValue(v.(string), varsMap, regMap)
				if err != nil {
					return nil, err
				}
				regMap["item"] = renderedVal
			case map[interface{}]interface{}:
				//NOTE: will use this for rendering if needed in the future
				regMap["item"] = v
			default:
				return nil, HenchErr(
					fmt.Errorf("Components in with_items must be a string or a json object."),
					map[string]interface{}{
						"item": v,
					}, "")
			}

			newTask, err := task.Render(varsMap, regMap)
			if err != nil {
				return nil, err
			}
			newTasks = append(newTasks, newTask)
		}
	}

	return newTasks, nil
}

// wrapper for Rendering each task
// This will return the rendered task and not manipulate the pointer to the
// task. b/c the pointer to the task is a template and a race condition can occur.
func (task Task) Render(vars VarsMap, registerMap RegMap) (*Task, error) {
	renderLock.Lock()
	defer renderLock.Unlock()

	var err error
	task.Name, err = renderValue(task.Name, vars, registerMap)
	if err != nil {
		return &task, err
	}

	if task.When != "" {
		task.When, err = renderValue("{{"+task.When+"}}", vars, registerMap)
		if err != nil {
			return &task, err
		}
	}

	// necessary b/c maps are ptrs and race conditions in here and Run(...)
	renderedModuleParams := make(map[string]string)
	for k, v := range task.Module.Params {
		renderedModuleParams[k], err = renderValue(v, vars, registerMap)
		if err != nil {
			return &task, err
		}
	}

	task.Module.Params = renderedModuleParams
	return &task, nil
}

// strings will be evaluated using pongo2 templating with context of
// VarsMap and RegisterMap
func renderValue(value string, varsMap VarsMap, registerMap map[string]interface{}) (string, error) {
	tmpl, err := pongo2.FromString(value)
	if err != nil {
		return "", HenchErr(err, map[string]interface{}{
			"value":    value,
			"solution": "Refer to wiki for proper pongo2 formatting",
		}, "While templating")
	}

	ctxt := pongo2.Context{"vars": varsMap}
	ctxt = ctxt.Update(registerMap)

	out, err := tmpl.Execute(ctxt)
	if err != nil {
		return "", HenchErr(err, map[string]interface{}{
			"value":    value,
			"context":  ctxt,
			"solution": "Refer to wiki for proper pongo2 formatting",
		}, "While executing")
	}
	return out, nil
}

// Creates the final Vars to be executed for the task
// Needs to produce a separate vars map because other tasks may use task.Vars
func (task *Task) SetupVars(plan *Plan, machine *Machine, vars VarsMap, registerMap RegMap) error {
	MergeMap(plan.Vars, vars, true)
	MergeMap(machine.Vars, vars, true)

	if err := task.RenderVars(vars, registerMap); err != nil {
		return HenchErr(err, map[string]interface{}{
			"plan":      plan.Name,
			"task":      task.Name,
			"host":      machine.Hostname,
			"task_vars": task.Vars,
		}, fmt.Sprintf("Error rendering task vars '%s'", task.Name))
	}

	MergeMap(task.Vars, vars, true)
	vars["current_hostname"] = machine.Hostname

	Debug(map[string]interface{}{
		"vars": fmt.Sprintf("%v", vars),
		"plan": plan.Name,
		"task": task.Name,
		"host": machine.Hostname,
	}, "Vars for Task")

	return nil
}

// renders the task level variables with global vars
func (task Task) RenderVars(varsMap VarsMap, registerMap map[string]interface{}) error {
	renderLock.Lock()
	defer renderLock.Unlock()
	ctxt := pongo2.Context{"vars": varsMap}
	ctxt = ctxt.Update(registerMap)

	if err := renderVarsHelper(task.Vars, ctxt); err != nil {
		return err
	}

	return nil
}

func renderVarsHelper(varsMap VarsMap, ctxt pongo2.Context) error {
	for key, value := range varsMap {
		switch v := value.(type) {
		case map[string]interface{}:
			if err := renderVarsHelper(varsMap, ctxt); err != nil {
				return HenchErr(err, map[string]interface{}{
					"value_map": value,
					"solution":  "Refer to wiki for proper pongo2 formatting",
				}, "While templating")
			}
		case string:
			tmpl, err := pongo2.FromString(v)
			if err != nil {
				return HenchErr(err, map[string]interface{}{
					"value":    value,
					"solution": "Refer to wiki for proper pongo2 formatting",
				}, "While templating")
			}
			out, err := tmpl.Execute(ctxt)
			if err != nil {
				return HenchErr(err, map[string]interface{}{
					"value":    value,
					"context":  ctxt,
					"solution": "Refer to wiki for proper pongo2 formatting",
				}, "While executing")
			}
			varsMap[key] = out
		}
	}

	return nil
}

// checks and converts when to bool
func (task *Task) ProcessWhen() (bool, error) {
	if task.When == "" {
		return true, nil
	}

	result, err := strconv.ParseBool(task.When)
	if err != nil {
		return false, HenchErr(err, map[string]interface{}{
			"task_when": task.When,
			"solution":  "make sure value is a bool",
		}, "")
	}

	return result, nil
}

func (task *Task) Run(machine *Machine, vars VarsMap, registerMap RegMap) (*TaskResult, error) {
	// Add current host to vars
	task.Id = uuid.New()

	//local copy of module params for each machine
	moduleParams := make(map[string]string)
	for k, v := range task.Module.Params {
		moduleParams[k] = v
	}
	proceed, err := task.ProcessWhen()
	if err != nil {
		return &TaskResult{}, HenchErr(err, nil, "While processing when")
	}

	if proceed == false {
		return &TaskResult{State: "skipped"}, nil
	}

	execOrder, err := task.Module.ExecOrder()
	// currently err will always be nil
	if err != nil {
		return &TaskResult{}, HenchErr(err, nil, "")
	}

	REMOTE_DIR := "${HOME}/.henchman/"

	// NOTE: Info or Debug level
	Debug(map[string]interface{}{
		"task":   task.Name,
		"host":   machine.Hostname,
		"module": task.Module.Name,
		"order":  execOrder,
	}, "Exec Order")

	var taskResult TaskResult
	for _, execStep := range execOrder {
		switch execStep {
		// Exec Order for Default
		case "exec_module":
			// Checks if the module is a standalone or has dependecies
			osName, err := getOsName(machine)
			if err != nil {
				return &TaskResult{}, HenchErr(err, nil, "While retrieving osName")
			}

			modPath, standalone, err := task.Module.Resolve(osName)
			if err != nil {
				return &TaskResult{}, HenchErr(err, nil, "While in exec_module")
			}

			// Creates the remoteModPath to use the module
			modPathChunks := strings.Split(modPath, "/")
			modPath = modPathChunks[len(modPathChunks)-1]
			if !standalone {
				modPath += "/exec"
			}
			remoteModPath := filepath.Join(REMOTE_DIR, modPath)

			Info(map[string]interface{}{
				"mod path": remoteModPath,
				"host":     machine.Hostname,
				"task":     task.Name,
				"module":   task.Module.Name,
			}, "Executing Module in Task")

			jsonParams, err := json.Marshal(moduleParams)
			if err != nil {
				return &TaskResult{}, HenchErr(err, nil, "In exec_module while json marshalling")
			}

			buf, err := machine.Transport.Exec(remoteModPath, jsonParams, task.Sudo)
			if err != nil {
				return &TaskResult{}, HenchErr(err, nil, "While in exec_module")
			}

			//This should not be empty
			err = setTaskResult(&taskResult, buf)
			if err != nil {
				return &TaskResult{}, HenchErr(err, nil, "While in exec_module")
			}
		case "stage":
			//scp's file from local location to remote location
			srcPath, present := moduleParams["src"]
			if !present {
				return &TaskResult{}, HenchErr(fmt.Errorf("Unable to find 'src' parameter"), nil, "")
			}

			info, err := os.Stat(srcPath)
			if err != nil {
				return &TaskResult{}, HenchErr(err, nil, "Unable to get info on file/dir")
			}

			if info.IsDir() {
				err = machine.Transport.Put(srcPath, REMOTE_DIR, "dir")
			} else {
				err = machine.Transport.Put(srcPath, REMOTE_DIR, "file")
			}

			if err != nil {
				return &TaskResult{}, HenchErr(err, nil, "Putting File")
			}
			moduleParams["rmtSrc"] = filepath.Join(".henchman/", filepath.Base(srcPath))
		case "process_template":
			if err := processTemplate(moduleParams, vars, machine.Hostname); err != nil {
				return &TaskResult{}, HenchErr(err, nil, "While processing templates")
			}
		case "reset_src":
			// remove the tplDir
			tplDir := machine.Hostname + "_templates"
			if err := os.RemoveAll(tplDir); err != nil {
				return &TaskResult{}, HenchErr(err, nil, "While in reset_src")
			}
			moduleParams["src"] = moduleParams["srcOrig"]
		}
	}

	// Set to status to ignored if the result is a failure
	if task.IgnoreErrors && (taskResult.State == "error" || taskResult.State == "failure") {
		taskResult.State = "ignored"
	}

	if task.Register != "" {
		registerMap[task.Register] = taskResult
	}

	return &taskResult, nil
}

func processTemplate(moduleParams map[string]string, vars VarsMap, hostname string) error {
	srcPath, present := moduleParams["src"]
	if !present {
		return HenchErr(fmt.Errorf("Unable to find 'src' parameter"), nil, "")
	}

	info, err := os.Stat(srcPath)
	if err != nil {
		return HenchErr(err, nil, "Unable to get info on file/dir")
	}

	// creates temp directory to store templated files
	tplDir := hostname + "_templates"

	if err := CreateDir(tplDir); err != nil {
		return err
	}

	// if the file(s) to template is a folder
	// create the approriate director in tplDir
	baseDir := ""
	if info.IsDir() {
		baseDir = filepath.Join(tplDir, filepath.Base(srcPath))
		if err := os.Mkdir(baseDir, 0755); err != nil {
			return HenchErr(err, nil, "Error creating baseDir for templating folders")
		}
	}

	err = filepath.Walk(srcPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return HenchErr(err, map[string]interface{}{
					"path": path,
				}, "While walking")
			}

			if info.IsDir() {
				// FIXME: make a cleaner work around
				// since first file checked is the dir, can't make dir
				ext := strings.TrimPrefix(path, srcPath)
				if ext != "" {
					return os.Mkdir(filepath.Join(baseDir, ext), 0755)
				}

				return nil
			}

			var buf []byte
			finfo, _ := os.Stat(path)
			if (finfo.Mode()&0111) != 0 ||
				filepath.Ext(path) != "" &&
					strings.Contains(IGNORED_EXTS, strings.TrimPrefix(filepath.Ext(path), ".")) &&
					strings.Contains(moduleParams["ext"], strings.TrimPrefix(filepath.Ext(path), ".")) {
				buf, err = ioutil.ReadFile(path)
				if err != nil {
					return HenchErr(err, map[string]interface{}{
						"file":     path,
						"source":   srcPath,
						"solution": "Verify if src file has proper pongo2 formatting",
					}, "While copying excluded files")
				}
			} else {
				tpl, err := pongo2.FromFile(path)
				if err != nil {
					return HenchErr(err, map[string]interface{}{
						"source":   srcPath,
						"file":     path,
						"solution": "Verify if src file has proper pongo2 formatting",
					}, "While processing template file")
				}
				out, err := tpl.Execute(pongo2.Context{"vars": vars})
				if err != nil {
					return HenchErr(err, map[string]interface{}{
						"source":   srcPath,
						"file":     path,
						"solution": "Verify if src file has proper pongo2 formatting",
					}, "While processing template file")
				}
				buf = []byte(out)
			}

			renderFilePath := filepath.Join(tplDir, filepath.Base(srcPath))
			if baseDir != "" {
				renderFilePath = filepath.Join(baseDir, strings.TrimPrefix(path, srcPath))
			}

			err = ioutil.WriteFile(renderFilePath, buf, finfo.Mode())
			if err != nil {
				return HenchErr(err, nil, "While processing template file")
			}

			return nil
		})
	if err != nil {
		return HenchErr(err, nil, "While walking in process template")
	}

	moduleParams["srcOrig"] = srcPath
	if baseDir != "" {
		moduleParams["src"] = baseDir
	} else {
		moduleParams["src"] = filepath.Join(tplDir, filepath.Base(srcPath))
	}

	return nil
}
