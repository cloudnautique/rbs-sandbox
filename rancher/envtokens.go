package rancher

import (
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/client"
)

func (r *RancherServer) GetEnvironmentRegistrationCommand(projectName string) (string, error) {
	var cmd string
	project, err := getProjectByName(projectName, r.client)
	if err != nil {
		return cmd, err
	}

	tokens, err := getRegistrationTokens(project, r.client)
	if err != nil {
		return cmd, err
	}

	if len(tokens.Data) == 0 {
		logrus.Infof("Creating command for: %s", project.Name)
		var token client.RegistrationToken
		err = r.client.Post(project.Links["registrationTokens"], &client.RegistrationToken{
			AccountId: project.Id,
		}, &token)
		if err != nil {
			return cmd, err
		}

		err = waitForCommand(r.client, &token.Resource, &token)

		cmd = token.Command
	} else {
		if tokens.Data[0].Command != "" && tokens.Data[0].State == "active" {
			cmd = tokens.Data[0].Command
		}
	}

	return cmd, nil
}

func waitForCommand(c *client.RancherClient, resource *client.Resource, token *client.RegistrationToken) error {
	for {
		if token.Command != "" {
			return nil
		}
		time.Sleep(150 * time.Millisecond)

		err := c.Reload(resource, &token)
		if err != nil {
			return err
		}
	}
}

func getRegistrationTokens(project *client.Project, rClient *client.RancherClient) (client.RegistrationTokenCollection, error) {
	var registrationTokens client.RegistrationTokenCollection
	err := rClient.GetLink(project.Resource, "registrationTokens", &registrationTokens)
	return registrationTokens, err
}
