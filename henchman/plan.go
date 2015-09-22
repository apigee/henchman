package henchman

type Plan struct {
	Name  string
	Hosts []string
	Vars  TaskVars
	Tasks []*Task
}

type Inventory map[string][]string
