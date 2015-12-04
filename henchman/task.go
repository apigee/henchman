package henchman

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/flosch/pongo2"
	"github.com/pborman/uuid"
)

type Task struct {
	Id           string
	Debug        bool
	IgnoreErrors bool `yaml:"ignore_errors"`
	Local        bool
	Module       *Module
	Name         string
	Register     string
	Retry        int
	Sudo         bool
	Vars         VarsMap
	When         string
}

type TaskResult struct {
	State  string      `json:"status"`
	Output interface{} `json:"output,omitempty"`
	Msg    string      `json:"msg"`
}

func getTaskResult(buf *bytes.Buffer) (*TaskResult, error) {
	var taskResult TaskResult
	resultInBytes := []byte(buf.String())
	err := json.Unmarshal(resultInBytes, &taskResult)
	if err != nil {
		return &TaskResult{}, HenchErr(err, nil, "While unmarshalling task results")
	}
	return &taskResult, nil
}

func setTaskResult(taskResult *TaskResult, buf *bytes.Buffer) error {
	resultInBytes := []byte(buf.String())
	fmt.Println(buf.String())
	err := json.Unmarshal(resultInBytes, &taskResult)
	if err != nil {
		return HenchErr(err, nil, "While unmarshalling task results")
	}
	return nil
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

// wrapper for Rendering each task
func (task *Task) Render(vars VarsMap, registerMap RegMap) error {
	var err error
	task.Name, err = renderValue(task.Name, vars, registerMap)
	if err != nil {
		return err
	}

	if task.When != "" {
		task.When, err = renderValue("{{"+task.When+"}}", vars, registerMap)
		if err != nil {
			return err
		}
	}

	for k, v := range task.Module.Params {
		task.Module.Params[k], err = renderValue(v, vars, registerMap)
		if err != nil {
			return err
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

	remoteModDir := "${HOME}/.henchman/"
	remoteModPath := filepath.Join(remoteModDir, task.Module.Name)
	// NOTE: Info or Debug level
	Debug(map[string]interface{}{
		"task":   task.Name,
		"host":   task.Vars["current_host"],
		"module": task.Module.Name,
		"order":  execOrder,
	}, "Exec Order")

	var taskResult TaskResult
	for _, execStep := range execOrder {
		switch execStep {
		// Exec Order for Default
		case "exec_module":
			// executes module by calling the copied module remotely
			Info(map[string]interface{}{
				"mod path": remoteModPath,
				"host":     task.Vars["current_host"],
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
		case "exec_tar_module":
			// executes module by calling the copied module remotely
			// NOTE: may want to just change the way remoteModPath is created
			newModPath := remoteModDir + task.Module.Name + "/exec"
			Info(map[string]interface{}{
				"mod path": newModPath,
				"host":     task.Vars["current_host"],
				"task":     task.Name,
				"module":   task.Module.Name,
			}, "Executing Module in Task")

			jsonParams, err := json.Marshal(moduleParams)
			if err != nil {
				return &TaskResult{}, HenchErr(err, nil, "In exec_tar_module while json marshalling")
			}
			buf, err := machine.Transport.Exec(newModPath, jsonParams, task.Sudo)
			if err != nil {
				return &TaskResult{}, HenchErr(err, nil, "While in exec_tar_module")
			}
			//This should not be empty
			err = setTaskResult(&taskResult, buf)
			if err != nil {
				return &TaskResult{}, HenchErr(err, nil, "While in exec_tar_module")
			}
		case "put_for_copy":
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
				err = machine.Transport.Put(srcPath, remoteModDir, "dir")
			} else {
				err = machine.Transport.Put(srcPath, remoteModDir, "file")
			}

			if err != nil {
				return &TaskResult{}, HenchErr(err, nil, "Putting File")
			}
		case "copy_remote":
			//copies file from remote .henchman location to expected location
			remoteSrcPath, present := moduleParams["src"]
			if !present {
				return &TaskResult{}, HenchErr(fmt.Errorf("Unable to find 'src' parameter"), nil, "")
			}

			dstPath, present := moduleParams["dest"]
			dstFldr := filepath.Dir(dstPath)

			if !present {
				return &TaskResult{}, HenchErr(fmt.Errorf("Unable to find 'dest' parameter"), nil, "")
			}

			srcPath := filepath.Join(remoteModDir, filepath.Base(remoteSrcPath))

			cmd := fmt.Sprintf("/bin/mkdir -p %s", dstFldr)
			buf, err := machine.Transport.Exec(cmd, nil, task.Sudo)
			if err != nil {
				return &TaskResult{}, HenchErr(err, nil, "While copying file")
			}

			cmd = fmt.Sprintf("/bin/cp -r %s %s", srcPath, dstPath)
			buf, err = machine.Transport.Exec(cmd, nil, task.Sudo)
			if err != nil {
				return &TaskResult{}, HenchErr(err, nil, "While copying file")
			}
			if len(buf.String()) != 0 {
				err = setTaskResult(&taskResult, buf)
				if err != nil {
					return &TaskResult{}, HenchErr(err, nil, "Setting task result from copying file")
				}
			}
		case "process_template":
			srcPath, present := moduleParams["src"]
			if !present {
				return &TaskResult{}, HenchErr(fmt.Errorf("Unable to find 'src' parameter"), nil, "")
			}

			info, err := os.Stat(srcPath)
			if err != nil {
				return &TaskResult{}, HenchErr(err, nil, "Unable to get info on file/dir")
			}

			// creates temp directory to store templated files
			tplDir := machine.Hostname + "_templates"

			// double checks to see if tplDir already exists, if it does remove it
			if _, err := os.Stat(tplDir); os.IsExist(err) {
				if err := os.RemoveAll(tplDir); err != nil {
					return &TaskResult{}, HenchErr(err, nil, "While removing old tplDir")
				}
			}

			if err := os.Mkdir(tplDir, 0755); err != nil {
				return &TaskResult{}, HenchErr(err, nil, "Error creating tplDir for templating")
			}

			// if the file(s) to template is a folder
			// create the approriate director in tplDir
			baseDir := ""
			if info.IsDir() {
				baseDir = filepath.Join(tplDir, filepath.Base(srcPath))
				if err := os.Mkdir(baseDir, 0755); err != nil {
					return &TaskResult{}, HenchErr(err, nil, "Error creating baseDir for templating folders")
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

					tpl, err := pongo2.FromFile(path)
					if err != nil {
						return HenchErr(err, map[string]interface{}{
							"file":     srcPath,
							"solution": "Verify if src file has proper pongo2 formatting",
						}, "While processing template file")
					}
					out, err := tpl.Execute(pongo2.Context{"vars": vars})
					if err != nil {
						return HenchErr(err, map[string]interface{}{
							"file":     srcPath,
							"solution": "Verify if src file has proper pongo2 formatting",
						}, "While processing template file")
					}

					renderFilePath := filepath.Join(tplDir, filepath.Base(srcPath))
					if baseDir != "" {
						renderFilePath = filepath.Join(baseDir, strings.TrimPrefix(path, srcPath))
					}

					err = ioutil.WriteFile(renderFilePath, []byte(out), 0644)
					if err != nil {
						return HenchErr(err, nil, "While processing template file")
					}

					return nil
				})
			if err != nil {
				return &TaskResult{}, HenchErr(err, nil, "While walking in process template")
			}

			moduleParams["srcOrig"] = srcPath
			if baseDir != "" {
				moduleParams["src"] = baseDir
			} else {
				moduleParams["src"] = filepath.Join(tplDir, filepath.Base(srcPath))
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
