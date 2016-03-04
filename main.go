package main

import (
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/cloudnautique/rbs-sandbox/rancher"
	"github.com/codegangsta/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "Rancher Bootstrap"
	app.Usage = "File driven Rancher configuration"
	app.Action = appInit
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "c,config-file",
			Usage: "Path to config file",
			Value: "./config.yml",
		},
		cli.StringFlag{
			Name:  "k,key-file",
			Usage: "Path where Admin Keys will be stored",
			Value: "./.keys",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:    "registration-command",
			Aliases: []string{"rc"},
			Usage:   "Get the registration command for nodes",
			Action:  appEnvironmentRegistrationTokens,
		},
	}

	app.Run(os.Args)
}

func appInit(c *cli.Context) {
	RancherServer := rancher.NewRancherServer(c.String("config-file"), c.String("key-file"))
	err := RancherServer.ConfigureAuthBackend()
	if err != nil {
		logrus.Fatalf("Failed to Configure auth: %s", err)
	}

	err = RancherServer.ConfigureAccounts()
	if err != nil {
		logrus.Fatalf("Failed to Configure Accounts: %s", err)
	}

	err = RancherServer.ConfigureEnvironments()
	if err != nil {
		logrus.Fatalf("Failed to Configure Accounts: %s", err)
	}
	err = RancherServer.ConfigureEnvironmentAccess()
	if err != nil {
		logrus.Fatalf("Failed to Configure Accounts: %s", err)
	}

	err = RancherServer.ConfigureRegistries()
	if err != nil {
		logrus.Fatalf("Failed to Configure Registries: %s", err)
	}
}

func appEnvironmentRegistrationTokens(c *cli.Context) {
	RancherServer := rancher.NewRancherServer(c.GlobalString("config-file"), c.GlobalString("key-file"))

	if len(c.Args()) == 0 {
		logrus.Fatalf("Need at least one environment name")
	}

	for _, project := range c.Args() {
		cmd, err := RancherServer.GetEnvironmentRegistrationCommand(project)
		if err != nil {
			logrus.Fatalf("Could not get Registration command for: %s\n%s", project, err)
		}
		fmt.Println(cmd)
	}
}
