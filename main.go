package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/hashicorp/vault/api"
)

var (
	helmCmd        = getEnv("ARGOCD_HELM_VAULT_CMD", "_helm")
	roleID         = getEnv("ARGOCD_HELM_VAULT_ROLE_ID", "")
	secretID       = getEnv("ARGOCD_HELM_VAULT_SECRET_ID", "")
	regexPath, _   = regexp.Compile(`(?mU)<vault:(.+?)\#(.+?)>`)
	regexSyntax, _ = regexp.Compile(`(?mU)vault:(.+?)\#(.+?)`)
)

func main() {

	// execute helm command
	data, err := cmd(helmCmd, os.Args[1:]...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", data)
		os.Exit(1)
	}

	// create client and login
	vault, err := newVaultClient()
	if err != nil {
		fatal(err)
	}

	// replace vault keys
	output, find := vaultReplaceKeys(vault, data)
	if find {
		fmt.Fprintf(os.Stdout, "%s---\n", output)
	} else {
		fmt.Fprintf(os.Stdout, "%s", data)
	}
}

func vaultReplaceKeys(vault *api.Client, value []byte) ([]byte, bool) {

	keys := map[string]map[string]interface{}{}

	find := false
	result := regexPath.ReplaceAllFunc(value, func(match []byte) []byte {
		data := strings.Trim(string(match), "<>")
		trim := []byte(data)
		if regexSyntax.Match(trim) {
			find = true

			// [input, path, key]
			matches := regexSyntax.FindSubmatch(trim)
			path := strings.TrimSpace(string(matches[1]))

			secrets := keys[path]
			if secrets == nil {
				tmp, err := getSecrets(vault, path)
				if err != nil {
					fatal(err)
				}
				keys[path] = tmp
				secrets = keys[path]
			}

			value, e := secrets[strings.TrimSpace(string(matches[2]))]
			if e && value != nil {
				switch v := value.(type) {
				case string:
					return []byte(v)
				default:
					return match
				}
			}
		}
		return match
	})
	return result, find
}

func cmd(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Env = os.Environ()
	return cmd.CombinedOutput()
}

func getEnv(name, defaultValue string) string {
	val, e := os.LookupEnv(name)
	if e {
		return val
	}
	return defaultValue
}

func fatal(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "[argocd-helm-vault] error: %v", err)
		os.Exit(1)
	}
}

func newVaultClient() (*api.Client, error) {
	apiClient, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return nil, err
	}

	appRole := map[string]interface{}{
		"role_id":   roleID,
		"secret_id": secretID,
	}

	data, err := apiClient.Logical().Write("auth/approle/login", appRole)
	if err != nil {
		return nil, err
	}
	apiClient.SetToken(data.Auth.ClientToken)
	return apiClient, nil
}

func getSecrets(client *api.Client, path string) (map[string]interface{}, error) {
	secret, err := client.Logical().Read(path)
	if err != nil {
		return nil, err
	}
	if _, ok := secret.Data["data"]; ok {
		return secret.Data["data"].(map[string]interface{}), nil
	}
	if len(secret.Data) == 0 {
		return nil, fmt.Errorf("path: %s is empty - did you forget to include <engine>/data/<path> in the Vault path for kv-v2?", path)
	}
	return nil, errors.New("could not get data from Vault, check the configuration")
}
