package henchman

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type InventoryConfig map[string]string
type InventoryInterface interface {
	Load(ic InventoryConfig) (Inventory, error)
}

//type Inventory map[string][]*Machine
type Inventory struct {
	//GroupHosts map[string][]*Machine
	Groups   map[string]HostGroup                   `yaml:"groups"`
	HostVars map[string]map[interface{}]interface{} `yaml:"hostvars"`
}

type HostGroup struct {
	Hosts []string                    `yaml:"hosts"`
	Vars  map[interface{}]interface{} `yaml:"vars"`
}

func (inv Inventory) Count() int {
	count := 0
	for _, hostGroup := range inv.Groups {
		count += len(hostGroup.Hosts)
	}
	return count
}

// FIXME: Have a way to provide specifics
type YAMLInventory struct {
	Groups   map[string]HostGroup                   `yaml:"groups"`
	HostVars map[string]map[interface{}]interface{} `yaml:"hostvars"`
}

func (yi *YAMLInventory) Load(ic InventoryConfig) (Inventory, error) {
	fname, present := ic["path"]
	if !present {
		return Inventory{}, fmt.Errorf("Missing 'path' in the config")
	}
	buf, err := ioutil.ReadFile(fname)
	if err != nil {
		return Inventory{}, err
	}
	err = yaml.Unmarshal(buf, &yi)
	if err != nil {
		return Inventory{}, err
	}
	iv := &Inventory{}
	iv.HostVars = yi.HostVars
	iv.Groups = yi.Groups
	return *iv, nil
}

func (inv *Inventory) MergeHostVars(hostname string, taskVars map[interface{}]interface{}) {
	if len(inv.HostVars) == 0 {
		return
	}
	if _, present := inv.HostVars[hostname]; present {
		MergeMap(inv.HostVars[hostname], taskVars, true)
	}
}
