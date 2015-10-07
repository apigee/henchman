package henchman

import (
	//"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskRun(t *testing.T) {
	mod := moduleTestSetup("sample_module")
	defer moduleTestTeardown(mod)

	origSearchPath := ModuleSearchPath
	ModuleSearchPath = append(ModuleSearchPath, "/tmp")
	defer func() {
		ModuleSearchPath = origSearchPath
	}()

	task := Task{}
	task.Name = "test"
	task.Module = mod

	testTransport := TestTransport{}

	localhost := Machine{}
	localhost.Hostname = "localhost"
	localhost.Transport = &testTransport

	_, err := task.Run(&localhost)
	require.NoError(t, err, "There shouldn't have been any errors")
}

func TestTaskRender(t *testing.T) {
	buf, err := ioutil.ReadFile("test/plan/planWithPongo2.yaml")
	require.NoError(t, err, "Could not read planWithPongo2.yaml")

	plan, err := PreprocessPlan(buf, nil)
	require.NoError(t, err, "This plan shouldn't be having an error")

	testTransport := TestTransport{}
	localhost := Machine{}
	localhost.Hostname = "localhost"
	localhost.Transport = &testTransport

	task := plan.Tasks[0]
	name, err := task.Render(task.Name)
	require.NoError(t, err, "This plan shouldn't be having an error")
	assert.Equal(t, "iptables with abcd1234", name, "Expected iptables at abcd1234. Received %v\n")

	params, err := task.Render(task.Module.Params)
	require.NoError(t, err, "This plan shouldn't be having an error")
	assert.Equal(t, "iptables", params.(map[string]string)["key"], "Expected iptables at abcd1234. Received %v\n")
}
