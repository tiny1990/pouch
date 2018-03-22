package main

import (
	"fmt"
	"os"
)

func main() {
	cli := NewCli()

	// set global flags for rootCmd in cli.
	cli.SetFlags()

	base := &baseCommand{cmd: cli.rootCmd, cli: cli}

	// Add all subcommands.
	cli.AddCommand(base, &PullCommand{})
	cli.AddCommand(base, &CreateCommand{})
	cli.AddCommand(base, &StartCommand{})
	cli.AddCommand(base, &StopCommand{})
	cli.AddCommand(base, &PsCommand{})
	cli.AddCommand(base, &RmCommand{})
	cli.AddCommand(base, &RestartCommand{})
	cli.AddCommand(base, &ExecCommand{})
	cli.AddCommand(base, &VersionCommand{})
	cli.AddCommand(base, &InfoCommand{})
	cli.AddCommand(base, &ImageMgmtCommand{})
	cli.AddCommand(base, &ImagesCommand{})
	cli.AddCommand(base, &RmiCommand{})
	cli.AddCommand(base, &VolumeCommand{})
	cli.AddCommand(base, &NetworkCommand{})

	cli.AddCommand(base, &InspectCommand{})
	cli.AddCommand(base, &RenameCommand{})
	cli.AddCommand(base, &PauseCommand{})
	cli.AddCommand(base, &UnpauseCommand{})
	cli.AddCommand(base, &RunCommand{})
	cli.AddCommand(base, &LoginCommand{})
	cli.AddCommand(base, &UpdateCommand{})
	cli.AddCommand(base, &LogoutCommand{})
	cli.AddCommand(base, &UpgradeCommand{})
	cli.AddCommand(base, &TopCommand{})
	cli.AddCommand(base, &LogsCommand{})

	// add generate doc command
	cli.AddCommand(base, &GenDocCommand{})

	if err := cli.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
