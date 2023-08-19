package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/kristofferostlund/llm-tools/internal/commands/autocompletion"
	"github.com/kristofferostlund/llm-tools/internal/commands/configure"
	"github.com/kristofferostlund/llm-tools/internal/commands/generateinterfacemocks"
	"github.com/kristofferostlund/llm-tools/internal/config"
	cli "github.com/urfave/cli/v2"
)

func main() {
	var debug bool

	app := &cli.App{
		Name:                 "llm-tools",
		Usage:                "Various LLM tools",
		EnableBashCompletion: true,

		BashComplete: cli.DefaultAppComplete,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "debug",
				Usage:       "enable debug logging",
				Destination: &debug,
				Value:       false,
				EnvVars:     []string{"DEBUG"},
			},
			&cli.StringFlag{
				Name:   "config",
				Usage:  "config file",
				Value:  config.Filepath,
				Hidden: true,
			},
		},
		Before: func(c *cli.Context) error {
			ctx := c.Context

			initGlobalLogger(debug)

			cfg, err := config.Load(ctx)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			c.Context = config.WithConfig(ctx, cfg)

			for _, cmd := range c.App.Commands {
				cmd.BashComplete = cli.DefaultCompleteWithFlags(cmd)
			}

			return nil
		},
		Commands: []*cli.Command{
			configure.Command(),
			generateinterfacemocks.Command(),
			autocompletion.Command(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("running: %v", err)
	}
}

func initGlobalLogger(debug bool) {
	logLevel := slog.LevelInfo
	if debug {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: false,
		Level:     logLevel,
	})))

	if debug {
		slog.Debug("debug logging enabled")
	}
}
