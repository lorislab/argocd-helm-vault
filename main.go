package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const (
	helmCommandTemplate = "template"
	helmFlagValues      = "--values"
)

var (
	helmCmd        = getEnv("ARGOCD_HELM_VAULT_CMD", "_helm3")
	regexPath, _   = regexp.Compile(`(?mU)<vault:(.+?)\#(.+?)>`)
	regexSyntax, _ = regexp.Compile(`(?mU)vault:(.+?)\#(.+?)`)
)

func main() {

	command := parseCommand()

	// before helm template update resources
	if command == helmCommandTemplate {

		// parse paraemters
		parameters := parseParameters(helmFlagValues)

		// update values
		values := parameters[helmFlagValues]
		if len(values) > 0 {

			vault, err := NewVaultClient()
			if err != nil {
				fatal(err)
			}
			keys := map[string]map[string]interface{}{}
			for _, file := range values {
				vaultFile(vault, keys, file)
			}
		}
	}

	// execute helm command
	cmd(helmCmd, os.Args[1:]...)
}

func vaultFile(vault *VaultClient, keys map[string]map[string]interface{}, file string) {
	data, err := ioutil.ReadFile(file)
	fatal(err, "file", file)
	output, find := vaultReplaceKeys(vault, keys, data)
	if find {
		err = ioutil.WriteFile(file, []byte(output), 0644)
		fatal(err, "file", file)
	}
}

func vaultReplaceKeys(vault *VaultClient, keys map[string]map[string]interface{}, value []byte) ([]byte, bool) {
	find := false
	result := regexPath.ReplaceAllFunc(value, func(match []byte) []byte {
		data := strings.Trim(string(match), "<>")
		trim := []byte(data)
		if regexSyntax.Match(trim) {
			find = true

			// [input, path, key]
			matches := regexSyntax.FindSubmatch(trim)
			path := string(matches[1])

			secrets := keys[path]
			if secrets == nil {
				if !vault.IsLogin() {
					err := vault.Login()
					if err != nil {
						fatal(err)
					}
				}
				secrets, err := vault.GetSecrets(path)
				if err != nil {
					fatal(err)
				}
				keys[path] = secrets
			}

			value, e := secrets[string(matches[2])]
			if e && value != nil {
				return value.([]byte)
			}
		}
		return match
	})
	return result, find
}

func cmd(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		os.Exit(1)
	}
}

func parseCommand() string {
	tmp := os.Args[1:]
	if len(tmp) > 0 {
		return tmp[0]
	}
	return ""
}

func parseParameters(params ...string) map[string][]string {
	values := map[string][]string{}

	for _, p := range params {
		values[p] = []string{}
	}

	args := os.Args
	for i, a := range args {
		for _, p := range params {
			if a == p && (i+1) < len(args) {
				values[p] = append(values[p], args[i+1])
			}
		}
	}
	return values
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
