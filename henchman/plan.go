package henchman

import (
	"archive/tar"
	"fmt"
	log "gopkg.in/Sirupsen/logrus.v0"
	"os"
	//"reflect"
	"sync"

	"github.com/mgutz/ansi"
)

// For ANSI color codes
var statuses = map[string]string{
	"reset":   ansi.ColorCode("reset"),
	"ok":      ansi.ColorCode("green"),
	"changed": ansi.ColorCode("yellow"),
	"failure": ansi.ColorCode("red"),
	"error":   ansi.ColorCode("red"),
	"ignored": ansi.ColorCode("cyan"),
}

type VarsMap map[interface{}]interface{}
type RegMap map[string]interface{}

type Plan struct {
	Name      string
	Inventory Inventory
	Vars      VarsMap
	Tasks     []*Task
}

const HENCHMAN_PREFIX = "henchman_"

func localhost() *Machine {
	tc := make(TransportConfig)
	local, _ := NewLocal(&tc)
	localhost := Machine{}
	localhost.Hostname = "127.0.0.1"
	localhost.Transport = local
	return &localhost
}

// Moves all modules to each host
func (plan *Plan) Setup(machines []*Machine) error {
	// initialize two sets to hold module names
	modSet := make(map[string]struct{})
	localSet := make(map[string]struct{})

	// get the curdir and move to location of modules
	// NOTE:  make sure to check every search path
	curDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("Plan Setup :: GetWd :: %s", err.Error())
	}
	os.Chdir("modules")

	// gather all modules needed
	// NOTE: just transfer everything to local too?
	for _, task := range plan.Tasks {
		if _, ok := localSet[task.Module.Name]; !ok && task.Local {
			localSet[task.Module.Name] = struct{}{}
		} else if _, ok := modSet[task.Module.Name]; !ok {
			modSet[task.Module.Name] = struct{}{}
		}
	}

	// create the tar file to be filled
	// create the writer to tar file
	target := "modules.tar"
	tarfile, err := os.Create(target)
	if err != nil {
		return err
	}
	tarball := tar.NewWriter(tarfile)

	// tars all modules needed
	// NOTE: maybe we gotta zip them too
	for key, _ := range modSet {
		info, err := os.Stat(key)
		if err != nil {
			return fmt.Errorf("%s :: %s", key, err.Error())
		}

		if info.IsDir() {
			if err = tarDir(key, tarball); err != nil {
				return fmt.Errorf("Plan Setup :: %s", err.Error())
			}
		} else {
			if err = tarFile(key, tarball); err != nil {
				return fmt.Errorf("Plan Setup :: %s", err.Error())
			}
		}
	}

	// don't defer closing it will ruin the .tar file
	tarball.Close()
	tarfile.Close()

	// transport tar to all machines
	// first create the dir
	// then check if md5sum is present
	// compare md5sums
	// then repeat biatch
	remoteModDir := "${HOME}/.henchman/"
	for _, machine := range machines {
		// create dir
		if _, err = machine.Transport.Exec(fmt.Sprintf("mkdir -p %s", remoteModDir),
			nil, false); err != nil {
			return fmt.Errorf("Plan Setup :: Creating Mod Path :: %s", err.Error())
		}
		// throw a check the check sum crap in here somewhere
		// transfer tar module
		if err = machine.Transport.Put(target, remoteModDir, "dir"); err != nil {
			return fmt.Errorf("Plan Setup :: Putting Tar Module :: %s", err.Error())
		}

		// untar the modules
		cmd := fmt.Sprintf("tar -xvf %s -C %s", remoteModDir+target, remoteModDir)
		if _, err := machine.Transport.Exec(cmd, nil, false); err != nil {
			return fmt.Errorf("Plan Setup[%s] :: Untar Module :: %s", machine.Hostname, err.Error())
		}

		// remove tar file
		cmd = fmt.Sprintf("/bin/rm %s", remoteModDir+target)
		if _, err = machine.Transport.Exec(cmd, nil, false); err != nil {
			return fmt.Errorf("Plan Setup :: Removing Tar :: %s", err.Error())
		}
	}

	os.Remove("modules.tar")
	os.Chdir(curDir)

	return nil
}

func (plan *Plan) Execute(machines []*Machine) error {
	local := localhost()

	log.WithFields(log.Fields{
		"plan":         plan.Name,
		"num machines": len(machines),
	}).Info("Executing plan")

	resetCode := statuses["reset"]
	wg := new(sync.WaitGroup)
	for _, _machine := range machines {
		machine := _machine
		wg.Add(1)
		//		machineVars := plan.Inventory.Groups[machine.Group].Vars
		// NOTE: need individual registerMap for each machine
		registerMap := make(RegMap)
		go func() {
			defer wg.Done()
			var actualMachine *Machine
			for _, task := range plan.Tasks {
				// copy of task.Vars. It'll be different for each machine
				if task.Local == true {
					actualMachine = local
				} else {
					actualMachine = machine
				}

				vars := make(VarsMap)
				MergeMap(plan.Vars, vars, true)
				MergeMap(machine.Vars, vars, true)

				task.Vars["current_host"] = actualMachine.Hostname
				MergeMap(task.Vars, vars, true)

				err := task.Render(vars, registerMap)

				if err != nil {
					log.WithFields(log.Fields{
						"task":  task.Name,
						"host":  actualMachine.Hostname,
						"error": err.Error(),
					}).Error("Error Rendering Task")

					return
				}

				log.WithFields(log.Fields{
					"task": task.Name,
					"host": actualMachine.Hostname,
				}).Info("Starting Task")

				taskResult, err := task.Run(actualMachine, vars, registerMap)
				if err != nil {
					log.WithFields(log.Fields{
						"task":  task.Name,
						"host":  actualMachine.Hostname,
						"error": err.Error(),
					}).Error("Error Running Task")

					return
				}

				colorCode := statuses[taskResult.State]

				//NOTE: make a color code create function
				log.WithFields(log.Fields{
					"task":  task.Name,
					"host":  actualMachine.Hostname,
					"state": colorCode + taskResult.State + resetCode,
					"msg":   taskResult.Msg,
				}).Info("Task Complete")

				// print only when --debug is on
				Debug(log.Fields{
					"task":   task.Name,
					"host":   actualMachine.Hostname,
					"output": printRecurse(taskResult.Output, "", "\n"),
				}, "Task Output")

				if (taskResult.State == "error" || taskResult.State == "failure") && (!task.IgnoreErrors) {
					break
				}
			}
		}()
	}
	wg.Wait()

	log.WithFields(log.Fields{
		"plan":         plan.Name,
		"num machines": len(machines),
	}).Info("Plan Complete")
	return nil
}
