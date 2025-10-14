package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/cterence/rangemusique/internal/app"
	altsrc "github.com/urfave/cli-altsrc/v3"
	"github.com/urfave/cli-altsrc/v3/yaml"
	"github.com/urfave/cli/v3"
)

func setLogger(logLevel string) error {
	var slogLogLevel slog.Level

	switch logLevel {
	case "debug":
		slogLogLevel = slog.LevelDebug
	case "info":
		slogLogLevel = slog.LevelInfo
	case "warn":
		slogLogLevel = slog.LevelWarn
	case "error":
		slogLogLevel = slog.LevelError
	default:
		return fmt.Errorf("unknown log level: %s", logLevel)
	}
	logOpts := slog.HandlerOptions{
		Level: slogLogLevel,
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &logOpts)))

	return nil
}

func main() {
	var (
		version = "dev"
		commit  = "unknown"
		date    = "unknown"
	)

	var (
		configPath   string
		inputDir     string
		outputDir    string
		discogsToken string
		copy         bool
		logLevel     string
	)

	cmd := &cli.Command{
		Name:    "rangemusique",
		Usage:   "Arrange music files based on metadata",
		Version: fmt.Sprintf("Version: %s\nCommit: %s\nBuild Date: %s", version, commit, date),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Aliases:     []string{"c"},
				Value:       "config.yaml",
				Usage:       "path to the configuration file",
				Sources:     cli.NewValueSourceChain(cli.EnvVar("CONFIG_PATH")),
				Destination: &configPath,
			},
			&cli.StringFlag{
				Name:        "input-dir",
				Aliases:     []string{"i"},
				Required:    true,
				Usage:       "path to the input directory containing music files",
				Sources:     cli.NewValueSourceChain(cli.EnvVar("INPUT_DIR"), yaml.YAML("inputDir", altsrc.NewStringPtrSourcer(&configPath))),
				Destination: &inputDir,
			},
			&cli.StringFlag{
				Name:        "output-dir",
				Aliases:     []string{"o"},
				Required:    true,
				Usage:       "path to the output directory where arranged files will be saved",
				Sources:     cli.NewValueSourceChain(cli.EnvVar("OUTPUT_DIR"), yaml.YAML("outputDir", altsrc.NewStringPtrSourcer(&configPath))),
				Destination: &outputDir,
			},
			&cli.StringFlag{
				Name:        "discogs-token",
				Usage:       "token for Discogs API",
				Sources:     cli.NewValueSourceChain(cli.EnvVar("DISCOGS_TOKEN"), yaml.YAML("discogs.token", altsrc.NewStringPtrSourcer(&configPath))),
				Destination: &discogsToken,
			},
			&cli.StringFlag{
				Name:        "log-level",
				Usage:       "log level",
				Value:       "info",
				Sources:     cli.NewValueSourceChain(cli.EnvVar("LOG_LEVEL"), yaml.YAML("logLevel", altsrc.NewStringPtrSourcer(&configPath))),
				Destination: &logLevel,
			},
			&cli.BoolFlag{
				Name:        "copy",
				Usage:       "copy instead of moving files",
				Sources:     cli.NewValueSourceChain(cli.EnvVar("COPY"), yaml.YAML("copy", altsrc.NewStringPtrSourcer(&configPath))),
				Destination: &copy,
			},
		},
		Action: func(context.Context, *cli.Command) error {
			ctx := context.Background()
			err := setLogger(logLevel)
			if err != nil {
				return fmt.Errorf("failed to set logger: %w", err)
			}
			cfg := app.Config{
				InputDir:     inputDir,
				OutputDir:    outputDir,
				DiscogsToken: discogsToken,
				Copy:         copy,
			}
			return app.Run(ctx, cfg)
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
