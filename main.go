package main

import (
	"os"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/99designs/aws-vault/cli"
)

// Version is provided at compile time
var Version = "dev"

func main() {
	run(os.Args[1:], os.Exit)
}

func run(args []string, exit func(int)) {
	app := kingpin.New(
		`aws-vault`,
		`A vault for securely storing and accessing AWS credentials in development environments.`,
	)

	app.ErrorWriter(os.Stderr)
	app.Writer(os.Stdout)
	app.Version(Version)
	app.Terminate(exit)

	cli.ConfigureGlobals(app)
	cli.ConfigureExecCommand(app)
	cli.ConfigureLoginCommand(app)
	cli.ConfigureAddCommand(app)
	cli.ConfigureRemoveCommand(app)
	cli.ConfigureListCommand(app)
	cli.ConfigureAddMfaCommand(app)
	cli.ConfigureRemoveMfaCommand(app)
	cli.ConfigureRotateCommand(app)
	cli.ConfigureServerCommand(app)

	kingpin.MustParse(app.Parse(args))
}
