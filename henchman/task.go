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

func (task *Task) renderValue(value string) (string, error) {
	tmpl, err := pongo2.FromString(value)
	if err != nil {
		log.Println("tmpl error")
		return "", err
	}
	// NOTE: add an update context when regMap is passed in
	ctxt := pongo2.Context{"vars": task.Vars} //, "machine": machine}
	out, err := tmpl.Execute(ctxt)
	if err != nil {
		log.Println("execute error")
		return "", err
	}
	return out, nil

}

// Renders any pongo2 formatting and converts it back to a task
func (task *Task) Render(input interface{}) (interface{}, error) {
	// changes Task struct back to a string so
	// templating can be done
	switch value := input.(type) {
	case map[string]string:
		output := make(map[string]string)
		for k, v := range value {
			result, err := task.renderValue(v)
			if err != nil {
				return "", err
			}
			output[k] = result
		}
		return output, nil
	case string:
		return task.renderValue(value)

	default:
		return "", errors.New("Unexpected value type passed to render")
	}
}

func (task *Task) Run(machine *Machine) (*TaskResult, error) {
	//Render task
	name, err := task.Render(task.Name)
	if err != nil {
		return &TaskResult{}, err
	}

	when, err := task.Render(task.When)
	if err != nil {
		return &TaskResult{}, err
	}

	params, err := task.Render(task.Module.Params)
	if err != nil {
		return &TaskResult{}, err
	}

	task.Name = name.(string)
	task.When = when.(string)
	task.Module.Params = params.(map[string]string)

	task.Id = uuid.New()
	if len(task.Vars) == 0 {
		task.Vars = make(VarsMap)
	}
	task.Vars["current_host"] = machine
	log.Println(task.Module)
	// Resolving module path
	modPath, err := task.Module.Resolve()
	if err != nil {
		return &TaskResult{}, err
	}
	log.Println("modpath ", modPath)
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
	return &taskResult, nil
}
