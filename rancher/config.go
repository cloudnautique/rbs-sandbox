package rancher

import "github.com/rancher/go-rancher/client"

type RancherServerConfig struct {
	URL string
}

type RancherBootstrapConfig struct {
	Server              *RancherServerConfig
	LdapConfig          *client.Ldapconfig
	Accounts            map[string]*client.Account
	Projects            map[string]*client.Project
	Memberships         map[string]map[string]*client.Identity `json:"memberships" yaml:"memberships"`
	Registries          map[string][]client.Registry           `yaml:"registries"`
	RegistryCredentials map[string]map[string][]*client.RegistryCredential
}
