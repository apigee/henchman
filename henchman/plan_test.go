// FIXME: Can't test plan reliably till we fix #19,

package henchman

// import (
// 	"testing"

// 	_ "github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// )

// func TestPlanExecute(t *testing.T) {
// 	inventory, err := loadValidInventory()
// 	require.NoError(t, err)
// 	// Just one task
// 	task := Task{}
// 	m, err := NewModule("test", "foo=bar")
// 	require.NoError(t, err)

// 	task.Id = "f"
// 	task.Sudo = false
// 	task.Name = "testTask"
// 	task.Module = m
// 	task.Local = false

// 	p := Plan{}
// 	p.Name = "Test Plan"
// 	p.Inventory = inventory
// 	p.Vars = make(VarsMap)
// 	p.Tasks = append(p.Tasks, &task)

// 	tc := make(TransportConfig)
// 	tc["username"] = "foobar"
// 	machines, err := inventory.GetMachines(tc)
// 	require.NoError(t, err)

// 	err = p.Execute(machines)
// 	require.NoError(t, err)
// }
