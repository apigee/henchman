package henchman

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"code.google.com/p/go-uuid/uuid"
	"github.com/flosch/pongo2"
	"gopkg.in/yaml.v2"
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

// Renders any pongo2 formatting and converts it back to a task
func (task *Task) Render(machine *Machine) error {
	var renderedTask Task
	// changes Task struct back to a string so
	// templating can be done
	taskBuf, err := yaml.Marshal(task)
	if err != nil {
		return err
	}
	tmpl, err := pongo2.FromString(string(taskBuf))
	if err != nil {
		return err
	}
	// NOTE: add an update context when regMap is passed in
	ctxt := pongo2.Context{"vars": task.Vars, "machine": machine}
	out, err := tmpl.Execute(ctxt)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal([]byte(out), &renderedTask)
	if err != nil {
		return err
	}
	*task = renderedTask
	return nil
}

func (task *Task) Run(machine *Machine) (*TaskResult, error) {
	// Resolving module path
	task.Id = uuid.New()
	log.Println(task.Module)
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
			localSrcPath, present := task.Module.Params["src"]
			if !present {
				return &TaskResult{}, errors.New("Unable to find 'src' parameter")
			}
			localSrcPath = strings.Trim(localSrcPath, "'")

			dstPath, present := task.Module.Params["dest"]
			if !present {
				return &TaskResult{}, errors.New("Unable to find 'dest' parameter")
			}
			dstPath = strings.Trim(dstPath, "'")

			_, localSrcFile := path.Split(localSrcPath)
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
			srcPath = strings.Trim(srcPath, "'")
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
		}
		// to be implemented
		//		case 'exec_template':
		//	}
	}
	return &taskResult, nil
}
