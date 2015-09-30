package main

import (
	"log"
	"os"
	"os/user"
	"path"

	_ "github.com/apigee/henchman/henchman"
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
	// Step 1: Validate Modules path and see if it exists
	// Step 2: Read the planFile
	// Step 3: Set up transports for the inventory
	// Step 4: For every machine, run the tasks
	log.Printf("Executing the plan\n")
}

func main() {
	app := cli.NewApp()
	app.Name = "henchman"
	app.Usage = "Orchestration framework"
	app.Version = "0.1"
	app.Commands = gatherCommands()
	app.Run(os.Args)
}
