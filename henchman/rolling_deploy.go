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
	var wg sync.WaitGroup
	errChan := make(chan error, 1)

	// NOTE: work in progress laid out to get ideas
	// case 1 no num hosts variable
	if rd.NumHosts == 0 || rd.NumHosts == 1 {
		go func() {
			defer close(errChan)
			for _, machine := range machines {
				rd.executeTasks(machine, plan, errChan)
			}
		}()
		// case 2 "whole number" specified
	} else if rd.NumHosts > 1.0 {
		go func() {
			defer close(errChan)
			numHosts := int(rd.NumHosts)
			for i := 0; i < len(machines); i += numHosts {
				if i+numHosts < len(machines) {
					for _, machine := range machines[i:(i + numHosts)] {
						wg.Add(1)
						go func(m *Machine) {
							defer wg.Done()
							rd.executeTasks(machine, plan, errChan)
						}(machine)
					}
				} else {
					for _, machine := range machines[i:] {
						wg.Add(1)
						go func(m *Machine) {
							defer wg.Done()
							rd.executeTasks(machine, plan, errChan)
						}(machine)
					}
				}
				wg.Wait()
			}
		}()
	}

	return errChan
}

// executeTasks sets up the final variable map, then renders and runs the task list
func (rd RollingDeploy) executeTasks(machine *Machine, plan *Plan, errs chan error) {
	registerMap := make(RegMap)
	var actualMachine *Machine
	for _, task := range plan.Tasks {
		if task.Local == true {
			actualMachine = localhost()
		} else {
			actualMachine = machine
		}

		// copy of task.Vars. It'll be different for each machine
		vars := make(VarsMap)
		if err := task.SetupVars(plan, actualMachine, vars, registerMap); err != nil {
			errs <- err
			return
		}

		// Checks for subtasks in the with_items field
		subTasks, err := task.ProcessWithItems(vars, registerMap)
		if err != nil {
			errs <- HenchErr(err, map[string]interface{}{
				"plan": plan.Name,
				"task": task.Name,
				"host": actualMachine.Hostname,
			}, fmt.Sprintf("Error generating with_items tasks '%s'", task.Name))
			return
		}

		if subTasks != nil {
			for _, subTask := range subTasks {
				acceptedState, err := plan.ManageTaskRun(subTask, actualMachine, vars, registerMap)
				if !acceptedState {
					if err != nil {
						errs <- err
						return
					}
					if !promptContinue() {
						return
					}
				}
			}
		} else {
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
					return
				}
				if !promptContinue() {
					return
				}
			}
		}
	}
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
