package henchman

import (
	"errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

// type Inventory map[string][]*Machine
type Inventory map[string][]string

type InventoryConfig map[string]string
type InventoryInterface interface {
	Load(ic InventoryConfig) (Inventory, error)
}

// FIXME: Have a way to provide specifics
type YAMLInventory struct{}

func (ti *YAMLInventory) Load(ic InventoryConfig) (Inventory, error) {
	fname, present := ic["path"]
	if !present {
		return nil, errors.New("Missing 'path' in the config")
	}
	buf, err := ioutil.ReadFile(fname)
	if err != nil {
		return nil, err
	}
	iv := make(Inventory)
	err = yaml.Unmarshal(buf, &iv)
	if err != nil {
		return nil, err
	}
	return iv, nil
}
