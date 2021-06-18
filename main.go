package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/hashicorp/vault/api"
)

const (
	flagValues = "--values"
)

var (
	helmCmd       = getEnv("ARGOCD_HELM_VAULT_CMD", "_helm")
	roleID        = getEnv("ARGOCD_HELM_VAULT_ROLE_ID", "")
	secretID      = getEnv("ARGOCD_HELM_VAULT_SECRET_ID", "")
	replaceValues = getEnv("ARGOCD_HELM_VAULT_VALUES", "true")
	replaceOutput = getEnv("ARGOCD_HELM_VAULT_OUTPUT", "false")
	enabled       = getEnv("ARGOCD_HELM_VAULT_ENABLED", "true")

	regexPath, _   = regexp.Compile(`(?mU)<vault:(.+)\#(.+)>`)
	regexSyntax, _ = regexp.Compile(`(?mU)vault:(.+?)\#(.+?)`)
	regexPipe, _   = regexp.Compile(`\|(.*)`)
)

func main() {

	// check if enabled
	if enabled != "true" {
		data, err := cmd(helmCmd, os.Args[1:]...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", data)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stdout, "%s", data)
		os.Exit(0)
	}

	keys := map[string]map[string]interface{}{}

	// create client and login with vault AppRole
	vault, err := newVaultClient()
	if err != nil {
		fatal(err)
	}

	// replace values
	if replaceValues == "true" {
		flags := parseCmdFlags(flagValues)
		values := flags[flagValues]
		if len(values) > 0 {
			for _, file := range values {
				vaultReplaceFile(vault, keys, file)
			}
		}
	}

	// execute helm command
	data, err := cmd(helmCmd, os.Args[1:]...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", data)
		os.Exit(1)
	}

	// replace vault keys in the output
	if replaceOutput == "true" {
		output, find := vaultReplaceKeys(vault, keys, data)
		if find {
			fmt.Fprintf(os.Stdout, "%s---\n", output)
		} else {
			fmt.Fprintf(os.Stdout, "%s", data)
		}
	} else {
		fmt.Fprintf(os.Stdout, "%s", data)
	}

}

func vaultReplaceFile(vault *api.Client, keys map[string]map[string]interface{}, file string) {
	data, err := ioutil.ReadFile(file)
	fatal(err, "file", file)
	output, find := vaultReplaceKeys(vault, keys, data)
	if find {
		err = ioutil.WriteFile(file, []byte(output), 0644)
		fatal(err, "file", file)
	}
}

func vaultReplaceKeys(vault *api.Client, keys map[string]map[string]interface{}, value []byte) ([]byte, bool) {

	find := false
	result := regexPath.ReplaceAllFunc(value, func(match []byte) []byte {
		data := strings.Trim(string(match), "<>")
		pipe := ""

		// check pipe
		if regexPipe.MatchString(data) {
			tmp := regexPipe.FindStringSubmatch(data)
			pipe = strings.TrimSpace(string(tmp[1]))
			data = strings.TrimSpace(strings.Split(data, "|")[0])
		}

		// check pattern
		if regexSyntax.MatchString(data) {
			find = true

			// [input, path, key]
			matches := regexSyntax.FindStringSubmatch(data)
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
					if len(pipe) > 0 {
						switch pipe {
						case "b64enc":
							tmp := base64.StdEncoding.EncodeToString([]byte(v))
							return []byte(tmp)
						default:
							fatal(fmt.Errorf("not supported function %v", pipe))
						}
					}
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

func fatal(err error, data ...string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "[argocd-helm-vault] error: %v parameters: %v", err, data)
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

func parseCmdFlags(flags ...string) map[string][]string {
	values := map[string][]string{}

	for _, p := range flags {
		values[p] = []string{}
	}

	args := os.Args
	for i, a := range args {
		for _, p := range flags {
			if a == p && (i+1) < len(args) {
				values[p] = append(values[p], args[i+1])
			}
		}
	}
	return values
}
