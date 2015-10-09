package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"

	"github.com/apigee/henchman/henchman"
	"github.com/codegangsta/cli"
)

func currentUsername() *user.User {
	u, err := user.Current()
	if err != nil {
		log.Printf("Couldn't get current username: %s. Assuming root" + err.Error())
		u, err = user.Lookup("root")
		if err != nil {
			log.Print(err.Error())
		}
		return u
	}
	return u
}

func defaultKeyFile() string {
	u := currentUsername()
	return path.Join(u.HomeDir, ".ssh", "id_rsa")
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
		Value: currentUsername().Username,
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

	inv, err := inventorySource.Load(inventoryConfig, tc)
	if err != nil {
		log.Fatalf("Error loading inventory - %s\n", err.Error())
	}

	// Step 3: Read the planFile
	planFile := args[0]
	planBuf, err := ioutil.ReadFile(planFile)
	if err != nil {
		log.Fatalf("Error when reading plan `%s': %s", planFile, err.Error())
	}
	plan, err := henchman.PreprocessPlan(planBuf, inv)
	if err != nil {
		log.Fatalf(err.Error())
	}
	plan.Execute()
}

func main() {
	app := cli.NewApp()
	app.Name = "henchman"
	app.Usage = "Orchestration framework"
	app.Version = "0.1"
	app.Commands = gatherCommands()
	app.Run(os.Args)
}
