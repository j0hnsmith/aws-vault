package cli

import (
	"fmt"

	"github.com/99designs/keyring"
	"github.com/aws/aws-sdk-go/aws/credentials"
	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/99designs/aws-vault/vault"
)

type RemoveYubikeyCommandInput struct {
	ProfileName string
	Keyring     keyring.Keyring
	Username    string
	Config      vault.Config
}

func ConfigureRemoveYubikeyCommand(app *kingpin.Application) {
	input := RemoveYubikeyCommandInput{}

	cmd := app.Command("remove-yubikey", "Removes Yubikey as a mfa device")
	cmd.Arg("username", "Name of the user to remove the Yubikey for").
		Required().
		StringVar(&input.Username)

	cmd.Arg("profile", "Name of the profile").
		Required().
		HintAction(awsConfigFile.ProfileNames).
		StringVar(&input.ProfileName)

	cmd.Action(func(c *kingpin.ParseContext) error {
		input.Keyring = keyringImpl
		RemoveYubikeyCommand(app, input)
		return nil
	})
}

func RemoveYubikeyCommand(app *kingpin.Application, input RemoveYubikeyCommandInput) {
	creds := credentials.NewEnvCredentials()
	val, err := creds.Get()
	if err != nil {
		app.Fatalf("Unable to get creds from env vars")
	}

	p, found := awsConfigFile.ProfileSection(input.ProfileName)
	if !found {
		app.Fatalf("Profile with name '%s' not found")
	}

	yubikey := vault.Yubikey{
		Keyring:        input.Keyring,
		Username:       input.Username,
		ProfileSection: p,
	}

	fmt.Printf("Removing yubikey for user %s using profile %s\n", input.Username, input.ProfileName)

	if err := yubikey.Remove(input.ProfileName, val); err != nil {
		app.Fatalf("error removing yubikey", err)
		return
	}

	fmt.Printf("Done!\n")
}
