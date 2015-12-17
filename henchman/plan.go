package henchman

import (
	"archive/tar"
	"fmt"
	"os"
	"path/filepath"
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

// For plan stats.  Records the number of states for each machine
var planStats = map[string]map[string]int{}

type VarsMap map[interface{}]interface{}
type RegMap map[string]interface{}

type Plan struct {
	Name      string
	Inventory Inventory
	Vars      VarsMap
	Tasks     []*Task
}

func localhost() *Machine {
	tc := make(TransportConfig)
	local, _ := NewLocal(&tc)
	localhost := Machine{}
	localhost.Hostname = "127.0.0.1"
	localhost.Transport = local
	return &localhost
}

/**
 * These functions deal with plan stats and details
 */
func updatePlanStats(state string, hostname string) {
	if _, ok := planStats[hostname]; !ok {
		planStats[hostname] = map[string]int{}
	}

	planStats[hostname][state]++
}

func printPlanStats() {
	var str string
	for hostname, states := range planStats {
		str = SprintfAndFill(25, " ", "[ %s ]", hostname)
		str += "=> "
		for state, counter := range states {
			str += SprintfAndFill(20, " ", "%s: %d", state, counter)
		}
		fmt.Println(str)
	}
}

func printTaskResults(taskResult *TaskResult, task *Task) {
	resetCode := statuses["reset"]
	colorCode := statuses[taskResult.State]
	Printf("%s => %s\n\n", colorCode+taskResult.State, taskResult.Msg+resetCode)

	if task.Debug {
		Println("Output\n-----" +
			colorCode +
			printRecurse(taskResult.Output, "", "\n") +
			resetCode)
	}
}

/**
 * These functions are helpers of plan.Setup
 */
// transfers the modules.tar to each machine, untars, and removes the tar file
func transferAndUntarModules(machine *Machine, remoteModDir string) error {
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
	if err := machine.Transport.Put(TARGET, remoteModDir, "file"); err != nil {
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

// Creates and populates modules.tar
func createModulesTar(isLocal *bool, tasks []*Task) error {
	// initialize set to hold module names
	modSet := make(map[string]bool)

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
			if _, _, err := task.Module.Resolve(); err != nil {
				return HenchErr(err, map[string]interface{}{
					"task": task.Name,
				}, "")
			}
			if task.Local == true {
				*isLocal = true
			}
			modSet[task.Module.Name] = false
		}
	}

	// tars all modules needed on remote machines
	// NOTE: maybe we gotta zip them too
	for _, modPath := range ModuleSearchPath {

		// add all modules in every search path
		for modName, added := range modSet {

			// if module has not been tarred add it
			if !added {
				filePath := filepath.Join(modPath, modName)
				_, err := os.Stat(filePath)

				// if module does not exists don't error out because it just doesn't
				// exist in this seach path
				if err == nil {
					if err := tarit(filePath, "", tarball); err != nil {
						return HenchErr(err, map[string]interface{}{
							"modPath": modPath,
						}, "While populating modules.tar")
					}
					// set module added to be true
					modSet[modName] = true
				}
			}
		}
	}

	return nil
}

/**
 * These functions are functions that can be utilized by plans
 */
// Moves all modules to each host
func (plan *Plan) Setup(machines []*Machine) error {
	Info(map[string]interface{}{
		"plan":         plan.Name,
		"num machines": len(machines),
	}, "Setting up plan")
	PrintfAndFill(75, "~", "SETTING UP PLAN [ %s ] ", plan.Name)
	fmt.Println("Creating modules.tar")

	// creates and populates modules.tar
	var isLocal bool
	if err := createModulesTar(&isLocal, plan.Tasks); err != nil {
		return HenchErr(err, map[string]interface{}{
			"plan": plan.Name,
		}, "While creating modules.tar")
	}

	fmt.Println("Transferring modules to all systems...")
	// transport modules.tar to all machines
	remoteModDir := "${HOME}/.henchman/"
	for _, machine := range machines {
		if err := transferAndUntarModules(machine, remoteModDir); err != nil {
			return HenchErr(err, map[string]interface{}{
				"plan": plan.Name,
				"host": machine.Hostname,
			}, "While transferring modules.tar")
		}
		fmt.Printf("Transferred to [ %s ]\n", machine.Hostname)
	}

	if isLocal == true {
		if err := transferAndUntarModules(localhost(), remoteModDir); err != nil {
			return HenchErr(err, map[string]interface{}{
				"plan": plan.Name,
				"host": "127.0.0.1",
			}, "While transferring modules.tar")
		}
		fmt.Println("Transferred to [ 127.0.0.1 ]")
	}

	// remove unnecessary modules.tar
	os.Remove("modules.tar")

	Info(map[string]interface{}{
		"plan":         plan.Name,
		"num machines": len(machines),
	}, "Done setting up plan")
	fmt.Printf("Setup complete\n\n")

	return nil
}

// For now it just removes the .henchman folder in each system
func (plan *Plan) Cleanup(machines []*Machine) error {
	remoteHenchmanDir := "${HOME}/.henchman/"
	for _, machine := range machines {
		if _, err := machine.Transport.Exec(fmt.Sprintf("rm -rf %s", remoteHenchmanDir),
			nil, false); err != nil {
			return HenchErr(err, map[string]interface{}{
				"remotePath": remoteHenchmanDir,
				"host":       machine.Hostname,
			}, "While removing .henchman")
		}
	}

	if _, err := localhost().Transport.Exec(fmt.Sprintf("rm -rf %s", remoteHenchmanDir),
		nil, false); err != nil {
		return HenchErr(err, map[string]interface{}{
			"remotePath": remoteHenchmanDir,
			"host":       "127.0.0.1",
		}, "While removing .henchman")
	}

	return nil
}

// Does execution of tasks
func (plan *Plan) Execute(machines []*Machine) error {
	local := localhost()

	Info(map[string]interface{}{
		"plan":         plan.Name,
		"num machines": len(machines),
	}, fmt.Sprintf("Executing plan '%s'", plan.Name))
	PrintfAndFill(75, "~", "EXECUTING PLAN [ %s ] ", plan.Name)

	wg := new(sync.WaitGroup)

	for _, _machine := range machines {
		machine := _machine
		wg.Add(1)

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
				MergeMap(machine.Vars, vars, true)
				task.Vars["current_hostname"] = actualMachine.Hostname
				MergeMap(task.Vars, vars, true)

				Debug(map[string]interface{}{
					"vars": vars,
					"plan": plan.Name,
					"task": task.Name,
					"host": actualMachine.Hostname,
				}, "Vars for Task")

				err := task.Render(vars, registerMap)

				if err != nil {
					henchErr := HenchErr(err, map[string]interface{}{
						"plan":  plan.Name,
						"task":  task.Name,
						"host":  actualMachine.Hostname,
						"error": err.Error(),
					}, "").(*HenchmanError)
					Fatal(henchErr.Fields, fmt.Sprintf("Error rendering task '%s'", task.Name))
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
					"plan": plan.Name,
				}, fmt.Sprintf("Starting Task '%s'", task.Name))

				// handles the retries
				var taskResult *TaskResult
				for numRuns := task.Retry + 1; numRuns > 0; numRuns-- {
					// If this is a retry print some info
					if numRuns <= task.Retry {
						Debug(map[string]interface{}{
							"task":      task.Name,
							"host":      actualMachine.Hostname,
							"plan":      plan.Name,
							"iteration": task.Retry + 1 - numRuns,
						}, fmt.Sprintf("Retrying Task '%s'", task.Name))
						PrintfAndFill(75, "~", "TASK FAILED. RETRYING [ %s | %s | %s ] ",
							actualMachine.Hostname, task.Name, task.Module.Name)
						printTaskResults(taskResult, task)
					}

					taskResult, err = task.Run(actualMachine, vars, registerMap)
					if err != nil {
						henchErr := HenchErr(err, map[string]interface{}{
							"plan":  plan.Name,
							"task":  task.Name,
							"host":  actualMachine.Hostname,
							"error": err.Error(),
						}, "").(*HenchmanError)
						Fatal(henchErr.Fields, fmt.Sprintf("Error running task '%s'", task.Name))
						return
						/*
							return HenchErr(err, log.Fields{
								"plan": plan.Name,
								"task": task.Name,
								"host": actualMachine.Hostname,
							}, "Error running task")
						*/
					}

					if taskResult.State == "ok" ||
						taskResult.State == "changed" {
						numRuns = 0
					}
				}

				// Fields for info
				fields := map[string]interface{}{
					"task":  task.Name,
					"host":  actualMachine.Hostname,
					"state": taskResult.State,
					"msg":   taskResult.Msg,
				}
				if task.Debug {
					fields["output"] = taskResult.Output
				}
				Info(fields, fmt.Sprintf("Task '%s' complete", task.Name))
				PrintfAndFill(75, "~", "TASK [ %s | %s | %s ] ",
					actualMachine.Hostname, task.Name, task.Module.Name)

				printTaskResults(taskResult, task)
				updatePlanStats(taskResult.State, actualMachine.Hostname)

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
	PrintfAndFill(75, "~", "PLAN STATS [ %s ] ", plan.Name)

	printPlanStats()
	return nil
}
