package rancher

import (
	"github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/client"
)

func (r *RancherServer) ConfigureRegistries() error {
	for projectName, configProjectRegistries := range r.config.Registries {

		logrus.Infof("Configuring registries for project: %s", projectName)
		project, err := getProjectByName(projectName, r.client)
		if err != nil {
			return err
		}

		projectKeys, err := generateProjectApiKeys(r.client, project)
		if err != nil {
			logrus.Errorf("Unable to create project keys")
			return err
		}
		defer r.client.ApiKey.Delete(projectKeys)

		projectClientOpts := &client.ClientOpts{
			Url:       r.config.Server.URL + "/projects/" + project.Id,
			AccessKey: projectKeys.PublicValue,
			SecretKey: projectKeys.SecretValue,
		}

		projectClient, err := getRancherClient(projectClientOpts)
		if err != nil {
			logrus.Errorf("Could not get client")
			return err
		}
		logrus.Debugf("Created client for project: %s", projectName)

		projectRegistries, err := getProjectRegistries(projectClient, project)
		if err != nil {
			logrus.Errorf("Unable to get registry list for project: %s", projectName)
			return err
		}

		for _, registry := range configProjectRegistries {
			registry.AccountId = project.Id
			registryExists := registryExists(projectRegistries, registry)

			if registry.State == "Purged" && registryExists {
				logrus.Infof("Removing registry: %s", registry.ServerAddress)

				registry = getExistingRegistry(projectRegistries, registry)
				err = deleteRegistry(registry, projectClient)
			} else if !registryExists && registry.State != "Purged" {
				logrus.Infof("Adding registry: %s", registry.ServerAddress)

				createdReg, err := addRegistry(registry, projectClient)
				if err != nil {
					return err
				}

				registry = *createdReg
				if credentials, ok := r.config.RegistryCredentials[project.Name][registry.ServerAddress]; ok {
					logrus.Infof("Configuring credentials for: %s", registry.ServerAddress)
					ConfigureRegistryCredentials(projectClient, registry, project, credentials)
				}
			} else if registryExists {
				logrus.Infof("Registry: %s exists", registry.ServerAddress)
				registry = getExistingRegistry(projectRegistries, registry)

				if credentials, ok := r.config.RegistryCredentials[project.Name][registry.ServerAddress]; ok {
					logrus.Infof("Configuring credentials for: %s", registry.ServerAddress)
					ConfigureRegistryCredentials(projectClient, registry, project, credentials)
				}
			} else {
				logrus.Infof("Registry: %s does not exist", registry.ServerAddress)
			}
		}

	}

	return nil
}

func ConfigureRegistryCredentials(rClient *client.RancherClient, registry client.Registry, project *client.Project, configCredentials []*client.RegistryCredential) error {
	existingCredentials, err := getRegistryCredentials(rClient, project)
	if err != nil {
		return err
	}
	for _, credential := range configCredentials {
		credential.RegistryId = registry.Id
		if !registryCredentialExists(existingCredentials, credential) {
			if _, err := rClient.RegistryCredential.Create(credential); err != nil {
				return err
			}
		}
	}
	return nil
}

func registryCredentialExists(collection client.RegistryCredentialCollection, credential *client.RegistryCredential) bool {
	exists := false
	for _, cred := range collection.Data {
		if cred.Email == credential.Email && cred.Kind == "registryCredential" {
			exists = true
		}
	}
	return exists
}

func getRegistryCredentials(rClient *client.RancherClient, project *client.Project) (client.RegistryCredentialCollection, error) {
	var credentialsCollection client.RegistryCredentialCollection
	err := rClient.GetLink(project.Resource, "credentials", &credentialsCollection)
	return credentialsCollection, err
}

func getProjectRegistries(rClient *client.RancherClient, project *client.Project) (client.RegistryCollection, error) {
	logrus.Infof("Getting registries for: %s", project.Id)

	var registries client.RegistryCollection
	err := rClient.GetLink(project.Resource, "registries", &registries)

	return registries, err
}

func registryExists(collection client.RegistryCollection, registry client.Registry) bool {
	exists := false

	for _, reg := range collection.Data {
		if registry.ServerAddress == reg.ServerAddress {
			exists = true
		}
	}
	return exists
}

func getExistingRegistry(collection client.RegistryCollection, registry client.Registry) client.Registry {
	var returnRegistry client.Registry
	for _, reg := range collection.Data {
		if registry.ServerAddress == reg.ServerAddress {
			returnRegistry = reg
		}
	}
	return returnRegistry
}

func deleteRegistry(registry client.Registry, prjClient *client.RancherClient) error {
	logrus.Infof("deactivating: %#v", registry.Id)

	_, err := prjClient.Registry.ActionDeactivate(&registry)
	if err != nil {
		return err
	}
	WaitFor(prjClient, &registry.Resource, registry, func() string {
		return registry.Transitioning
	})

	return prjClient.Registry.Delete(&registry)
}

func addRegistry(registry client.Registry, prjClient *client.RancherClient) (*client.Registry, error) {
	return prjClient.Registry.Create(&registry)
}
