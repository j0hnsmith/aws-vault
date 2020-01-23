package cli

import (
	"fmt"

	"github.com/99designs/aws-vault/vault"
	"github.com/99designs/keyring"
	"gopkg.in/alecthomas/kingpin.v2"
)

type AddYubikeyCommandInput struct {
	ProfileName  string
	Keyring      keyring.Keyring
	Username     string
	RequireTouch bool
}

func ConfigureAddYubikeyCommand(app *kingpin.Application) {
	input := AddYubikeyCommandInput{}

	cmd := app.Command("add-yubikey", "Adds a Yubikey as device")
	cmd.Arg("username", "Name of the user to add the Yubikey as device for").
		Required().
		StringVar(&input.Username)

	cmd.Arg("profile", "Name of the profile").
		Required().
		HintAction(awsConfigFile.ProfileNames).
		StringVar(&input.ProfileName)

	cmd.Flag("touch", "Require Yubikey touch to generate OTP").
		BoolVar(&input.RequireTouch)

	cmd.Action(func(c *kingpin.ParseContext) error {
		input.Keyring = keyringImpl
		AddYubikeyCommand(app, input)
		return nil
	})
}

func AddYubikeyCommand(app *kingpin.Application, input AddYubikeyCommandInput) {
	p, found := awsConfigFile.ProfileSection(input.ProfileName)
	if !found {
		app.Fatalf("Profile with name '%s' not found")
	}

	yubikey := vault.Yubikey{
		Keyring:        input.Keyring,
		Username:       input.Username,
		ProfileSection: p,
	}

	fmt.Printf("Adding yubikey to user %s using profile %s\n", input.Username, input.ProfileName)

	if err := yubikey.Register(p.Name, input.RequireTouch); err != nil {
		app.Fatalf("error registering yubikey", err)
	}

	fmt.Printf("Done!\n")
}
