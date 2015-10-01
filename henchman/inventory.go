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
	seen := make(map[string]bool) // Set of machines that've been 'seen'
	for _, machines := range inv {
		for _, machine := range machines {
			seen[machine.Hostname] = true
		}
	}
	return len(seen)
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
