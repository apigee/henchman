package henchman

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "gopkg.in/Sirupsen/logrus.v0"
	"io/ioutil"
	"path"
	"strconv"

	"github.com/flosch/pongo2"
	"github.com/pborman/uuid"
)

type Task struct {
	Id           string
	Sudo         bool
	Name         string
	Module       *Module
	IgnoreErrors bool `yaml:"ignore_errors"`
	Local        bool
	When         string
	Register     string
	Vars         VarsMap
	Debug        bool
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
	remoteModPath := path.Join(remoteModDir, task.Module.Name)
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
			log.WithFields(log.Fields{
				"mod path": remoteModPath,
				"host":     task.Vars["current_host"],
				"task":     task.Name,
				"module":   task.Module.Name,
			}).Info("Executing Module in Task")

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
			log.WithFields(log.Fields{
				"mod path": newModPath,
				"host":     task.Vars["current_host"],
				"task":     task.Name,
				"module":   task.Module.Name,
			}).Info("Executing Module in Task")
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
		case "put_file":
			//scp's file from local location to remote location
			srcPath, present := moduleParams["src"]
			if !present {
				return &TaskResult{}, HenchErr(fmt.Errorf("Unable to find 'src' parameter"), nil, "")
			}
			_, srcFile := path.Split(srcPath)
			dstPath := path.Join(remoteModDir, srcFile)

			err = machine.Transport.Put(srcPath, dstPath, "file")

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
			dstFldr := path.Dir(dstPath)

			if !present {
				return &TaskResult{}, HenchErr(fmt.Errorf("Unable to find 'dest' parameter"), nil, "")
			}

			srcPath := path.Join(remoteModDir, path.Base(remoteSrcPath))

			cmd := fmt.Sprintf("/bin/mkdir -p %s && /bin/cp -r %s %s", dstFldr, srcPath, dstPath)
			buf, err := machine.Transport.Exec(cmd, nil, task.Sudo)
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

			tpl, err := pongo2.FromFile(srcPath)
			if err != nil {
				return &TaskResult{}, HenchErr(err, map[string]interface{}{
					"file":     srcPath,
					"solution": "Verify if src file has proper pongo2 formatting",
				}, "While processing template file")
			}
			out, err := tpl.Execute(pongo2.Context{"vars": vars})
			if err != nil {
				return &TaskResult{}, HenchErr(err, map[string]interface{}{
					"file":     srcPath,
					"solution": "Verify if src file has proper pongo2 formatting",
				}, "While processing template file")
			}
			tmpDir, srcFile := path.Split(srcPath)
			srcFile = srcFile + "_" + machine.Hostname
			tmpFileName := fmt.Sprintf(".%s", srcFile)
			tmpFile := path.Join(tmpDir, tmpFileName)

			err = ioutil.WriteFile(tmpFile, []byte(out), 0644)
			if err != nil {
				return &TaskResult{}, HenchErr(err, nil, "While processing template file")
			}
			moduleParams["srcOrig"] = srcPath
			moduleParams["src"] = tmpFile
		case "reset_src":
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
