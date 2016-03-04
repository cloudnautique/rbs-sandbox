package rancher

import (
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/cloudfoundry-incubator/candiedyaml"
	"github.com/rancher/go-rancher/client"
)

type RancherServer struct {
	client *client.RancherClient
	config *RancherBootstrapConfig
}

func NewRancherServer(configFile string, keyFile string) *RancherServer {
	file, err := os.Open(configFile)
	if err != nil {
		logrus.Fatalf("File does not exist: %s\n%s", configFile, err)
	}
	defer file.Close()

	config := &RancherBootstrapConfig{}
	decoder := candiedyaml.NewDecoder(file)
	if err = decoder.Decode(&config); err != nil {
		logrus.Fatalf("Could not parse config: %s", err.Error())
	}

	logrus.Infof("Using Rancher URL: %s", config.Server.URL)

	opts := &client.ClientOpts{
		Url: config.Server.URL,
	}

	setOptAdminKeys(opts, keyFile)

	rClient, _ := getRancherClient(opts)

	if opts.AccessKey == "" || opts.SecretKey == "" {
		adminKeys, _ := generateAndSetAdminApiKeys(rClient, keyFile)
		rClient.Opts.AccessKey = adminKeys.PublicValue
		rClient.Opts.SecretKey = adminKeys.SecretValue
	}

	logrus.Infof("Using Access Key: %s", rClient.Opts.AccessKey)

	return &RancherServer{
		client: rClient,
		config: config,
	}
}

func setOptAdminKeys(opts *client.ClientOpts, keyFile string) error {
	file, err := os.Open(keyFile)
	if err != nil {
		logrus.Warnf("Keys could not be opened\n %s", err.Error())
		return err
	}
	defer file.Close()

	keys := make(map[string]string)
	decoder := candiedyaml.NewDecoder(file)
	if err = decoder.Decode(&keys); err != nil {
		logrus.Fatalf("Could not parse keys: %s", err.Error())
	}

	if accessKey, ok := keys["access_key"]; ok {
		opts.AccessKey = accessKey
	}

	if secretKey, ok := keys["secret_key"]; ok {
		opts.SecretKey = secretKey
	}

	return nil
}

func (r *RancherServer) ConfigureAuthBackend() error {
	if r.config.LdapConfig == nil {
		logrus.Warn("No Ldap configuration found")
		return nil
	}

	enabled, err := ldapconfigEnabled(r.client)
	if enabled != true {
		logrus.Infof("enabling Ldap config")
		_, err = r.client.Ldapconfig.Create(r.config.LdapConfig)
	}

	return err
}

func (r *RancherServer) ConfigureAccounts() error {
	accounts, err := r.client.Account.List(&client.ListOpts{})
	if err != nil {
		return err
	}

	for key, acct := range r.config.Accounts {
		if !accountExists(accounts, acct) {
			logrus.Infof("Adding Acct: %s", key)
			if _, err := r.client.Account.Create(acct); err != nil {
				return err
			}
		}
	}
	return nil
}

func accountExists(collection *client.AccountCollection, account *client.Account) bool {
	exists := false
	for _, acct := range collection.Data {
		if account.ExternalId == acct.ExternalId {
			exists = true
		}
	}
	return exists
}

func (r *RancherServer) ConfigureEnvironments() error {
	for _, project := range r.config.Projects {
		if project.State == "Purged" {
			prj, err := getProjectByName(project.Name, r.client)
			if err != nil {
				return err
			}

			if err := removeProject(prj, r.client); err != nil {
				return err
			}
		} else {
			if err := addProject(project, r.client); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *RancherServer) ConfigureEnvironmentAccess() error {
	for projectName, newProjectMembers := range r.config.Memberships {
		existingProjectMembers, err := getExistingProjectMembers(projectName, r.client)
		if err != nil {
			return err
		}

		for _, member := range newProjectMembers {
			newMember, err := getProjectMemberIdentity(member.Name, member.Role, r.client)
			if err != nil {
				return err
			}
			existingProjectMembers = append(existingProjectMembers, newMember)
		}

		if err = addEnvironmentMembers(projectName, existingProjectMembers, r.client); err != nil {
			return err
		}
	}
	return nil
}

func projectExists(prj *client.Project, rClient *client.RancherClient) bool {
	exists := false

	projects, err := rClient.Project.List(&client.ListOpts{})
	if err != nil {
		logrus.Fatalf("Could not list projects")
	}

	for _, project := range projects.Data {
		if project.Name == prj.Name {
			exists = true
		}
	}

	return exists
}
func addProject(prj *client.Project, rClient *client.RancherClient) error {
	if !projectExists(prj, rClient) {
		logrus.Infof("Addingproject: %s", prj.Name)
		_, err := rClient.Project.Create(prj)
		return err
	}

	return nil
}

func removeProject(project *client.Project, rClient *client.RancherClient) error {
	if projectExists(project, rClient) {
		logrus.Infof("Removing project: %s", project.Name)
		if err := rClient.Project.Delete(project); err != nil {
			logrus.Fatalf("Error removing project: %s\n%s", project.Name, err)
		}
	}

	return nil
}

func generateAndSetAdminApiKeys(rClient *client.RancherClient, keyFile string) (*client.ApiKey, error) {
	apiKey, err := rClient.ApiKey.Create(&client.ApiKey{
		AccountId: "1a1",
	})
	if err != nil {
		return apiKey, err
	}

	keyDataOut := make(map[string]string)
	keyDataOut["access_key"] = apiKey.PublicValue
	keyDataOut["secret_key"] = apiKey.SecretValue

	fileToWrite, err := os.Create(keyFile)
	if err != nil {
		logrus.Fatalf("Could not write out keys: %s", err)
	}

	encoder := candiedyaml.NewEncoder(fileToWrite)
	err = encoder.Encode(keyDataOut)
	if err != nil {
		logrus.Fatalf("Failed to encode keys: %s", err)
	}

	return apiKey, err
}

func generateProjectApiKeys(rClient *client.RancherClient, project *client.Project) (*client.ApiKey, error) {
	logrus.Infof("Creating key for: %s", project.Id)
	apiKey, err := rClient.ApiKey.Create(&client.ApiKey{
		AccountId: project.Id,
	})
	if err != nil {
		return apiKey, err
	}

	return apiKey, WaitFor(rClient, &apiKey.Resource, apiKey, func() string {
		return apiKey.Transitioning
	})
}

func getRancherClient(opts *client.ClientOpts) (*client.RancherClient, error) {
	return client.NewRancherClient(opts)
}

func ldapconfigEnabled(rClient *client.RancherClient) (bool, error) {
	ldapconfig, err := rClient.Ldapconfig.List(&client.ListOpts{})
	return ldapconfig.Data[0].Enabled, err
}
