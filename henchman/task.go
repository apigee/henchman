package henchman

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"

	"code.google.com/p/go-uuid/uuid"
	"github.com/flosch/pongo2"
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
}

type TaskResult struct {
	State  string `json:"status"`
	Output string `json:"output,omitempty"`
	Msg    string `json:"msg"`
}

func getTaskResult(buf *bytes.Buffer) (*TaskResult, error) {
	var taskResult TaskResult
	resultInBytes := []byte(buf.String())
	err := json.Unmarshal(resultInBytes, &taskResult)
	if err != nil {
		return &TaskResult{}, err
	}
	return &taskResult, nil
}

func setTaskResult(taskResult *TaskResult, buf *bytes.Buffer) error {
	resultInBytes := []byte(buf.String())
	err := json.Unmarshal(resultInBytes, &taskResult)
	if err != nil {
		return err
	}
	return nil
}

// strings will be evaluated using pongo2 templating with context of
// VarsMap and RegisterMap
func renderValue(value string, varsMap VarsMap, registerMap map[string]interface{}) (string, error) {
	tmpl, err := pongo2.FromString(value)
	if err != nil {
		log.Println("tmpl error")
		return "", err
	}
	// NOTE: add an update context when regMap is passed in
	ctxt := pongo2.Context{"vars": varsMap} //, "machine": machine}
	ctxt = ctxt.Update(registerMap)

	out, err := tmpl.Execute(ctxt)
	if err != nil {
		log.Println("execute error")
		return "", err
	}
	return out, nil
}

// wrapper for Rendering each task
func (task *Task) Render(registerMap RegMap) error {
	var err error
	task.Name, err = renderValue(task.Name, task.Vars, registerMap)
	if err != nil {
		return err
	}

	//NOTE: just a place holder here since ProcessWhen will actually check the when line
	task.When, err = renderValue(task.When, task.Vars, registerMap)
	if err != nil {
		return err
	}

	for k, v := range task.Module.Params {
		task.Module.Params[k], err = renderValue(v, task.Vars, registerMap)
		if err != nil {
			return err
		}
	}

	return nil
}

// Does a conditional check for Tasks When Param.  Any error will cause
// this function to return false
func (task *Task) ProcessWhen(registerMap RegMap) (bool, error) {
	if task.When == "" {
		return true, nil
	}

	out, err := renderValue("{{"+task.When+"}}", task.Vars, registerMap)
	if err != nil {
		return false, err
	}

	result, err := strconv.ParseBool(out)
	if err != nil {
		return false, err
	}

	return result, nil
}

func (task *Task) Run(machine *Machine, registerMap RegMap) (*TaskResult, error) {
	// Add current host to vars
	// NOTE: task.Vars is initialized in preprocessor.go
	if len(task.Vars) == 0 {
		log.Println("Shouldn't enter here since Vars should be initialized in preprocess")
		task.Vars = make(VarsMap)
	}
	task.Vars["current_host"] = machine
	task.Id = uuid.New()

	modPath, err := task.Module.Resolve()
	if err != nil {
		return &TaskResult{}, err
	}

	execOrder, err := task.Module.ExecOrder()
	if err != nil {
		log.Printf("Error while creating remote module path\n")
		return &TaskResult{}, err
	}
	remoteModDir := "$HOME/.henchman"
	remoteModPath := path.Join(remoteModDir, task.Module.Name)
	log.Println("exec order", execOrder)

	var taskResult TaskResult
	for _, execStep := range execOrder {
		switch execStep {
		case "create_dir":
			// creates remote .henchman location
			_, err = machine.Transport.Exec(fmt.Sprintf("mkdir -p %s\n", remoteModDir),
				nil, false)
			if err != nil {
				log.Printf("Error while creating remote module path\n")
				return &TaskResult{}, err
			}

		case "put_module":
			// copies module from local location to remote location
			err = machine.Transport.Put(modPath, remoteModDir, "dir")
			if err != nil {
				return &TaskResult{}, err
			}

		case "exec_module":
			// executes module by calling the copied module remotely
			log.Printf("Executing script - %s\n", remoteModPath)
			jsonParams, err := json.Marshal(task.Module.Params)
			if err != nil {
				return &TaskResult{}, err
			}
			buf, err := machine.Transport.Exec(remoteModPath, jsonParams, task.Sudo)
			if err != nil {
				return &TaskResult{}, err
			}
			//This should not be empty
			err = setTaskResult(&taskResult, buf)
			if err != nil {
				return &TaskResult{}, err
			}
			log.Println(taskResult)

		case "copy_remote":
			//copies file from remote .henchman location to expected location
			remoteSrcPath, present := task.Module.Params["src"]
			if !present {
				return &TaskResult{}, errors.New("Unable to find 'src' parameter")
			}

			dstPath, present := task.Module.Params["dest"]
			if !present {
				return &TaskResult{}, errors.New("Unable to find 'dest' parameter")
			}

			_, localSrcFile := path.Split(remoteSrcPath)
			srcPath := path.Join(remoteModDir, localSrcFile)

			cmd := fmt.Sprintf("/bin/cp %s %s", srcPath, dstPath)
			buf, err := machine.Transport.Exec(cmd, nil, task.Sudo)
			if err != nil {
				return &TaskResult{}, err
			}
			if len(buf.String()) != 0 {
				err = setTaskResult(&taskResult, buf)
				if err != nil {
					return &TaskResult{}, err
				}
				log.Println(taskResult)
			}
		case "put_file":
			//scp's file from local location to remote location
			srcPath, present := task.Module.Params["src"]
			if !present {
				return &TaskResult{}, errors.New("Unable to find 'src' parameter")
			}
			curDir, err := os.Getwd()
			if err != nil {
				return &TaskResult{}, err
			}
			_, srcFile := path.Split(srcPath)
			dstPath := path.Join(remoteModDir, srcFile)
			completeSrcPath := path.Join(curDir, srcPath)

			err = machine.Transport.Put(completeSrcPath, dstPath, "file")

			if err != nil {
				return &TaskResult{}, err
			}

		case "process_template":
			srcPath, present := task.Module.Params["src"]
			if !present {
				return &TaskResult{}, errors.New("Unable to find 'src' parameter")
			}
			tpl, err := pongo2.FromFile(srcPath)
			if err != nil {
				return &TaskResult{}, err
			}
			out, err := tpl.Execute(pongo2.Context{"vars": task.Vars})
			if err != nil {
				return &TaskResult{}, err
			}
			tmpDir, srcFile := path.Split(srcPath)
			tmpFileName := fmt.Sprintf(".%s", srcFile)
			tmpFile := path.Join(tmpDir, tmpFileName)

			err = ioutil.WriteFile(tmpFile, []byte(out), 0644)
			if err != nil {
				return &TaskResult{}, err
			}
			task.Module.Params["srcOrig"] = srcPath
			task.Module.Params["src"] = tmpFile

		case "reset_src":
			task.Module.Params["src"] = task.Module.Params["srcOrig"]
			delete(task.Module.Params, "srcOrig")
		}
	}

	if task.Register != "" {
		registerMap[task.Register] = taskResult
	}

	return &taskResult, nil
}
