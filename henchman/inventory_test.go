package henchman

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadValidInventory() (Inventory, error) {
	ic := make(InventoryConfig)
	ic["path"] = "test/inventory/validInventory.yaml"
	yi := YAMLInventory{}
	inventory, err := yi.Load(ic)
	return inventory, err
}

func TestLoadInvalidInventory(t *testing.T) {
	ic := make(InventoryConfig)
	ic["path"] = "test/inventory/missingGroupsInventory.yaml"
	yi := YAMLInventory{}
	_, err := yi.Load(ic)
	require.Error(t, err)

	ic["path"] = "test/inventory/missingHostsInventory.yaml"
	_, err = yi.Load(ic)
	require.Error(t, err)

	ic["path"] = "test/inventory/invalidInventory.yaml"
	_, err = yi.Load(ic)
	require.Error(t, err)
}

func TestGetInventoryGroups(t *testing.T) {
	buf, err := ioutil.ReadFile("test/plan/inventoryAtHostLevel.yaml")
	require.NoError(t, err)

	groups, err := GetInventoryGroups(buf)
	require.NoError(t, err)
	assert.Equal(t, 1, len(groups))
	assert.Equal(t, []string{"nginx"}, groups)
}

func TestGetInventoryForGroups(t *testing.T) {
	inventory, err := loadValidInventory()
	require.NoError(t, err)
	require.NotNil(t, inventory)

	buf, err := ioutil.ReadFile("test/plan/inventoryAtHostLevel.yaml")
	require.NoError(t, err)

	groups, err := GetInventoryGroups(buf)
	require.NoError(t, err)
	assert.Equal(t, 1, len(groups))
	assert.Equal(t, []string{"nginx"}, groups)

	newInv := inventory.GetInventoryForGroups(groups)
	assert.Equal(t, 3, len(newInv.GlobalVars))
	assert.Equal(t, 2, len(newInv.HostVars))
}

func TestValidYAMLInventorygroup(t *testing.T) {
	inventory, err := loadValidInventory()

	require.NoError(t, err)
	require.NotNil(t, inventory)
	assert.Equal(t, 3, len(inventory.Groups), "Expected 3 inventory groups")
	assert.NotEmpty(t, inventory.Groups["nginx"].Vars, "Expected nginx vars to be non empty")
	assert.Empty(t, inventory.Groups["app"].Vars, "Expected app group vars to be empty")
	assert.Equal(t, []string{"192.168.1.1", "192.168.1.2"}, inventory.Groups["nginx"].Hosts)
	assert.Equal(t, 2, len(inventory.Groups["nginx"].Hosts), "Expected 2 nginx hosts")
	assert.Equal(t, 3, len(inventory.Groups["db"].Hosts), "Expected 3 db hosts")
	assert.NotEmpty(t, inventory.Groups["nginx"].Vars["ulimit"], "Ulimit was defined for nginx group")

	nginxUlimit := inventory.Groups["nginx"].Vars["ulimit"].(int)
	assert.Equal(t, 300, nginxUlimit, "NginxLimit was supposed to be 200")
	assert.Equal(t, "~/.ssh/ssh_key", inventory.Groups["nginx"].Vars["henchman_keyfile"], "henchman keyfile was expected")
}

func TestValidYAMLInventoryHostgroup(t *testing.T) {
	inventory, err := loadValidInventory()
	require.NoError(t, err)
	require.NotNil(t, inventory)

	assert.Equal(t, 2, len(inventory.HostVars), "Expected 2 host overrides")
	files := inventory.HostVars["1.1.1.1"]["files"].(int)
	assert.Equal(t, 240, inventory.HostVars["1.1.1.1"]["ulimit"].(int), "Expected 2 host overrides")
	assert.Equal(t, "~/.ssh/another_key",
		inventory.HostVars["192.168.1.1"]["keyfile"], "keyfile expected to be set to ~/.ssh/another_key")
	assert.Equal(t, files,
		inventory.HostVars["1.1.1.1"]["files"], "keyfile expected to be set to ~/.ssh/another_key")
}

// FIXME: Can't finish this test because of the transport config aspect
func TestGetMachines(t *testing.T) {
	inventory, err := loadValidInventory()
	require.NoError(t, err)
	require.NotNil(t, inventory)

	assert.Equal(t, 2, len(inventory.HostVars), "Expected 2 host overrides")
	files := inventory.HostVars["1.1.1.1"]["files"].(int)
	assert.Equal(t, 240, inventory.HostVars["1.1.1.1"]["ulimit"].(int), "Expected 2 host overrides")
	assert.Equal(t, "~/.ssh/another_key",
		inventory.HostVars["192.168.1.1"]["keyfile"], "keyfile expected to be set to ~/.ssh/another_key")
	assert.Equal(t, files,
		inventory.HostVars["1.1.1.1"]["files"], "keyfile expected to be set to ~/.ssh/another_key")
}

//NOTE: MergeHostVars is not being used in the code.  Remove at 1/4/16
/*
func TestMergeHostVars(t *testing.T) {
	inventory, err := loadValidInventory()
	require.NoError(t, err)
	require.NotNil(t, inventory)

	taskvars1 := map[interface{}]interface{}{"ulimit": 400}

	inventory.MergeHostVars("1.1.1.1", taskvars1)

	assert.Equal(t, 240, taskvars1["ulimit"], "Expected hostvars override for ulimit to be 240")

	taskvars2 := map[interface{}]interface{}{"ulimit": 400}
	inventory.MergeHostVars("1.1.1.2", taskvars2)
	assert.Equal(t, 400, taskvars2["ulimit"], "Expected hostvars override for ulimit to be 240")
}
*/
