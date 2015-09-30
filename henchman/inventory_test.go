package henchman

import (
	"testing"
)

func TestValidYAMLInventory(t *testing.T) {
	ic := make(InventoryConfig)
	ic["path"] = "test/inventory/validInventory.yaml"
	yi := YAMLInventory{}
	inventory, err := yi.Load(ic)
	if err != nil {
		t.Fatalf("Unexpected error - %s\n", err.Error())
	}
	if inventory == nil {
		t.Fatalf("Inventory shouldn't be nil")
	}
	if len(inventory["nginx"]) != 2 {
		t.Errorf("Expected 2 nginx machines. Got %d instead\n", len(inventory["nginx"]))
	}
}
