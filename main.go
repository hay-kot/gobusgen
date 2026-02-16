package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"

	"github.com/hay-kot/gobusgen/internal/commands"
)

var (
	// Build information. Populated at build-time via -ldflags flag.
	version = "dev"
	commit  = "HEAD"
	date    = "now"
)

func build() string {
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			version = info.Main.Version
			for _, s := range info.Settings {
				switch s.Key {
				case "vcs.revision":
					commit = s.Value
				case "vcs.time":
					date = s.Value
				}
			}
		}
	}

	short := commit
	if len(commit) > 7 {
		short = commit[:7]
	}

	return fmt.Sprintf("%s (%s) %s", version, short, date)
}

func setupLogger(level string, noColor bool) error {
	parsedLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		return fmt.Errorf("failed to parse log level: %w", err)
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: noColor}).Level(parsedLevel)

	return nil
}

func run() (noColor bool, err error) {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	flags := &commands.Flags{}

	app := &cli.Command{
		Name:                  "gobusgen",
		Usage:                 `A generator for in-process event bus with type safe wrappers`,
		Version:               build(),
		EnableShellCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "log-level",
				Usage:       "log level (debug, info, warn, error, fatal, panic)",
				Sources:     cli.EnvVars("LOG_LEVEL"),
				Value:       "info",
				Destination: &flags.LogLevel,
			},
			&cli.BoolFlag{
				Name:        "no-color",
				Usage:       "disable colored output",
				Sources:     cli.EnvVars("NO_COLOR"),
				Destination: &flags.NoColor,
			},
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			if err := setupLogger(flags.LogLevel, flags.NoColor); err != nil {
				return ctx, err
			}

			return ctx, nil
		},
	}
	app = commands.NewGenerateCmd(flags).Register(app)
	// +scaffold:command:register

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return flags.NoColor, app.Run(ctx, os.Args)
}

func main() {
	noColor, err := run()
	if err != nil {
		colorRed := "\033[38;2;215;95;107m"
		colorGray := "\033[38;2;163;163;163m"
		colorReset := "\033[0m"
		if noColor {
			colorRed = ""
			colorGray = ""
			colorReset = ""
		}
		fmt.Fprintf(os.Stderr, "\n%s╭ Error%s\n%s│%s %s%s%s\n%s╵%s\n",
			colorRed, colorReset,
			colorRed, colorReset, colorGray, err.Error(), colorReset,
			colorRed, colorReset,
		)
		os.Exit(1)
	}
}
