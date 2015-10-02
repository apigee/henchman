package henchman

import (
	"bytes"
	"encoding/json"
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
			_, err = machine.Transport.Exec(fmt.Sprintf("mkdir -p %s\n", remoteModDir),
				nil, false)
			if err != nil {
				log.Printf("Error while creating remote module path\n")
				return &TaskResult{}, err
			}

		case "put_module":
			err = machine.Transport.Put(modPath, remoteModDir, "dir")
			if err != nil {
				return &TaskResult{}, err
			}

		case "exec_module":
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
			localSrcPath, dstPath := "", ""
			//			k, present := task.Module.Params["src"]
			for k, v := range task.Module.Params {
				if k == "src" {
					localSrcPath = v
				}
				if k == "dest" {
					dstPath = v
				}
			}
			dstPath = strings.Trim(dstPath, "'")
			remoteModDir := "$HOME/.henchman"
			localSrcPath = strings.Trim(localSrcPath, "'")
			_, localSrc := path.Split(localSrcPath)
			srcPath := path.Join(remoteModDir, localSrc)
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
			srcPath := ""
			for k, v := range task.Module.Params {
				if k == "src" {
					srcPath = v
				}
			}
			curDir, err := os.Getwd()
			if err != nil {
				return &TaskResult{}, err
			}
			srcPath = strings.Trim(srcPath, "'")
			remoteModDir := "$HOME/.henchman"
			_, src := path.Split(srcPath)
			dstPath := path.Join(remoteModDir, src)
			completeSrcPath := path.Join(curDir, srcPath)
			log.Println("srcpath", completeSrcPath, "destpath", dstPath)
			err = machine.Transport.Put(completeSrcPath, dstPath, "file")

			if err != nil {
				return &TaskResult{}, err
			}
			//msg := fmt.Sprintf("Copied %s to %s", completeSrcPath, dstPath)
			//			taskResult := &TaskResult{State: "changed", Msg: msg}
			//			log.Println(taskResult)
		}
		// to be implemented
		//		case 'exec_template':
		//		case 'copy_remote':
		//	}
	}
	return &taskResult, nil
}
