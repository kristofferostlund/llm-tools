package configure

import (
	"fmt"

	"github.com/kristofferostlund/llm-tools/internal/config"
	"github.com/urfave/cli/v2"
)

func Command() *cli.Command {
	var openaiAPIKey string

	return &cli.Command{
		Name:  "config",
		Usage: "Configure LLM tools",
		Subcommands: []*cli.Command{
			{
				Name: "where",
				Action: func(c *cli.Context) error {
					exists, err := config.Exists(c.Context)
					if err != nil {
						return fmt.Errorf("checking if config exists: %w", err)
					}
					if !exists {
						return fmt.Errorf("no config file")
					}

					fmt.Println(config.Filepath)
					return nil
				},
			},
			{
				Name:  "set",
				Usage: "Set configuration values",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "openai-api-key",
						Usage:       "OpenAI API key",
						Required:    false,
						Destination: &openaiAPIKey,
					},
				},
				Action: func(c *cli.Context) error {
					ctx := c.Context
					cfg, err := config.Load(ctx)
					if err != nil {
						return fmt.Errorf("loading config: %w", err)
					}

					isUpdated := false
					if openaiAPIKey != "" && cfg.OpenaiAPIKey != openaiAPIKey {
						isUpdated = true
						cfg.OpenaiAPIKey = openaiAPIKey
					}

					if isUpdated {
						if err := config.Save(ctx, cfg); err != nil {
							return fmt.Errorf("saving config: %w", err)
						}
					}

					return nil
				},
			},
		},
	}
}
