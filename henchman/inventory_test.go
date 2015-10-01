package henchman

import (
	"testing"
)

func loadValidInventory() (Inventory, error) {
	ic := make(InventoryConfig)
	ic["path"] = "test/inventory/validInventory.yaml"
	yi := YAMLInventory{}
	tt := TestTransport{}
	tt.Initialize(nil)
	inventory, err := yi.Load(ic, &tt)
	return inventory, err
}

func TestValidYAMLInventory(t *testing.T) {
	inventory, err := loadValidInventory()
	if err != nil {
		t.Fatalf("Unexpected error - %s\n", err.Error())
	}
	if inventory == nil {
		t.Fatalf("Inventory shouldn't be nil")
	}
	if len(inventory["nginx"]) != 2 {
		t.Errorf("Expected 2 nginx machines. Got %d instead\n", len(inventory["nginx"]))
	}
	if inventory.Count() != 3 {
		t.Errorf("Unexpected inventory count. Got %d\n", inventory.Count())
	}
}
