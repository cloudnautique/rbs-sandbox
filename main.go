package main

import (
	"github.com/BurntSushi/toml"
	"github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/client"
)

const (
	rancherURL = "http://192.168.99.100"
	configFile = "./config.toml"
)

type tomlConfig struct {
	rancherUrl  string
	LdapConfig  *client.Ldapconfig
	Accounts    map[string]*client.Account
	Projects    map[string]*client.Project
	Memberships map[string]map[string]*client.ProjectMember
}

type rancherServer struct {
	client *client.RancherClient
	config *tomlConfig
}

func newRancherServer() *rancherServer {
	config := &tomlConfig{}

	opts := &client.ClientOpts{
		Url:       rancherURL,
		AccessKey: "admin",
		SecretKey: "admin",
	}

	rClient, _ := getRancherClient(opts)
	//adminKeys, _ := generateAdminApiKeys(rClient)
	//rClient.Opts.AccessKey = adminKeys.PublicValue
	//rClient.Opts.SecretKey = adminKeys.SecretValue

	logrus.Infof("Using Access Key: %s", rClient.Opts.AccessKey)

	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		logrus.Fatalf("Could not parse config: %s", err)
	}

	return &rancherServer{
		client: rClient,
		config: config,
	}
}

func main() {
	rancherServer := newRancherServer()
	//rancherServer.enableAdAuth()
	//rancherServer.setupAccounts()
	//rancherServer.removeDefaultProject()
	//rancherServer.addProjects()
	rancherServer.addMembers()
}

func generateAdminApiKeys(rClient *client.RancherClient) (*client.ApiKey, error) {
	return rClient.ApiKey.Create(&client.ApiKey{
		AccountId: "1a1",
	})
}

func getRancherClient(opts *client.ClientOpts) (*client.RancherClient, error) {
	return client.NewRancherClient(opts)
}

func (r *rancherServer) enableAdAuth() {
	rancherClient, err := getRancherClient(&client.ClientOpts{
		Url: rancherURL,
	})
	if err != nil {
		logrus.Fatalf("Error getting Rancher Client: %s", err)
	}

	if _, err = rancherClient.Ldapconfig.Create(r.config.LdapConfig); err != nil {
		logrus.Fatal(err)
	}

	logrus.Infof("enabled Ldap config")
}

func (r *rancherServer) setupAccounts() {
	for key, acct := range r.config.Accounts {
		logrus.Infof("Adding Acct: %s", key)
		r.client.Account.Create(acct)
	}
}

func (r *rancherServer) removeDefaultProject() {
	projects, err := r.client.Project.List(&client.ListOpts{})
	if err != nil {
		logrus.Fatalf("Could not list projects")
	}

	for _, project := range projects.Data {
		logrus.Infof("Found project: %s", project.Name)
		if project.Name == "Default" {
			logrus.Infof("Removing default Project")
			if err = r.client.Project.Delete(&project); err != nil {
				logrus.Fatalf("Error removing default project: %s", err)
			}
		}
	}
}

func (r *rancherServer) addProjects() {
	for key, value := range r.config.Projects {
		logrus.Infof("Adding project: %s", key)
		r.client.Project.Create(value)
	}
}

func (r *rancherServer) addMembers() {
	for projectName, projectMembers := range r.config.Memberships {
		project, err := r.findExistingProject(projectName)
		if err != nil {
			logrus.Fatalf("Could not find project %s", projectName)
		}

		setProjectMembersInput, _ := r.client.SetProjectMembersInput.List(&client.ListOpts{
			Filters: map[string]interface{}{}
		})
		members, err := r.getProjectMembers(project)
		if err != nil {
			logrus.Fatalf("Error: %s", err)
		}

		for _, member := range projectMembers {
			members = append(members, *member)
		}

		logrus.Infof("members %s", members)
	}
}

func (r *rancherServer) getProjectMembers(project *client.Project) ([]client.ProjectMember, error) {
	var projectMembers client.ProjectMemberCollection
	err := r.client.GetLink(project.Resource, "projectMembers", &projectMembers)
	return projectMembers.Data, err
}

func (r *rancherServer) resolveProjectId(name string) (string, error) {
	projects, err := r.client.Project.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"name":         name,
			"removed_null": nil,
		},
	})
	if err != nil {
		logrus.Fatalf("Could not get project ID for %s", name)
		return "", nil
	}

	if len(projects.Data) == 0 {
		return "", nil
	}

	return projects.Data[0].Id, nil
}

func (r *rancherServer) findExistingProject(name string) (*client.Project, error) {
	logrus.Debugf("Finding Project: %s", name)

	projectId, err := r.resolveProjectId(name)
	if err != nil {
		logrus.Fatalf("Could not get project Id for %s", name)
		return nil, err
	}

	projects, err := r.client.Project.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"Id":   projectId,
			"name": name,
		},
	})

	if len(projects.Data) == 0 {
		return nil, nil
	}
	return &projects.Data[0], nil
}
