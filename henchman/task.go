package henchman

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "gopkg.in/Sirupsen/logrus.v0"
	"io/ioutil"
	"os"
	"os/exec"
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
	State  string      `json:"status"`
	Output interface{} `json:"output,omitempty"`
	Msg    string      `json:"msg"`
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
		return "", fmt.Errorf("Templating :: %s", err.Error())
	}

	ctxt := pongo2.Context{"vars": varsMap}
	ctxt = ctxt.Update(registerMap)

	out, err := tmpl.Execute(ctxt)
	if err != nil {
		return "", fmt.Errorf("Executing :: %s", err.Error())
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

	modPath, _, err := task.Module.Resolve()
	if err != nil {
		return &TaskResult{}, fmt.Errorf("Module Path :: %s", err.Error())
	}

	execOrder, err := task.Module.ExecOrder()
	// currently err will always be nil
	if err != nil {
		return &TaskResult{}, fmt.Errorf("Exec Order :: %s", err.Error())
	}

	remoteModDir := "${HOME}/.henchman/"
	remoteModPath := path.Join(remoteModDir, task.Module.Name)
	// NOTE: Info or Debug level
	Debug(log.Fields{
		"task":   task.Name,
		"host":   task.Vars["current_host"],
		"module": task.Module.Name,
		"order":  execOrder,
	}, "Exec Order")

	var taskResult TaskResult
	for _, execStep := range execOrder {
		switch execStep {
		// Exec Order for Default
		case "create_dir":
			// creates remote .henchman location
			_, err = machine.Transport.Exec(fmt.Sprintf("mkdir -p %s", remoteModDir),
				nil, false)
			if err != nil {
				return &TaskResult{}, fmt.Errorf("Creating Mod Path :: %s", err.Error())
			}

		case "put_module":
			// copies module from local location to remote location
			err = machine.Transport.Put(modPath, remoteModDir, "dir")
			if err != nil {
				return &TaskResult{}, fmt.Errorf("Putting Module :: %s", err.Error())
			}

		case "exec_module":
			// executes module by calling the copied module remotely
			log.WithFields(log.Fields{
				"mod path": remoteModPath,
				"hosts":    task.Vars["current_host"],
				"task":     task.Name,
				"module":   task.Module.Name,
			}).Info("Executing Module in Task")

			jsonParams, err := json.Marshal(moduleParams)
			if err != nil {
				return &TaskResult{}, fmt.Errorf("Exec Module :: Json :: %s", err.Error())
			}
			buf, err := machine.Transport.Exec(remoteModPath, jsonParams, task.Sudo)
			if err != nil {
				return &TaskResult{}, fmt.Errorf("Exec Module :: %s", err.Error())
			}
			//This should not be empty
			err = setTaskResult(&taskResult, buf)
			if err != nil {
				return &TaskResult{}, err
			}

			// Exec Order for dependencies
		case "tar_module":
			// check if the local tar module exists
			// else create the local tar of the module
			if _, err := os.Stat(modPath + ".tar"); os.IsNotExist(err) {
				curDir, err := os.Getwd()
				if err != nil {
					return &TaskResult{}, fmt.Errorf("Tarring Module :: GetWd :: %s", err.Error())
				}

				os.Chdir(modPath)
				os.Chdir("../")
				cmd := "tar"
				args := []string{"-cvf", task.Module.Name + ".tar", task.Module.Name}
				if err := exec.Command(cmd, args...).Run(); err != nil {
					return &TaskResult{}, fmt.Errorf("Tarring Module :: %s", err.Error())
				}
				os.Chdir(curDir)
			}
		case "put_tar_module":
			// copies module from local location to remote location
			err = machine.Transport.Put(modPath+".tar", remoteModDir, "dir")
			if err != nil {
				return &TaskResult{}, fmt.Errorf("Putting Tar Module :: %s", err.Error())
			}

			// deletes module.tar from local modules folder
			/*
				cmd := "rm"
				args := []string{modPath + ".tar"}
				if err := exec.Command(cmd, args...).Run(); err != nil {
					return &TaskResult{}, fmt.Errorf("Putting Tar Module :: %s", err.Error())
				}
			*/
		case "untar_module":
			// untars the module
			cmd := fmt.Sprintf("tar -xvf %s -C %s", remoteModPath+".tar", remoteModDir)
			_, err := machine.Transport.Exec(cmd, nil, task.Sudo)
			if err != nil {
				return &TaskResult{}, fmt.Errorf("Untar Module :: %s", err.Error())
			}

			cmd = fmt.Sprintf("/bin/rm %s", remoteModPath+".tar")
			_, err = machine.Transport.Exec(cmd, nil, task.Sudo)
			if err != nil {
				return &TaskResult{}, fmt.Errorf("Untar Module :: %s", err.Error())
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
				return &TaskResult{}, fmt.Errorf("Exec Tar Module :: Json :: %s", err.Error())
			}
			buf, err := machine.Transport.Exec(newModPath, jsonParams, task.Sudo)
			if err != nil {
				return &TaskResult{}, fmt.Errorf("Exec Tar Module :: %s", err.Error())
			}

			//This should not be empty
			err = setTaskResult(&taskResult, buf)
			if err != nil {
				return &TaskResult{}, fmt.Errorf("Exec Tar Module :: %s", err.Error())
			}
		case "copy_remote":
			//copies file from remote .henchman location to expected location
			remoteSrcPath, present := moduleParams["src"]
			if !present {
				return &TaskResult{}, fmt.Errorf("Unable to find 'src' parameter")
			}

			dstPath, present := moduleParams["dest"]
			if !present {
				return &TaskResult{}, fmt.Errorf("Unable to find 'dest' parameter")
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
				//log.Println(taskResult)
			}
		case "put_file":
			//scp's file from local location to remote location
			srcPath, present := moduleParams["src"]
			if !present {
				return &TaskResult{}, fmt.Errorf("Unable to find 'src' parameter")
			}
			_, srcFile := path.Split(srcPath)
			dstPath := path.Join(remoteModDir, srcFile)

			err = machine.Transport.Put(srcPath, dstPath, "file")

			if err != nil {
				return &TaskResult{}, fmt.Errorf("Putting File :: %s", err.Error())
			}

		case "process_template":
			srcPath, present := moduleParams["src"]
			if !present {
				return &TaskResult{}, fmt.Errorf("Unable to find 'src' parameter")
			}

			tpl, err := pongo2.FromFile(srcPath)
			if err != nil {
				return &TaskResult{}, fmt.Errorf("Process Template :: %s", err.Error())
			}
			out, err := tpl.Execute(pongo2.Context{"vars": vars})
			if err != nil {
				return &TaskResult{}, fmt.Errorf("Process Template :: %s", err.Error())
			}
			tmpDir, srcFile := path.Split(srcPath)
			srcFile = srcFile + "_" + machine.Hostname
			tmpFileName := fmt.Sprintf(".%s", srcFile)
			tmpFile := path.Join(tmpDir, tmpFileName)

			err = ioutil.WriteFile(tmpFile, []byte(out), 0644)
			if err != nil {
				return &TaskResult{}, fmt.Errorf("Process Template :: %s", err.Error())
			}
			moduleParams["srcOrig"] = srcPath
			moduleParams["src"] = tmpFile
		case "reset_src":
			moduleParams["src"] = moduleParams["srcOrig"]
		}
	}

	// Set to status to ignored if the result is a failure
	if task.IgnoreErrors && (taskResult.State != "ok") {
		taskResult.State = "ignored"
	}

	if task.Register != "" {
		registerMap[task.Register] = taskResult
	}

	return &taskResult, nil
}
