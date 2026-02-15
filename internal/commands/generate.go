package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"

	"github.com/hay-kot/gobusgen/internal/generator"
	"github.com/hay-kot/gobusgen/internal/parser"
)

// GenerateCmd implements the generate command
type GenerateCmd struct {
	flags *Flags

	dir     string
	output  string
	varName string
}

// NewGenerateCmd creates a new generate command
func NewGenerateCmd(flags *Flags) *GenerateCmd {
	return &GenerateCmd{flags: flags}
}

// Register adds the generate command to the application
func (cmd *GenerateCmd) Register(app *cli.Command) *cli.Command {
	app.Commands = append(app.Commands, &cli.Command{
		Name:  "generate",
		Usage: "generate type-safe event bus from map variable declaration",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "dir",
				Aliases:     []string{"d"},
				Usage:       "directory containing Go source with event map",
				Value:       ".",
				Destination: &cmd.dir,
			},
			&cli.StringFlag{
				Name:        "output",
				Aliases:     []string{"o"},
				Usage:       "output file path (default: <dir>/eventbus.gen.go)",
				Destination: &cmd.output,
			},
			&cli.StringFlag{
				Name:        "var",
				Usage:       "name of the map variable to parse",
				Value:       "Events",
				Destination: &cmd.varName,
			},
		},
		Action: cmd.run,
	})

	return app
}

func (cmd *GenerateCmd) run(ctx context.Context, c *cli.Command) error {
	dir := cmd.dir
	output := cmd.output
	if output == "" {
		output = filepath.Join(dir, "eventbus.gen.go")
	}

	log.Debug().
		Str("dir", dir).
		Str("output", output).
		Str("var", cmd.varName).
		Msg("parsing event definitions")

	input, err := parser.Parse(dir, cmd.varName)
	if err != nil {
		return fmt.Errorf("parsing events: %w", err)
	}

	log.Debug().Int("events", len(input.Events)).Msg("generating event bus")

	src, err := generator.Generate(input)
	if err != nil {
		return fmt.Errorf("generating code: %w", err)
	}

	if err := os.WriteFile(output, src, 0o644); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	log.Info().
		Str("output", output).
		Int("events", len(input.Events)).
		Msg("generated event bus")

	return nil
}
