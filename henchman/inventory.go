package henchman

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"

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

func (inv *Inventory) GetInventoryForGroups(groups []string) Inventory {
	// FIXME: Support globbing in the groups
	// No groups? No problem. Just return the full inventory
	//	return fullInventory
	if len(groups) == 0 {
		return *inv
	} else {
		filtered := Inventory{}
		filtered.Groups = make(map[string]HostGroup)
		//filtered.HostVars = fullInventory.HostVars
		//log.Println(fullInventory)
		for _, group := range groups {
			machines, present := inv.Groups[group]
			if present {
				filtered.Groups[group] = machines
			}
		}
		filtered.HostVars = inv.HostVars
		return filtered
	}
}

func (inv *Inventory) GetMachines(tc TransportConfig) ([]*Machine, error) {
	var machines []*Machine
	machineSet := make(map[string]bool)
	for _, hostGroup := range inv.Groups {
		for _, hostname := range hostGroup.Hosts {
			if _, present := machineSet[hostname]; !present {
				machine := &Machine{}
				machineSet[hostname] = true
				machine.Hostname = hostname
				machine.Vars = hostGroup.Vars
				machines = append(machines, machine)
			} else {
				//machine part of multiple groups
				//update vars if any
				//latter group's vars overrides prev. groups vars
				for _, machine := range machines {
					if machine.Hostname == hostname {
						MergeMap(hostGroup.Vars, machine.Vars, true)
					}
					//get second groups henchman tc vars
				}
			}
		}
	}

	//update hostvars
	for _, machine := range machines {
		for hostname, vars := range inv.HostVars {
			if machine.Hostname == hostname {
				MergeMap(vars, machine.Vars, true)
			}
		}
		// now open ssh connection for each machine
		tcCurr := make(TransportConfig)
		tcCurr["hostname"] = machine.Hostname
		for k, v := range tc {
			tcCurr[k] = v
		}
		henchmanVars := GetHenchmanVars(machine.Vars)
		for k, v := range henchmanVars {
			tcCurr[k.(string)] = v.(string)
		}
		log.Println("Transport Config for machine", machine.Hostname, ": ", tcCurr)
		ssht, err := NewSSH(&tcCurr)
		if err != nil {
			return nil, err
		}
		machine.Transport = ssht
	}

	return machines, nil
}

//func mergeHenchmanVarsWithTransport(tcCurr TransportConfig, hostname string, hostGroupVars VarsMap,
//	hostVars map[string]map[interface{}]interface{}) {
//
//	henchmanVars := make(VarsMap)
//	henchmanVars = GetHenchmanVars(hostGroupVars)
//	for k, v := range henchmanVars {
//		tcCurr[k.(string)] = v.(string)
//	}
//	if _, present := hostVars[hostname]; present {
//		henchmanVars = GetHenchmanVars(hostVars[hostname])
//		for k, v := range henchmanVars {
//			tcCurr[k.(string)] = v.(string)
//		}
//	}
//}

func GetHenchmanVars(vars VarsMap) VarsMap {
	var henchmanVars VarsMap

	for k, v := range vars {
		if strings.Contains(k.(string), HENCHMAN_PREFIX) {
			if len(henchmanVars) == 0 {
				henchmanVars = make(VarsMap)
			}
			parts := strings.Split(k.(string), HENCHMAN_PREFIX)
			henchmanVars[parts[len(parts)-1]] = v
		}
	}

	return henchmanVars
}
