package henchman

import (
	"fmt"
	"io/ioutil"
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
	Groups     map[string]HostGroup `yaml:"groups"`
	HostVars   map[string]VarsMap   `yaml:"host_vars"`
	GlobalVars VarsMap              `yaml:"global_vars"`
}

type HostGroup struct {
	Hosts []string `yaml:"hosts"`
	Vars  VarsMap  `yaml:"vars"`
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
	Groups     map[string]HostGroup `yaml:"groups"`
	HostVars   map[string]VarsMap   `yaml:"host_vars"`
	GlobalVars VarsMap              `yaml:"global_vars"`
}

func (yi *YAMLInventory) Load(ic InventoryConfig) (Inventory, error) {
	fname, present := ic["path"]
	if !present {
		return Inventory{}, fmt.Errorf("Missing 'path' in the config")
	}
	buf, err := ioutil.ReadFile(fname)
	if err != nil {
		return Inventory{}, HenchErr(err, map[string]interface{}{
			"file":     fname,
			"solution": "make sure directory exists, correct permissions, or is not corrupted",
		}, "While Reading File")
	}
	err = yaml.Unmarshal(buf, &yi)
	if err != nil {
		return Inventory{}, HenchErr(err, map[string]interface{}{
			"file":     fname,
			"solution": "Make sure inventory follows proper formatting.  Also check for tabs when there should be spaces.",
		}, "While unmarshalling inventory")
	}

	if yi.Groups == nil {
		return Inventory{}, HenchErr(fmt.Errorf("Groups field is required."), map[string]interface{}{
			"file":     fname,
			"solution": "Refer to the wiki for proper formatting.",
		}, "")
	}

	for key, val := range yi.Groups {
		if key == "hosts" {
			return Inventory{}, HenchErr(fmt.Errorf("'hosts' is not a valid group name"), map[string]interface{}{
				"file":     fname,
				"solution": "Change a group name away from hosts",
			}, "")
		}
		if val.Hosts == nil {
			return Inventory{}, HenchErr(fmt.Errorf("%v requires a hosts field.", key), map[string]interface{}{
				"file":     fname,
				"group":    key,
				"solution": "Refet to the wiki for proper formatting.",
			}, "")
		}
	}

	iv := &Inventory{}
	iv.HostVars = yi.HostVars
	iv.Groups = yi.Groups
	iv.GlobalVars = yi.GlobalVars
	if iv.GlobalVars == nil {
		iv.GlobalVars = make(VarsMap)
	}
	return *iv, nil
}

/**
 * gets the groups under hosts section from plan file
 */
func GetInventoryGroups(planBuf []byte) ([]string, error) {
	hostsProxy := struct {
		Groups []string `yaml:"hosts"`
	}{}

	err := yaml.Unmarshal(planBuf, &hostsProxy)
	if err != nil {
		return nil, HenchErr(err, map[string]interface{}{
			"solution": "Check if hosts section exists",
		}, "While unmarshalling hosts section")
	}

	return hostsProxy.Groups, nil
}

/**
 * Returns a filtered inventory depending on which groups are specified
 * Keeps the groups specified in the filtered inventory
 */
func (inv *Inventory) GetInventoryForGroups(groups []string) Inventory {
	// FIXME: Support globbing in the groups
	// No groups? No problem. Just return the full inventory
	//	return fullInventory
	if len(groups) == 0 {
		return *inv
	}
	filtered := Inventory{}
	filtered.Groups = make(map[string]HostGroup)
	//filtered.HostVars = fullInventory.HostVars
	if len(groups) == 1 &&
		(groups[0] == "localhost" || groups[0] == "127.0.0.1") {
		filtered.Groups["localhost"] = HostGroup{Hosts: []string{"localhost"},
			Vars: nil}
	} else {
		for _, group := range groups {
			machines, present := inv.Groups[group]
			if present {
				filtered.Groups[group] = machines
			}
		}
	}
	filtered.HostVars = inv.HostVars
	filtered.GlobalVars = inv.GlobalVars
	return filtered
}

/**
 * Sets all_hosts, and groups with attached hosts lists
 * These can be accessed in vars.inv.whatevs
 */
func (inv *Inventory) SetGlobalVarsFromInventoryGroups(groups map[string]HostGroup) {
	var all_hosts []string
	globalVars := make(map[string]interface{})
	duplicates := make(map[string]bool)
	for group, hostGroup := range groups {
		globalVars[group] = hostGroup.Hosts
		for _, host := range hostGroup.Hosts {
			if _, present := duplicates[host]; !present {
				duplicates[host] = true
				all_hosts = append(all_hosts, host)
			}
		}
	}
	globalVars["all_hosts"] = all_hosts
	inv.GlobalVars["inv"] = globalVars
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
				if machine.Vars == nil {
					machine.Vars = make(VarsMap)
				}
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

	// gets henchman specific vars from global_vars
	globalInvHenchmanVars := GetHenchmanVars(inv.GlobalVars)

	// update hostvars
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

		// gets henchman specific vars from each machine and merges with global
		henchmanVars := GetHenchmanVars(machine.Vars)
		MergeMap(globalInvHenchmanVars, henchmanVars, false)

		for k, v := range henchmanVars {
			tcCurr[k] = v.(string)
		}
		Debug(map[string]interface{}{
			"host":   machine.Hostname,
			"config": tcCurr,
		}, "Transport Config for machine")

		// FIXME: This is frigging wrong
		// See #47
		if machine.Hostname != "localhost" {
			ssht, err := NewSSH(&tcCurr)
			if err != nil {
				return nil, err
			}
			machine.Transport = ssht
		} else {
			machine.Transport = nil
		}
	}

	return machines, nil
}

/**
 * Gets any vars with the prefix henchman
 */
func GetHenchmanVars(vars VarsMap) VarsMap {
	henchmanVars := VarsMap{}

	for k, v := range vars {
		if strings.Contains(k, HENCHMAN_PREFIX) {
			parts := strings.Split(k, HENCHMAN_PREFIX)
			henchmanVars[parts[len(parts)-1]] = v
		}
	}

	return henchmanVars
}
