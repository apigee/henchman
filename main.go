package main

import (
	"io/ioutil"
	"log"
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
		log.Println("Couldn't get current username. Assuming root")
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
			Flags:  append(globalFlags, moduleFlag, inventoryFlag, usernameFlag, keyFileFlag),
		},
	}
}

func setInventoryVars(plan *henchman.Plan, inv henchman.Inventory) {
	var all_hosts []string
	duplicates := make(map[string]bool)
	for group, hostGroup := range inv.Groups {
		plan.Vars[group] = hostGroup.Hosts
		for _, host := range hostGroup.Hosts {
			if _, present := duplicates[host]; !present {
				duplicates[host] = true
				all_hosts = append(all_hosts, host)
			}
		}
	}

	plan.Vars["all_hosts"] = all_hosts
}

func executePlan(c *cli.Context) {
	args := c.Args()
	if len(args) == 0 {
		// FIXME: Just print out the usage info?
		log.Fatalf("Missing path to the plan")
	}
	// Step 1: Validate Modules path and see if it exists
	modulesPath := c.String("modules")
	user := c.String("user")
	keyfile := c.String("keyfile")
	_, err := os.Stat(modulesPath)
	if err != nil {
		log.Fatalf("Error when validating the modules directory - %s\n", err.Error())
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
		log.Fatalf("Error loading inventory - %s\n", err.Error())
	}

	// Step 3: Read the planFile
	planFile := args[0]
	planBuf, err := ioutil.ReadFile(planFile)
	if err != nil {
		log.Fatalf("Error when reading plan `%s': %s", planFile, err.Error())
	}
	invGroups, err := henchman.GetInventoryGroups(planBuf)
	if err != nil {
		log.Fatalf(err.Error())
	}
	inventory := inv.GetInventoryForGroups(invGroups)
	machines, err := inventory.GetMachines(tc)

	plan, err := henchman.PreprocessPlan(planBuf, inventory)
	if err != nil {
		log.Fatalf(err.Error())
	}

	setInventoryVars(plan, inv)
	plan.Execute(machines)
}

func main() {
	app := cli.NewApp()
	app.Name = "henchman"
	app.Usage = "Orchestration framework"
	app.Version = "0.1"
	app.Commands = gatherCommands()
	app.Run(os.Args)
}
