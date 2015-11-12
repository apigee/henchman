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

func transferUntarModules(machine *Machine, remoteModDir string, target string) {
	// create dir
	if _, err := machine.Transport.Exec(fmt.Sprintf("mkdir -p %s", remoteModDir),
		nil, false); err != nil {
		log.WithFields(log.Fields{
			"process":    "Creating Remote Module Dir",
			"remotePath": remoteModDir,
			"error":      err.Error(),
			"host":       machine.Hostname,
		}).Fatal("Error in plan setup")
	}

	// throw a check the check sum stuff in here somewhere
	// transfer tar module
	if err := machine.Transport.Put(target, remoteModDir, "dir"); err != nil {
		log.WithFields(log.Fields{
			"process":    "Putting tar module in remote path",
			"remotePath": remoteModDir,
			"error":      err.Error(),
			"host":       machine.Hostname,
		}).Fatal("Error in plan setup")
	}

	// untar the modules
	cmd := fmt.Sprintf("tar -xvf %s -C %s", remoteModDir+target, remoteModDir)
	if _, err := machine.Transport.Exec(cmd, nil, false); err != nil {
		log.WithFields(log.Fields{
			"process":    "Untarring module in remote path",
			"remotePath": remoteModDir,
			"error":      err.Error(),
			"host":       machine.Hostname,
		}).Fatal("Error in plan setup")
	}

	// remove tar file
	cmd = fmt.Sprintf("/bin/rm %s", remoteModDir+target)
	if _, err := machine.Transport.Exec(cmd, nil, false); err != nil {
		log.WithFields(log.Fields{
			"process":    "Removing tar file in remote path",
			"remotePath": remoteModDir,
			"error":      err.Error(),
			"host":       machine.Hostname,
		}).Fatal("Error in plan setup")
	}
}

// Moves all modules to each host
func (plan *Plan) Setup(machines []*Machine) {
	log.WithFields(log.Fields{
		"plan":         plan.Name,
		"num machines": len(machines),
	}).Info("Setting up plan")

	// initialize set to hold module names
	modSet := make(map[string]bool)

	// get the curdir and move to location of modules
	curDir, err := os.Getwd()
	if err != nil {
		log.WithFields(log.Fields{
			"process": "os.Getwd()",
			"error":   err.Error(),
		}).Fatal("Error in plan setup")
	}

	// create the tar file to be filled
	// create the writer to tar file
	target := "modules.tar"
	tarfile, err := os.Create(target)
	if err != nil {
		log.WithFields(log.Fields{
			"process": "creating target file modules.tar",
			"error":   err.Error(),
		}).Fatal("Error in plan setup")
	}
	tarball := tar.NewWriter(tarfile)

	// gather all modules needed and verify they exist
	// NOTE: just transfer everything to local
	for _, task := range plan.Tasks {
		/*
			if _, ok := localSet[task.Module.Name]; !ok && task.Local {
				localSet[task.Module.Name] = struct{}{}
			} else if _, ok := modSet[task.Module.Name]; !ok {
		*/
		if _, ok := modSet[task.Module.Name]; !ok {
			if _, err := task.Module.Resolve(); err != nil {
				log.Fatal(err.Error())
			}
			modSet[task.Module.Name] = false
		}
	}

	// tars all modules needed on remote machines
	// NOTE: maybe we gotta zip them too
	for _, modPath := range ModuleSearchPath {

		// change to mod path
		os.Chdir(modPath)

		// add all modules in every search path
		for modName, added := range modSet {

			// if module has not been tarred add it
			if !added {
				info, err := os.Stat(modName)
				if err != nil {
					Debug(log.Fields{
						"process": "getting module info",
						"modPath": modPath,
						"module":  modName,
						"error":   err.Error(),
					}, "Module not found")
				} else {
					if info.IsDir() {
						if err = tarDir(modName, tarball); err != nil {
							log.WithFields(log.Fields{
								"process": "tarring dir",
								"module":  modName,
								"error":   err.Error(),
							}).Fatal("Error in plan setup")
						}
					} else {
						if err = tarFile(modName, tarball); err != nil {
							log.WithFields(log.Fields{
								"process": "tarring file",
								"module":  modName,
								"error":   err.Error(),
							}).Fatal("Error in plan setup")
						}
					}

					// set module added to be true
					modSet[modName] = true
				}
			}
		}

		// go back to dir where modules.tar is
		os.Chdir(curDir)
	}

	// don't defer closing it will ruin the .tar file
	tarball.Close()
	tarfile.Close()

	// transport modules.tar to all machines
	remoteModDir := "${HOME}/.henchman/"
	for _, machine := range machines {
		transferUntarModules(machine, remoteModDir, target)
	}
	transferUntarModules(localhost(), remoteModDir, target)

	// remove unnecessary modules.tar
	os.Remove("modules.tar")
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
				fields := log.Fields{
					"task":  task.Name,
					"host":  actualMachine.Hostname,
					"state": colorCode + taskResult.State + resetCode,
					"msg":   taskResult.Msg,
				}

				if task.Debug {
					fields["output"] = printRecurse(taskResult.Output, "", "\n")
				}

				log.WithFields(fields).Info("Task Complete")

				// print only when --debug is on
				/*
					Debug(log.Fields{
						"task":   task.Name,
						"host":   actualMachine.Hostname,
						"output": printRecurse(taskResult.Output, "", "\n"),
					}, "Task Output")
				*/

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
