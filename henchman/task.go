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

type TaskVars map[interface{}]interface{}

type Task struct {
	Id           string
	Sudo         bool
	Name         string
	Module       *Module
	IgnoreErrors bool `yaml:"ignore_errors"`
	Local        bool
	When         string
	Register     string
	Vars         TaskVars
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

func (task *Task) Run(machine *Machine) error {
	// Resolving module path
	task.Id = uuid.New()
	log.Println(task.Module)
	modPath, err := task.Module.Resolve()
	if err != nil {
		return err
	}
	log.Println("modpath ", modPath)
	execOrder, err := task.Module.ExecOrder()
	if err != nil {
		log.Printf("Error while creating remote module path\n")
		return err
	}
	remoteModDir := "$HOME/.henchman"
	remoteModPath := path.Join(remoteModDir, task.Module.Name)
	log.Println("exec order", execOrder)
	for _, execStep := range execOrder {
		switch execStep {
		case "create_dir":
			_, err = machine.Transport.Exec(fmt.Sprintf("mkdir -p %s\n", remoteModDir),
				nil, false)
			if err != nil {
				log.Printf("Error while creating remote module path\n")
				return err
			}

		case "put_module":
			log.Printf("Transferring module from %s to %s\n", modPath, remoteModDir)
			err = machine.Transport.Put(modPath, remoteModDir, "dir")
			if err != nil {
				return err
			}

		case "exec_module":
			log.Printf("Executing script - %s\n", remoteModPath)
			jsonParams, err := json.Marshal(task.Module.Params)
			if err != nil {
				return err
			}
			buf, err := machine.Transport.Exec(remoteModPath, jsonParams, task.Sudo)
			if err != nil {
				return err
			}
			taskResult, err := getTaskResult(buf)
			if err != nil {
				return err
			}
			log.Println(taskResult)

		case "copy_src":
			err = machine.Transport.Put(modPath, remoteModDir, "file")
			if err != nil {
				return err
			}
			log.Println(task.Module.Params)
			srcPath, destPath := "", ""
			//owner, group, mode := task.Mac

			for k, v := range task.Module.Params {
				if k == "src" {
					srcPath = v
				}
				if k == "dest" {
					destPath = v
				}
			}
			curDir, err := os.Getwd()
			if err != nil {
				return err
			}
			srcPath = strings.Trim(srcPath, "'")
			destPath = strings.Trim(destPath, "'")
			completeSrcPath := path.Join(curDir, srcPath)

			err = machine.Transport.Put(completeSrcPath, destPath, "file")
			msg := fmt.Sprintf("Copied %s to %s", srcPath, destPath)
			taskResult := &TaskResult{State: "changed", Msg: msg}
			log.Println(taskResult)
		}
		// to be implemented
		//		case 'exec_template':
		//		case 'copy_remote':
		//	}
	}
	return nil
}
