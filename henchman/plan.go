package henchman

import (
	"archive/tar"
	"fmt"
	"os"
	_ "reflect"
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
const TARGET = "modules.tar"

func localhost() *Machine {
	tc := make(TransportConfig)
	local, _ := NewLocal(&tc)
	localhost := Machine{}
	localhost.Hostname = "127.0.0.1"
	localhost.Transport = local
	return &localhost
}

// transfers the modules.tar to each machine, untars, and removes the tar file
func transferUntarModules(machine *Machine, remoteModDir string) error {
	// create dir
	if _, err := machine.Transport.Exec(fmt.Sprintf("mkdir -p %s", remoteModDir),
		nil, false); err != nil {
		return HenchErr(err, map[string]interface{}{
			"remotePath": remoteModDir,
			"host":       machine.Hostname,
		}, "While creating dir")
	}

	// throw a check the check sum stuff in here somewhere
	// transfer tar module
	if err := machine.Transport.Put(TARGET, remoteModDir, "dir"); err != nil {
		return HenchErr(err, map[string]interface{}{
			"remotePath": remoteModDir,
			"host":       machine.Hostname,
		}, "While transfering tar")
	}

	// untar the modules
	cmd := fmt.Sprintf("tar -xvf %s -C %s", remoteModDir+TARGET, remoteModDir)
	if _, err := machine.Transport.Exec(cmd, nil, false); err != nil {
		return HenchErr(err, map[string]interface{}{
			"remotePath": remoteModDir,
			"host":       machine.Hostname,
		}, "While untarring")
	}

	// remove tar file
	cmd = fmt.Sprintf("/bin/rm %s", remoteModDir+TARGET)
	if _, err := machine.Transport.Exec(cmd, nil, false); err != nil {
		return HenchErr(err, map[string]interface{}{
			"remotePath": remoteModDir,
			"host":       machine.Hostname,
		}, "While removing tar in remote path")
	}

	return nil
}

// Tars modules into modules.tar
func tarModule(modName string, tarball *tar.Writer) error {
	info, _ := os.Stat(modName)
	if info.IsDir() {
		if err := tarDir(modName, tarball); err != nil {
			return HenchErr(err, map[string]interface{}{
				"module": modName,
			}, "While Tarring Dir")
		}
	} else {
		if err := tarFile(modName, tarball); err != nil {
			return HenchErr(err, map[string]interface{}{
				"module": modName,
			}, "While Tarring file")
		}
	}

	return nil
}

// Creates and populates modules.tar
func createModulesTar(tasks []*Task) error {
	// initialize set to hold module names
	modSet := make(map[string]bool)

	// get the curdir and move to location of modules
	curDir, err := os.Getwd()
	if err != nil {
		return err
	}

	// os.Create will O_TRUNC the file if it exists
	tarfile, err := os.Create(TARGET)
	if err != nil {
		return HenchErr(err, map[string]interface{}{
			"target": TARGET,
		}, "")
	}
	tarball := tar.NewWriter(tarfile)
	defer tarfile.Close()
	defer tarball.Close()

	// gather all modules needed and verify they exist
	// NOTE: just transfer everything to local
	for _, task := range tasks {
		if _, ok := modSet[task.Module.Name]; !ok {
			if _, err := task.Module.Resolve(); err != nil {
				return HenchErr(err, map[string]interface{}{
					"task": task.Name,
				}, "")
			}
			modSet[task.Module.Name] = false
		}
	}

	// tars all modules needed on remote machines
	// NOTE: maybe we gotta zip them too
	for _, modPath := range ModuleSearchPath {

		//change to mod path
		os.Chdir(modPath)

		// add all modules in every search path
		for modName, added := range modSet {

			// if module has not been tarred add it
			if !added {
				_, err := os.Stat(modName)

				// if module does not exists don't error out because it just doesn't
				// exist in this seach path
				if err == nil {
					if err := tarModule(modName, tarball); err != nil {
						return HenchErr(err, map[string]interface{}{
							"modPath": modPath,
						}, "While populating modules.tar")
					}

					// set module added to be true
					modSet[modName] = true
				}
			}
		}

		// go back to dir where modules.tar is
		os.Chdir(curDir)
	}

	return nil
}

// Moves all modules to each host
func (plan *Plan) Setup(machines []*Machine) error {
	Info(map[string]interface{}{
		"plan":         plan.Name,
		"num machines": len(machines),
	}, "Setting up plan")

	Debug(map[string]interface{}{
		"plan":         plan.Name,
		"num machines": len(machines),
	}, "Creating modules.tar")

	// creates and populates modules.tar
	if err := createModulesTar(plan.Tasks); err != nil {
		return HenchErr(err, map[string]interface{}{
			"plan": plan.Name,
		}, "While creating modules.tar")
	}

	Debug(map[string]interface{}{
		"plan":         plan.Name,
		"num machines": len(machines),
	}, "Finished creating modules.tar")

	Debug(map[string]interface{}{
		"plan":         plan.Name,
		"num machines": len(machines),
	}, "Transporting modules.tar")

	// transport modules.tar to all machines
	remoteModDir := "${HOME}/.henchman/"
	for _, machine := range machines {
		if err := transferUntarModules(machine, remoteModDir); err != nil {
			return HenchErr(err, map[string]interface{}{
				"plan": plan.Name,
			}, "While transferring modules.tar")
		}
	}
	if err := transferUntarModules(localhost(), remoteModDir); err != nil {
		return HenchErr(err, map[string]interface{}{
			"plan": plan.Name,
		}, "While trasnferring modules.tar")
	}

	Debug(map[string]interface{}{
		"plan":         plan.Name,
		"num machines": len(machines),
	}, "Finished transporting modules.tar")

	// remove unnecessary modules.tar
	os.Remove("modules.tar")

	Info(map[string]interface{}{
		"plan":         plan.Name,
		"num machines": len(machines),
	}, "Done setting up plan")

	return nil
}

// Does execution of tasks
func (plan *Plan) Execute(machines []*Machine) error {
	local := localhost()

	Info(map[string]interface{}{
		"plan":         plan.Name,
		"num machines": len(machines),
	}, "Executing plan")

	resetCode := statuses["reset"]
	wg := new(sync.WaitGroup)
	for _, _machine := range machines {
		machine := _machine
		wg.Add(1)
		//		machineVars := plan.Inventory.Groups[machine.Group].Vars
		// NOTE: need individual registerMap for each machine
		registerMap := make(RegMap)
		// NOTE: returning errors requires channels.
		// FIXME: create channels for stuff m8
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
				Debug(map[string]interface{}{"vars": printRecurse(vars, "", "\n")}, "vars map")
				MergeMap(machine.Vars, vars, true)
				Debug(map[string]interface{}{"vars": printRecurse(machine.Vars, "", "\n")}, "machine vars map")

				task.Vars["current_host"] = actualMachine.Hostname
				MergeMap(task.Vars, vars, true)

				err := task.Render(vars, registerMap)

				if err != nil {
					henchErr := HenchErr(err, map[string]interface{}{
						"plan":  plan.Name,
						"task":  task.Name,
						"host":  actualMachine.Hostname,
						"error": err.Error(),
					}, "").(*HenchmanError)
					Fatal(henchErr.Fields, "Error rendering task")
					return
					/*
						return HenchErr(err, log.Fields{
							"plan": plan.Name,
							"task": task.Name,
							"host": actualMachine.Hostname,
						}, "Error rendering task")
					*/
				}

				Info(map[string]interface{}{
					"task": task.Name,
					"host": actualMachine.Hostname,
				}, "Starting Task")

				// handles the retries
				var taskResult TaskResult
				for numRuns := task.Retry + 1; numRuns > 0; numRuns-- {
					taskResult, err := task.Run(actualMachine, vars, registerMap)
					if err != nil {
						henchErr := HenchErr(err, map[string]interface{}{
							"plan":  plan.Name,
							"task":  task.Name,
							"host":  actualMachine.Hostname,
							"error": err.Error(),
						}, "").(*HenchmanError)
						Fatal(henchErr.Fields, "Error running task")
						return
						/*
							return HenchErr(err, log.Fields{
								"plan": plan.Name,
								"task": task.Name,
								"host": actualMachine.Hostname,
							}, "Error running task")
						*/
					}

					if taskResult.State != "error" ||
						taskResult.State != "ignored" ||
						taskResult.State != "failure" {
						numRuns = 0
					}
				}

				colorCode := statuses[taskResult.State]

				//NOTE: make a color code create function
				fields := map[string]interface{}{
					"task":  task.Name,
					"host":  actualMachine.Hostname,
					"state": colorCode + taskResult.State + resetCode,
					"msg":   taskResult.Msg,
				}

				if task.Debug {
					fields["output"] = printRecurse(taskResult.Output, "", "\n")
				}

				Info(fields, "Task Complete")

				// NOTE: if IgnoreErrors is true then state will be set to ignored in task.Run(...)
				if taskResult.State == "error" || taskResult.State == "failure" {
					break
				}
			}
		}()
	}
	wg.Wait()

	Info(map[string]interface{}{
		"plan":         plan.Name,
		"num machines": len(machines),
	}, "Plan Complete")
	return nil
}
