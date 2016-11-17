package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"koding/kites/kloud/utils/object"
	"koding/klientctl/kloud/credential"

	"github.com/codegangsta/cli"
	"github.com/koding/logging"
	"golang.org/x/crypto/ssh/terminal"
)

func CredentialList(c *cli.Context, log logging.Logger, _ string) (int, error) {
	opts := &credential.ListOptions{
		Provider: c.String("provider"),
		Team:     c.String("team"),
	}

	creds, err := credential.List(opts)
	if err != nil {
		return 0, err
	}

	if len(creds) == 0 {
		fmt.Fprintln(os.Stderr, "You have no matching credentials attached to your Koding account.")
		return 0, nil
	}

	if c.Bool("json") {
		object.JSONPrinter.Print(creds)
		return 0, nil
	}

	object.TabPrinter.Print(creds.ToSlice())

	return 0, nil
}

func ask(format string, args ...interface{}) (string, error) {
	fmt.Printf(format, args...)
	s, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(s), nil
}

func askSecret(format string, args ...interface{}) (string, error) {
	fmt.Printf(format, args...)
	p, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return string(p), nil
}

func AskCredential(c *cli.Context) (provider string, creds map[string]string, err error) {
	descs, err := credential.Describe()
	if err != nil {
		return "", nil, err
	}

	provider = c.String("provider")

	if provider == "" {
		s, err := ask("Provider type []: ")
		if err != nil {
			return "", nil, err
		}

		provider = s
	}

	desc, ok := descs[provider]
	if !ok {
		return "", nil, fmt.Errorf("provider %q does not exist", provider)
	}

	creds = make(map[string]string, len(desc.Credential))

	// TODO(rjeczalik): Add field.OmitEmpty so we validate required
	// fields client-side.
	//
	// TODO(rjeczalik): Refactor part which validates credential
	// input on kloud/provider side to a separate library
	// and use it here.
	for _, field := range desc.Credential {
		var value string

		if field.Secret {
			value, err = askSecret("%s [***]: ", field.Label)
		} else {
			var defaultValue string

			if len(field.Values) > 0 {
				if s, ok := field.Values[0].Value.(string); ok {
					defaultValue = s
				}
			}

			value, err = ask("%s [%s]: ", field.Label, defaultValue)

			if value == "" {
				value = defaultValue
			}
		}

		if err != nil {
			return "", nil, err
		}

		creds[field.Name] = value
	}

	return provider, creds, nil
}

func CredentialCreate(c *cli.Context, log logging.Logger, _ string) (int, error) {
	var p []byte
	var err error
	var provider = c.String("provider")

	switch file := c.String("file"); file {
	case "":
		s, creds, err := AskCredential(c)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error building credential data:", err)
			return 1, err
		}

		p, err = json.Marshal(creds)
		provider = s
	case "-":
		p, err = ioutil.ReadAll(os.Stdin)
	default:
		p, err = ioutil.ReadFile(file)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading credential file:", err)
		return 1, err
	}

	fmt.Fprintln(os.Stderr, "Creating credential... ")

	opts := &credential.CreateOptions{
		Provider: provider,
		Team:     c.String("team"),
		Title:    c.String("title"),
		Data:     p,
	}

	cred, err := credential.Create(opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating credential:", err)
		return 1, err
	}

	fmt.Fprintf(os.Stderr, "Created %q credential with %s identifier.\n", cred.Title, cred.Identifier)

	return 0, nil
}
