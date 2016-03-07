package henchman

import (
	"fmt"
	"sync"
)

type StandardDeploy struct{}

func (sd StandardDeploy) ExecuteTasksOnMachines(machines []*Machine, plan *Plan) <-chan error {
	var wgMain, wgMachines sync.WaitGroup
	var taskError bool
	var err error

	errChan := make(chan error, 1)

	wgMain.Add(1)
	go func() {
		defer wgMain.Done()
		for _, task := range plan.Tasks {
			PrintfAndFill(75, "~", "\nTASK [ %s | %s ] ", task.Name, task.Module.Name)
			printShellModule(task)
			for ndx, machine := range machines {
				wgMachines.Add(1)
				go func(m *Machine, t *Task, ndx int) {
					defer wgMachines.Done()
					taskError, err = sd.executeTask(plan.registerMaps[ndx], m, t, plan)
					if err != nil {
						errChan <- err
					}
				}(machine, task, ndx)
			}
			wgMachines.Wait()
			if taskError {
				return
			}
		}
	}()

	go func() {
		wgMain.Wait()
		close(errChan)
	}()

	return errChan
}

// Uses plans ManageTaskRun(...)
func (sd StandardDeploy) executeTask(registerMap RegMap, machine *Machine, task *Task, plan *Plan) (bool, error) {
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
