package rancher

import (
	"errors"
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/client"
)

func getExistingProjectMembers(projectName string, rClient *client.RancherClient) ([]client.ProjectMember, error) {
	logrus.Infof("Getting project members for: %s", projectName)

	// ToDo: Do not explode if project doesn't exist

	var existing []client.ProjectMember
	project, err := getProjectByName(projectName, rClient)
	if err != nil {
		return existing, err
	}

	existing, err = getProjectMembers(project, rClient)
	if err != nil {
		return existing, err
	}

	return existing, nil
}
func getProjectMemberIdentity(name string, role string, rClient *client.RancherClient) (client.ProjectMember, error) {
	logrus.Infof("Getting Identity for: %s", name)
	var newMember = &client.ProjectMember{}
	identityCollection, err := rClient.Identity.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"name": name,
		},
	})
	if err != nil {
		return *newMember, err
	}

	var identity client.Identity
	if len(identityCollection.Data) > 0 {
		identity = identityCollection.Data[0]
	} else {
		return *newMember, errors.New(fmt.Sprintf("Could not get Identity: %s\n Got: %#v", name, identityCollection))
	}

	newMember.ExternalId = identity.ExternalId
	newMember.ExternalIdType = identity.ExternalIdType
	newMember.Role = role

	return *newMember, nil
}

func addEnvironmentMembers(projectName string, members []client.ProjectMember, rClient *client.RancherClient) error {
	setProjectMembersInput := &client.SetProjectMembersInput{
		Members: members,
	}

	project, err := getProjectByName(projectName, rClient)
	if err != nil {
		return err
	}

	_, err = rClient.Project.ActionSetmembers(project, setProjectMembersInput)
	if err != nil {
		return err
	}

	return nil
}

func getProjectMembers(project *client.Project, rClient *client.RancherClient) ([]client.ProjectMember, error) {
	var projectMembers client.ProjectMemberCollection
	err := rClient.GetLink(project.Resource, "projectMembers", &projectMembers)
	return projectMembers.Data, err
}

func getProjectByName(name string, rClient *client.RancherClient) (*client.Project, error) {
	var project client.Project
	projects, err := rClient.Project.List(&client.ListOpts{})
	if err != nil {
		logrus.Fatalf("Could not get project ID for %s", name)
		return &project, nil
	}

	if len(projects.Data) == 0 {
		return &project, nil
	}

	for _, prj := range projects.Data {
		if prj.Name == name {
			project = prj
		}
	}

	return &project, nil
}
