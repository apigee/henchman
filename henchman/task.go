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

	ctxt := pongo2.Context{"vars": varsMap}
	ctxt = ctxt.Update(registerMap)

	out, err := tmpl.Execute(ctxt)
	if err != nil {
		log.Println("execute error")
		return "", err
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
		return false, err
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
		return &TaskResult{}, err
	}

	if proceed == false {
		return &TaskResult{State: "skipped"}, nil
	}

	modPath, err := task.Module.Resolve()
	if err != nil {
		return &TaskResult{}, err
	}

	execOrder, err := task.Module.ExecOrder()
	if err != nil {
		log.Printf("Error while creating remote module path\n")
		return &TaskResult{}, err
	}
	remoteModDir := "${HOME}/.henchman"
	remoteModPath := path.Join(remoteModDir, task.Module.Name)
	log.Println("exec order", execOrder)

	var taskResult TaskResult
	for _, execStep := range execOrder {
		switch execStep {
		case "create_dir":
			// creates remote .henchman location
			_, err = machine.Transport.Exec(fmt.Sprintf("mkdir -p %s", remoteModDir),
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
			jsonParams, err := json.Marshal(moduleParams)
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

		case "copy_remote":
			//copies file from remote .henchman location to expected location
			remoteSrcPath, present := moduleParams["src"]
			if !present {
				return &TaskResult{}, errors.New("Unable to find 'src' parameter")
			}

			dstPath, present := moduleParams["dest"]
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
			srcPath, present := moduleParams["src"]
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
			srcPath, present := moduleParams["src"]
			if !present {
				return &TaskResult{}, errors.New("Unable to find 'src' parameter")
			}
			tpl, err := pongo2.FromFile(srcPath)
			if err != nil {
				return &TaskResult{}, err
			}
			out, err := tpl.Execute(pongo2.Context{"vars": vars})
			if err != nil {
				return &TaskResult{}, err
			}
			tmpDir, srcFile := path.Split(srcPath)
			srcFile = srcFile + "_" + machine.Hostname
			tmpFileName := fmt.Sprintf(".%s", srcFile)
			tmpFile := path.Join(tmpDir, tmpFileName)

			err = ioutil.WriteFile(tmpFile, []byte(out), 0644)
			if err != nil {
				return &TaskResult{}, err
			}
			moduleParams["srcOrig"] = srcPath
			moduleParams["src"] = tmpFile
		case "reset_src":
			moduleParams["src"] = moduleParams["srcOrig"]
		}
	}

	if task.Register != "" {
		registerMap[task.Register] = taskResult
	}

	return &taskResult, nil
}
