package henchman

type DeployInterface interface {
	ExecuteTasksOnMachines(machines []*Machine, plan *Plan) <-chan error
}

type Deploy struct {
	DeployInterface DeployInterface `yaml:-`
	Method          string
	NumHosts        float64 `yaml:"num_hosts"`
}

// InitDeployMethod links a deployment method to the DeployInterface.
func (d *Deploy) InitDeployMethod() {
	switch d.Method {
	case "rolling":
		d.DeployInterface = RollingDeploy{numHosts: d.NumHosts}
	default:
		d.DeployInterface = StandardDeploy{}
	}
}
