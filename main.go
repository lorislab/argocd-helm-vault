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

		// generated value files
		values := parseValues()
		if len(values) > 0 {
			for _, file := range values {
				vaultFile(file)
			}
		}
	}

	// execute helm command
	cmd(helmCmd, os.Args[1:]...)
}

func vaultFile(file string) {
	data, err := ioutil.ReadFile(file)
	fatal(err)
	output, find := vaultReplaceKeys(data)
	if find {
		err = ioutil.WriteFile(file, []byte(output), 0644)
		fatal(err)
	}
}

func vaultReplaceKeys(value []byte) ([]byte, bool) {
	find := false
	result := regexPath.ReplaceAllFunc(value, func(match []byte) []byte {
		data := strings.Trim(string(match), "<>")
		trim := []byte(data)
		if regexSyntax.Match(trim) {
			find = true

			// [input, path, key]
			matches := regexSyntax.FindSubmatch(trim)

			// call vault to get the value for the key
			if len(matches) > 2 {
				return cmdOutput("vault", "kv", "get", "-format=yaml", "-field="+string(matches[2]), string(matches[1]))
			}
		}
		return match
	})
	return result, find
}

func cmdOutput(name string, args ...string) []byte {
	cmd := exec.Command(name, args...)
	data, err := cmd.Output()
	if err != nil {
		fatal(err)
	}
	return data
}

func cmd(name string, args ...string) {
	cmd := exec.Command(name, args...)
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

func parseValues() []string {
	values := []string{}
	args := os.Args
	for i, a := range args {
		if a == helmFlagValues && (i+1) < len(args) {
			values = append(values, args[i+1])
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

func fatal(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v", err)
		os.Exit(1)
	}
}
