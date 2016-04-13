package henchman

import (
	"fmt"
	"sync"
)

type RollingDeploy struct {
	NumHosts float64
}

// ExecuteTasksOnMachines creates a go func to run the task set on a given slice of machines
func (rd RollingDeploy) ExecuteTasksOnMachines(machines []*Machine, plan *Plan) <-chan error {
	var wgMain, wgMachines sync.WaitGroup
	var taskError bool
	var err error

	errChan := make(chan error, 1)

	// case 0: num_hosts: 0 or 1 - make sure to validate to see if it's a whole number
	wgMain.Add(1)
	go func() {
		defer wgMain.Done()
		for ndx, machine := range machines {
			for _, task := range plan.Tasks {
				PrintfAndFill(75, "~", "\nTASK [ %s | %s ] ", task.Name, task.Module.Name)
				printShellModule(task)
				taskError, err = sd.executeTask(plan.registerMaps[ndx], machine, task, plan)
				if err != nil {
					errChan <- err
				}
				if taskError {
					return
				}
			}
		}
	}()

	// case 1: num_hosts: whole num greater than 0 or 1
	// case 2: num_hosts: percentage case.  It will be a num > 0 && num < 1

	go func() {
		wgMain.Wait()
		close(errChan)
	}()

	return errChan
}

// executeTasks sets up the final variable map, then renders and runs the task list
func (rd RollingDeploy) executeTasks(registerMap RegMap, machine *Machine, task *Task, plan *Plan) (bool, error) {
	var actualMachine *Machine
	if task.Local == true {
		actualMachine = localhost()
	} else {
		actualMachine = machine
	}

	vars := make(VarsMap)
	if err := task.SetupVars(plan, actualMachine, vars, registerMap); err != nil {
		return true, err
	}

	renderedTask, err := task.Render(vars, registerMap)
	if err != nil {
		return true, HenchErr(err, map[string]interface{}{
			"plan": plan.Name,
			"task": renderedTask.Name,
			"host": actualMachine.Hostname,
		}, fmt.Sprintf("Error rendering task '%s'", renderedTask.Name))
	}

	// accepted states are ok, success, ignored
	taskError, err := plan.ManageTaskRun(renderedTask, actualMachine, vars, registerMap)
	return taskError, err
}

// promptContinue asks the user if s/he wants to continue executing the plan
// if a task fails
func promptContinue() bool {
	fmt.Println("Task failed. Continue? (y/n)")

	var ans string
	fmt.Scanf("%s\n", &ans)
	for ans != "y" && ans != "Y" {
		if ans == "n" || ans == "N" {
			return false
		} else {
			fmt.Println("Invalid answer.")
			fmt.Scanf("%s\n", &ans)
		}
	}

	return true
}
