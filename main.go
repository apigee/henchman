package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/apigee/henchman/henchman"
	"github.com/codegangsta/cli"
)

// NOTE: We're not using os/user because of the requirement on cgo.
// This prevents us from creating cross-builds. Therefore just env vars.
// Check https://github.com/golang/go/issues/6376
func currentUsername() string {
	// FIXME: Do we even care for Windows?
	currUser := os.Getenv("USER")
	if currUser == "" {
		henchman.Warn(nil, "Couldn't get current username. Assuming root")
		return "root"
	}
	return currUser
}

func defaultKeyFile() string {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		currUser := currentUsername()
		homeDir = path.Join("/home", currUser)
	}
	return path.Join(homeDir, ".ssh", "id_rsa")
}

func gatherCommands() []cli.Command {
	// For now, we don't have any flags that are common
	// across all the subcommands.
	globalFlags := []cli.Flag{}
	inventoryFlag := cli.StringFlag{
		Name:  "inventory",
		Value: "hosts",
		Usage: "Path to the inventory",
	}
	moduleFlag := cli.StringFlag{
		Name:   "modules",
		Value:  "modules",
		Usage:  "Root directory to the modules",
		EnvVar: "HENCHMAN_MODULES",
	}
	usernameFlag := cli.StringFlag{
		Name:  "user",
		Value: currentUsername(),
		Usage: "Remote user executing this plan",
	}
	debugFlag := cli.BoolFlag{
		Name:  "debug",
		Usage: "Allows indepth output.  E.G the register map after every task",
	}
	cleanupFlag := cli.BoolFlag{
		Name:  "cleanup",
		Usage: "Will removed .henchman directory from all remote machines",
	}
	configurationFlag := cli.StringFlag{
		Name:  "configuration",
		Value: "conf.json",
		Usage: "Path to the configuration file",
	}
	// FIXME: Should this come from the transport instead.
	// Different transports can come up with their own set of flags
	// For now our world is just ssh
	keyFileFlag := cli.StringFlag{
		Name:  "keyfile",
		Value: defaultKeyFile(),
		Usage: "Path to the ssh private key. Make sure that this key is in the authorized_keys in the remote nodes",
	}

	// TODO: Password based authentication
	// Password auth and keyfile auth are mutually exclusive
	return []cli.Command{
		{
			Name:   "exec",
			Usage:  "Execute a plan on the given group in the inventory",
			Action: executePlan,
			Flags:  append(globalFlags, moduleFlag, inventoryFlag, usernameFlag, keyFileFlag, debugFlag, cleanupFlag, configurationFlag),
		},
	}
}

func executePlan(c *cli.Context) {
	args := c.Args()
	if len(args) == 0 {
		// FIXME: Just print out the usage info?
		henchman.Fatal(nil, "Missing path to the plan")
	}

	// Step 0: Set global variables and Init stuff
	henchman.DebugFlag = c.Bool("debug")

	// NOTE: can't use HenchErr b/c it hasn't been initialized yet
	if err := henchman.InitConfiguration(c.String("configuration")); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if err := henchman.InitLog(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Step 1: Validate Modules path and see if it exists
	modulesPath := c.String("modules")
	user := c.String("user")
	keyfile := c.String("keyfile")
	_, err := os.Stat(modulesPath)
	if err != nil {
		henchman.Fatal(map[string]interface{}{
			"mod path": modulesPath,
			"error":    err.Error(),
		}, "Error Validating Modules Dir")
	}
	henchman.ModuleSearchPath = append(henchman.ModuleSearchPath, modulesPath)

	// Step 2: Read the inventory
	// FIXME: Support multiple inventory types.
	// We're dealing with YAML for now
	inventoryPath := c.String("inventory")
	inventoryConfig := make(henchman.InventoryConfig)
	inventoryConfig["path"] = inventoryPath
	inventorySource := new(henchman.YAMLInventory)
	tc := make(henchman.TransportConfig)
	tc["username"] = user
	tc["keyfile"] = keyfile

	inv, err := inventorySource.Load(inventoryConfig)
	if err != nil {
		henchErr := henchman.HenchErr(err, map[string]interface{}{
			"error": err.Error(),
		}, "").(*henchman.HenchmanError)
		henchman.Fatal(henchErr.Fields, "Error Loading Inventory")
	}

	// Step 3: Preprocess Plans and execute
	planFile := args[0]
	planBuf, err := ioutil.ReadFile(planFile)
	if err != nil {
		henchErr := henchman.HenchErr(err, map[string]interface{}{
			"plan":  planFile,
			"error": err.Error(),
		}, "").(*henchman.HenchmanError)
		henchman.Fatal(henchErr.Fields, "Error Reading Plan")
	}

	// Step 3.1: Find the groups being used by plan
	groups, err := henchman.GetInventoryGroups(planBuf)
	if err != nil {
		henchErr := henchman.HenchErr(err, map[string]interface{}{
			"plan":  planFile,
			"error": err.Error(),
		}, "").(*henchman.HenchmanError)
		henchman.Fatal(henchErr.Fields, "Error Getting Inv Groups")
	}

	// Step 3.2: Create a filtered Inv with only groups the plan specified
	inventory := inv.GetInventoryForGroups(groups)

	// Step 3.3: For each machine assign group and host vars
	machines, err := inventory.GetMachines(tc)
	if err != nil {
		henchErr := henchman.HenchErr(err, map[string]interface{}{
			"plan":  planFile,
			"error": err.Error(),
		}, "").(*henchman.HenchmanError)
		henchman.Fatal(henchErr.Fields, "Error Getting Machines")
	}
	// Step 3.4: Add entire set of inventory variables in inv
	// to inventory.groups GlobalVars
	inventory.SetGlobalVarsFromInventoryGroups(inv.Groups)

	// Step 3.5: Preprocess plan to create plan struct
	//           Setup final version of vars
	plan, err := henchman.PreprocessPlan(planBuf, &inventory)
	if err != nil {
		henchErr := henchman.HenchErr(err, map[string]interface{}{
			"plan":  planFile,
			"error": err.Error(),
		}, "").(*henchman.HenchmanError)
		henchman.Fatal(henchErr.Fields, "Error Preprocessing Plan")
	}

	// Step 3.5: Setup plan and execute
	if err := plan.Setup(machines); err != nil {
		henchErr := henchman.HenchErr(err, map[string]interface{}{
			"error": err.Error(),
		}, "").(*henchman.HenchmanError)
		henchman.Fatal(henchErr.Fields, "Error in plan setup")
	}

	if err := plan.Execute(machines); err != nil {
		henchErr := henchman.HenchErr(err, map[string]interface{}{
			"error": err.Error(),
		}, "").(*henchman.HenchmanError)
		henchman.Fatal(henchErr.Fields, "Error in executing plan")
	}

	if c.Bool("cleanup") {
		if err := plan.Cleanup(machines); err != nil {
			henchErr := henchman.HenchErr(err, map[string]interface{}{
				"error": err.Error(),
			}, "").(*henchman.HenchmanError)
			henchman.Fatal(henchErr.Fields, "Error in plan cleanup")
		}
	}
}

var minversion string

func main() {
	app := cli.NewApp()
	app.Name = "henchman"
	app.Usage = "Orchestration framework"
	app.Version = "0.1." + minversion
	app.Commands = gatherCommands()
	app.Run(os.Args)
}
