package main

import (
	"errors"
	"fmt"

	"github.com/hashicorp/vault/api"
)

type VaultClient struct {
	Client  *api.Client
	appRole map[string]interface{}
	login   bool
}

func NewVaultClient() (*VaultClient, error) {
	apiClient, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return nil, err
	}

	return &VaultClient{Client: apiClient,
		appRole: map[string]interface{}{
			"role_id":   getEnv("ARGOCD_HELM_VAULT_ROLE_ID", ""),
			"secret_id": getEnv("ARGOCD_HELM_VAULT_SECRET_ID", ""),
		},
	}, nil
}

func (c *VaultClient) Login() error {
	data, err := c.Client.Logical().Write("auth/approle/login", c.appRole)
	if err != nil {
		return err
	}
	c.Client.SetToken(data.Auth.ClientToken)
	c.login = true
	return nil
}

func (c *VaultClient) IsLogin() bool {
	return c.login
}

func (v *VaultClient) GetSecrets(path string) (map[string]interface{}, error) {
	secret, err := v.Client.Logical().Read(path)
	if err != nil {
		return nil, err
	}
	if _, ok := secret.Data["data"]; ok {
		return secret.Data["data"].(map[string]interface{}), nil
	}
	if len(secret.Data) == 0 {
		return nil, fmt.Errorf("The Vault path: %s is empty - did you forget to include /data/ in the Vault path for kv-v2?", path)
	}
	return nil, errors.New("Could not get data from Vault, check that kv-v2 is the correct engine")
}
