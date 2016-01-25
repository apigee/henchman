package henchman

var Deploy DeployInterface

type DeployInterface interface {
	ExecuteTasksOnMachines(machines []*Machine, plan *Plan) <-chan error
}

func SetDeployType(deployType string) {
	switch deployType {
	case "rolling":
		Deploy = RollingDeploy{}
	default:
		Deploy = StandardDeploy{}
	}
}
