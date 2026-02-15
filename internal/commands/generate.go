package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"

	"github.com/hay-kot/gobusgen/internal/generator"
	"github.com/hay-kot/gobusgen/internal/parser"
)

// GenerateCmd implements the generate command
type GenerateCmd struct {
	flags  *Flags
	output string
}

// NewGenerateCmd creates a new generate command
func NewGenerateCmd(flags *Flags) *GenerateCmd {
	return &GenerateCmd{flags: flags}
}

// Register adds the generate command to the application
func (cmd *GenerateCmd) Register(app *cli.Command) *cli.Command {
	app.Commands = append(app.Commands, &cli.Command{
		Name:                      "generate",
		Usage:                     "generate type-safe event bus from map variable declaration",
		DisableSliceFlagSeparator: true,
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:    "package",
				Aliases: []string{"p"},
				Usage:   "package target as <dirpath>.<VarName> (repeatable)",
			},
			&cli.StringFlag{
				Name:        "output",
				Aliases:     []string{"o"},
				Usage:       "output file path (only valid with a single --package target)",
				Destination: &cmd.output,
			},
		},
		Action: cmd.run,
	})

	return app
}

// defaultOutputFilename returns the output filename based on the prefix.
// Empty prefix → "eventbus.gen.go", "Command" → "commandbus.gen.go".
func defaultOutputFilename(prefix string) string {
	if prefix == "" {
		return "eventbus.gen.go"
	}
	return strings.ToLower(prefix) + "bus.gen.go"
}

// parseTarget splits "path/to/pkg.VarName" on the last dot.
// A leading dot with no preceding path (e.g. ".Events") is treated as
// the current directory ".".
func parseTarget(s string) (dir, varName string, err error) {
	i := strings.LastIndex(s, ".")
	if i < 0 {
		return "", "", fmt.Errorf("invalid target %q: missing '.' separator", s)
	}

	dir = s[:i]
	varName = s[i+1:]

	if varName == "" {
		return "", "", fmt.Errorf("invalid target %q: empty variable name", s)
	}

	if dir == "" {
		dir = "."
	}

	return dir, varName, nil
}

func (cmd *GenerateCmd) run(ctx context.Context, c *cli.Command) error {
	targets := c.StringSlice("package")
	if len(targets) == 0 {
		targets = []string{".Events"}
	}

	if cmd.output != "" && len(targets) > 1 {
		return fmt.Errorf("--output cannot be used with multiple --package targets")
	}

	for _, t := range targets {
		dir, varName, err := parseTarget(t)
		if err != nil {
			return err
		}

		log.Debug().
			Str("dir", dir).
			Str("var", varName).
			Msg("parsing event definitions")

		input, err := parser.Parse(dir, varName)
		if err != nil {
			return fmt.Errorf("parsing events in %s: %w", t, err)
		}

		output := cmd.output
		if output == "" {
			output = filepath.Join(dir, defaultOutputFilename(input.Prefix))
		}

		log.Debug().Int("events", len(input.Events)).Str("prefix", input.Prefix).Msg("generating event bus")

		src, err := generator.Generate(input)
		if err != nil {
			return fmt.Errorf("generating code for %s: %w", t, err)
		}

		if err := os.WriteFile(output, src, 0o644); err != nil {
			return fmt.Errorf("writing output for %s: %w", t, err)
		}

		log.Info().
			Str("output", output).
			Int("events", len(input.Events)).
			Msg("generated event bus")
	}

	return nil
}
