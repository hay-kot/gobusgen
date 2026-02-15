package commands

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

// GenerateCmd implements the generate command
type GenerateCmd struct {
	flags *Flags
}

// NewGenerateCmd creates a new generate command
func NewGenerateCmd(flags *Flags) *GenerateCmd {
	return &GenerateCmd{flags: flags}
}

// Register adds the generate command to the application
func (cmd *GenerateCmd) Register(app *cli.Command) *cli.Command {
	app.Commands = append(app.Commands, &cli.Command{
		Name:  "generate",
		Usage: "generate command",
		Flags: []cli.Flag{
			// Add command-specific flags here
		},
		Action: cmd.run,
	})

	return app
}

func (cmd *GenerateCmd) run(ctx context.Context, c *cli.Command) error {
	log.Info().Msg("running generate command")

	fmt.Println("Hello World!")

	return nil
}
