package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/cterence/rangemusique/internal/app"
	altsrc "github.com/urfave/cli-altsrc/v3"
	"github.com/urfave/cli-altsrc/v3/yaml"
	"github.com/urfave/cli/v3"
)

func main() {
	var (
		configPath   string
		inputDir     string
		outputDir    string
		discogsToken string
		copy         bool
	)

	cmd := &cli.Command{
		Name:  "rangemusique",
		Usage: "Arrange music files based on metadata and API data",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Aliases:     []string{"c"},
				Value:       "config.yaml",
				Usage:       "Path to the configuration file",
				Sources:     cli.NewValueSourceChain(cli.EnvVar("CONFIG_PATH")),
				Destination: &configPath,
			},
			&cli.StringFlag{
				Name:        "input-dir",
				Aliases:     []string{"i"},
				Usage:       "Path to the input directory containing music files",
				Sources:     cli.NewValueSourceChain(cli.EnvVar("INPUT_DIR"), yaml.YAML("inputDir", altsrc.NewStringPtrSourcer(&configPath))),
				Destination: &inputDir,
			},
			&cli.StringFlag{
				Name:        "output-dir",
				Aliases:     []string{"o"},
				Usage:       "Path to the output directory where arranged files will be saved",
				Sources:     cli.NewValueSourceChain(cli.EnvVar("OUTPUT_DIR"), yaml.YAML("outputDir", altsrc.NewStringPtrSourcer(&configPath))),
				Destination: &outputDir,
			},
			&cli.StringFlag{
				Name:        "discogs-token",
				Usage:       "Token for Discogs API",
				Sources:     cli.NewValueSourceChain(cli.EnvVar("DISCOGS_TOKEN"), yaml.YAML("discogs.token", altsrc.NewStringPtrSourcer(&configPath))),
				Destination: &discogsToken,
			},
			&cli.BoolFlag{
				Name:        "copy",
				Usage:       "Copy instead of moving files",
				Sources:     cli.NewValueSourceChain(cli.EnvVar("COPY"), yaml.YAML("copy", altsrc.NewStringPtrSourcer(&configPath))),
				Destination: &copy,
			},
		},
		Action: func(context.Context, *cli.Command) error {
			ctx := context.Background()
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
