package henchman

type Plan struct {
	Name      string
	Inventory Inventory
	Vars      TaskVars
	Tasks     []*Task
}

func (plan *Plan) Execute() error {
	return nil
}
