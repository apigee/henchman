package henchman

import (
	"fmt"
	"sync"
)

type StandardDeploy struct{}

func (sd StandardDeploy) ExecuteTasksOnMachines(machines []*Machine, plan *Plan) <-chan error {
	var wgMain, wgMachines sync.WaitGroup

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
					sd.executeTask(plan.registerMaps[ndx], m, t, plan, errChan)
				}(machine, task, ndx)
			}
			wgMachines.Wait()
		}
	}()

	go func() {
		wgMain.Wait()
		close(errChan)
	}()

	return errChan
}

// Uses plans ManageTaskRun(...)
func (sd StandardDeploy) executeTask(registerMap RegMap, machine *Machine, task *Task, plan *Plan, errs chan error) {
	var actualMachine *Machine
	if task.Local == true {
		actualMachine = localhost()
	} else {
		actualMachine = machine
	}

	vars := make(VarsMap)
	if err := task.SetupVars(plan, actualMachine, vars, registerMap); err != nil {
		errs <- err
		return
	}

	renderedTask, err := task.Render(vars, registerMap)
	if err != nil {
		errs <- HenchErr(err, map[string]interface{}{
			"plan": plan.Name,
			"task": renderedTask.Name,
			"host": actualMachine.Hostname,
		}, fmt.Sprintf("Error rendering task '%s'", renderedTask.Name))
		return
	}

	// accepted states are ok, success, ignored
	acceptedState, err := plan.ManageTaskRun(renderedTask, actualMachine, vars, registerMap)
	if !acceptedState {
		if err != nil {
			errs <- err
		}
		return
	}
}
