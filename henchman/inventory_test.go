package henchman

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadValidInventory() (Inventory, error) {
	ic := make(InventoryConfig)
	ic["path"] = "test/inventory/validInventory.yaml"
	yi := YAMLInventory{}
	tc := make(TransportConfig)
	tc["hostname"] = "foo"
	tc["username"] = "foobar"
	tc["password"] = "bar"
	inventory, err := yi.Load(ic, tc)
	return inventory, err
}

func TestValidYAMLInventory(t *testing.T) {
	inventory, err := loadValidInventory()

	require.NoError(t, err)
	require.NotNil(t, inventory)
	assert.Equal(t, 2, len(inventory["nginx"]), "Expected 2 nginx machines")
	assert.Equal(t, 3, inventory.Count(), "Unexpected inventory count")
	assert.Equal(t, 3, len(inventory.Machines()), "Unexpected machine count")
}
