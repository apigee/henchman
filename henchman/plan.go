package henchman

import (
	"archive/tar"
	"errors"
	"fmt"
	"os"
	_ "path/filepath"
	_ "reflect"
	"strings"
	_ "sync"

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

// NOTE: eventually change this to map[string]interface{}
type VarsMap map[string]interface{}
type RegMap map[string]interface{}

type Plan struct {
	Name      string
	Inventory Inventory
	Deploy    DeployInterface
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

// NOTE: This function in addition to printing stats also figures out
// if there were any errors or failures when executing the plan. A boolean true
// is returned in that case
func printPlanStats() (taskError bool) {
	var str string
	taskError = false
	for hostname, states := range planStats {
		str = SprintfAndFill(25, " ", "[ %s ]", hostname)
		str += "=> "
		for state, counter := range states {
			if state == "failure" || state == "error" {
				taskError = true
			}
			str += SprintfAndFill(20, " ", "%s: %d", state, counter)
		}
		fmt.Println(str)
	}
	return
}

func printTaskResults(taskResult *TaskResult, task *Task, hostname string, retry int) {
	resetCode := statuses["reset"]
	colorCode := statuses[taskResult.State]

	if retry == 0 {
		Printf("%s %s => %s\n",
			SprintfAndFill(20, " ", "[ %s ]", hostname),
			colorCode+taskResult.State,
			taskResult.Msg+resetCode)
	} else {
		Printf("%s %s => %s\n",
			SprintfAndFill(20, " ", "[ %s | RETRY %v ]", hostname, retry),
			colorCode+taskResult.State,
			taskResult.Msg+resetCode)
	}

	if task.Debug {
		Println("------\nOutput" +
			colorCode +
			printRecurse(taskResult.Output, "", "\n") +
			resetCode)
	}
}

func printShellModule(task *Task) {
	if task.Module.Name == "shell" {
		if _, present := task.Module.Params["env"]; present {
			PrintfAndFill(75, "~", "SHELL [ cmd => %v | env => %v ]", task.Module.Params["cmd"], task.Module.Params["env"])
		} else {
			PrintfAndFill(75, "~", "SHELL [ cmd => %v ]", task.Module.Params["cmd"])
		}
	}
}

/**
 * These functions are helpers of plan.Setup
 */
// transfers the modules.tar to each machine, untars, and removes the tar file
func transferAndUntarModules(machine *Machine) error {
	// create dir
	if _, err := machine.Transport.Exec(fmt.Sprintf("mkdir -p %s", REMOTE_DIR),
		nil, false); err != nil {
		return HenchErr(err, nil, "While creating dir")
	}

	// gets the name of the proper module tar
	modulesTar, err := getModuleTar(machine)
	if err != nil {
		return HenchErr(err, nil, "While getting system info")
	}

	// transfer tar module
	if err := machine.Transport.Put(modulesTar, REMOTE_DIR, "file"); err != nil {
		return HenchErr(err, nil, "While transfering tar")
	}

	// untar the modules
	cmd := fmt.Sprintf("tar -xvf %s -C %s", REMOTE_DIR+modulesTar, REMOTE_DIR)
	if _, err := machine.Transport.Exec(cmd, nil, false); err != nil {
		return HenchErr(err, nil, "While untarring")
	}

	// remove tar file
	cmd = fmt.Sprintf("/bin/rm %s", REMOTE_DIR+modulesTar)
	if _, err := machine.Transport.Exec(cmd, nil, false); err != nil {
		return HenchErr(err, nil, "While removing tar in remote path")
	}

	return nil
}

// getOsName returns the os name of the machine
func getOsName(machine *Machine) (string, error) {
	bytesBuf, err := machine.Transport.Exec("uname -a", nil, false)
	if err != nil {
		return "", err
	}

	osName := strings.ToLower(strings.Split(bytesBuf.String(), " ")[0])
	return osName, nil
}

// getModuleTar returns the name of the module tar file based off the system's os
func getModuleTar(machine *Machine) (string, error) {
	osName, err := getOsName(machine)
	if err != nil {
		return "", err
	}

	return osName + "_" + MODULES_TARGET, nil
}

// Creates and populates modules.tar
func createModulesTar(tasks []*Task, osName string) error {
	// initialize set to hold module names and paths
	modSet := make(map[string]string)

	// os.Create will O_TRUNC the file if it exists
	tarfile, err := os.Create(osName + "_" + MODULES_TARGET)
	if err != nil {
		return HenchErr(err, map[string]interface{}{
			"target": osName + "_" + MODULES_TARGET,
		}, "")
	}
	tarball := tar.NewWriter(tarfile)
	defer tarfile.Close()
	defer tarball.Close()

	// gather all modules needed and verify they exist
	// NOTE: just transfer everything to local
	for _, task := range tasks {
		if _, ok := modSet[task.Module.Name]; !ok {
			modulePath, _, err := task.Module.Resolve(osName)
			if err != nil {
				return HenchErr(err, map[string]interface{}{
					"task": task.Name,
				}, "")
			}
			modSet[task.Module.Name] = modulePath
		}
	}

	// tars all modules needed on remote machines
	// NOTE: maybe we gotta zip them too
	// add all modules in every search path
	for _, modPath := range modSet {
		if err := tarit(modPath, "", tarball); err != nil {
			return HenchErr(err, map[string]interface{}{
				"modPath": modPath,
			}, "While populating modules.tar")
		}
	}

	return nil
}

/**
 * These functions are functions that can be utilized by plans
 */
// Moves all modules to each host.
// If machines only has localhost, ignore this activity since we anyway do that later
func (plan *Plan) Setup(machines []*Machine) error {
	if len(machines) == 0 {
		return HenchErr(fmt.Errorf("This has no machines to execute on"), map[string]interface{}{
			"plan":     plan.Name,
			"solution": "Check if inventory is valid",
		}, "")
	}

	Info(map[string]interface{}{
		"plan":         plan.Name,
		"num machines": len(machines),
	}, "Setting up plan")
	PrintfAndFill(75, "~", "SETTING UP PLAN [ %s ] ", plan.Name)
	fmt.Println("Creating modules.tar")

	// creates and populates modules.tar
	for _, osName := range OsNames {
		if err := createModulesTar(plan.Tasks, osName); err != nil {
			return HenchErr(err, map[string]interface{}{
				"plan": plan.Name,
			}, "While creating modules.tar")
		}
	}

	Println("Transferring modules to all systems...")
	// transport modules.tar to all machines
	for _, machine := range machines {
		if machine.Hostname == "localhost" {
			continue
		}
		if err := transferAndUntarModules(machine); err != nil {
			return HenchErr(err, map[string]interface{}{
				"plan":       plan.Name,
				"remotePath": REMOTE_DIR,
				"host":       machine.Hostname,
			}, "While transferring modules.tar")
		}
		Printf("Transferred to [ %s ]\n", machine.Hostname)
	}
	if err := transferAndUntarModules(localhost()); err != nil {
		return HenchErr(err, map[string]interface{}{
			"plan": plan.Name,
			"host": "127.0.0.1",
		}, "While transferring modules.tar")
	}
	Println("Transferred to [ 127.0.0.1 ]")

	// remove unnecessary modules.tar
	for _, osName := range OsNames {
		os.Remove(osName + "_" + MODULES_TARGET)
	}

	Info(map[string]interface{}{
		"plan":         plan.Name,
		"num machines": len(machines),
	}, "Done setting up plan")
	fmt.Printf("Setup complete\n\n")

	return nil
}

// For now it just removes the .henchman folder in each system
func (plan *Plan) Cleanup(machines []*Machine) error {
	for _, machine := range machines {
		if _, err := machine.Transport.Exec(fmt.Sprintf("rm -rf %s", REMOTE_DIR),
			nil, false); err != nil {
			return HenchErr(err, map[string]interface{}{
				"remotePath": REMOTE_DIR,
				"host":       machine.Hostname,
			}, "While removing .henchman")
		}
	}

	if _, err := localhost().Transport.Exec(fmt.Sprintf("rm -rf %s", REMOTE_DIR),
		nil, false); err != nil {
		return HenchErr(err, map[string]interface{}{
			"remotePath": REMOTE_DIR,
			"host":       "127.0.0.1",
		}, "While removing .henchman")
	}

	return nil
}

// Does execution of tasks
func (plan *Plan) Execute(machines []*Machine) error {
	Info(map[string]interface{}{
		"plan":         plan.Name,
		"num machines": len(machines),
	}, fmt.Sprintf("Executing plan '%s'", plan.Name))

	PrintfAndFill(75, "~", "EXECUTING PLAN [ %s ] ", plan.Name)

	/*
		machineChans := []<-chan error{}
		for _, machine := range machines {
			machineChans = append(machineChans, Deploy.ExecuteTasks(machine, plan))
		}

		errorsChan := mergeErrs(machineChans)
	*/
	errorsChan := plan.Deploy.ExecuteTasksOnMachines(machines, plan)
	err := <-errorsChan
	if err != nil {
		return err
	}

	Info(map[string]interface{}{
		"plan":         plan.Name,
		"num machines": len(machines),
	}, "Plan Complete")
	PrintfAndFill(75, "~", "PLAN STATS [ %s ] ", plan.Name)

	taskError := printPlanStats()
	if taskError {
		return errors.New("One of the tasks failed or error'ed out")
	}

	return nil
}

// Manages all the print outs and updating of planStats for running a task
// FIXME: shouldn't be making this a plan function just b/c we need plan.Name...
func (plan Plan) ManageTaskRun(task *Task, machine *Machine, vars VarsMap, registerMap RegMap) (bool, error) {
	Info(map[string]interface{}{
		"task": task.Name,
		"host": machine.Hostname,
		"plan": plan.Name,
	}, fmt.Sprintf("Starting Task '%s'", task.Name))

	// handles the running and retrying of tasks
	taskResult, err := taskRunAndRetries(task, machine, vars, registerMap)
	if err != nil {
		return false, HenchErr(err, map[string]interface{}{
			"plan": plan.Name,
		}, "")
	}

	// Fields for info
	fields := map[string]interface{}{
		"task":  task.Name,
		"host":  machine.Hostname,
		"state": taskResult.State,
		"msg":   taskResult.Msg,
	}
	if task.Debug {
		fields["output"] = taskResult.Output
	}
	Info(fields, fmt.Sprintf("Task '%s' complete on '%s'", task.Name, machine.Hostname))

	updatePlanStats(taskResult.State, machine.Hostname)

	// NOTE: if IgnoreErrors is true then state will be set to ignored in task.Run(...)
	if taskResult.State == "error" || taskResult.State == "failure" {
		return false, nil
	}

	return true, nil
}

// Runs the task and the retries
func taskRunAndRetries(task *Task, machine *Machine, vars VarsMap, registerMap RegMap) (*TaskResult, error) {
	var err error
	var taskResult *TaskResult
	for numRuns := task.Retry + 1; numRuns > 0; numRuns-- {
		// If this is a retry print some info
		if numRuns <= task.Retry {
			Debug(map[string]interface{}{
				"task":      task.Name,
				"host":      machine.Hostname,
				"mod":       task.Module.Name,
				"iteration": task.Retry + 1 - numRuns,
			}, fmt.Sprintf("Retrying Task '%s'", task.Name))
		}

		taskResult, err = task.Run(machine, vars, registerMap)
		if err != nil {
			return nil, HenchErr(err, map[string]interface{}{
				"task": task.Name,
				"mod":  task.Module.Name,
				"host": machine.Hostname,
			}, fmt.Sprintf("Error running task '%s'", task.Name))
		}
		printTaskResults(taskResult, task, machine.Hostname, task.Retry-numRuns+1)

		if taskResult.State == "ok" ||
			taskResult.State == "changed" ||
			taskResult.State == "skipped" {
			break
		}
	}

	return taskResult, nil
}
