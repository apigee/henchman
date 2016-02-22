package henchman

type DeployInterface interface {
	ExecuteTasksOnMachines(machines []*Machine, plan *Plan) <-chan error
}
