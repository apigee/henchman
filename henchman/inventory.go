package henchman

import (
	"errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type InventoryConfig map[string]string
type InventoryInterface interface {
	Load(ic InventoryConfig, transport TransportInterface) (Inventory, error)
}

type Inventory map[string][]*Machine

func (inv Inventory) Count() int {
	return len(inv.Machines())
}

func (inv Inventory) Machines() []*Machine {
	// The same machine might be in different groups.
	// We don't want duplicates when Machines() is being invoked
	machineSet := make(map[string]bool)
	var machines []*Machine
	for _, ms := range inv {
		for _, m := range ms {
			_, present := machineSet[m.Hostname]
			if !present {
				machines = append(machines, m)
				machineSet[m.Hostname] = true
			}
		}
	}
	return machines
}

// FIXME: Have a way to provide specifics
type YAMLInventory map[string][]string

func (yi *YAMLInventory) Load(ic InventoryConfig, transport TransportInterface) (Inventory, error) {
	fname, present := ic["path"]
	if !present {
		return nil, errors.New("Missing 'path' in the config")
	}
	buf, err := ioutil.ReadFile(fname)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(buf, &yi)
	if err != nil {
		return nil, err
	}
	iv := make(Inventory)
	for group, hostnames := range *yi {
		for _, hostname := range hostnames {
			machine := Machine{}
			machine.Hostname = hostname
			machine.Transport = transport
			iv[group] = append(iv[group], &machine)
		}
	}
	return iv, nil
}
