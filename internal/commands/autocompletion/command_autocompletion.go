package autocompletion

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/kristofferostlund/llm-tools/internal/config"
	"github.com/urfave/cli/v2"
)

var AutoCompletionFilepath = path.Join(config.FolderPath, "autocomplete")

//go:embed assets/zsh_autocomplete
var zshAutoCompletion string

//go:embed assets/bash_autocomplete
var bashAutoCompletion string

const (
	shellZSH  = "zsh"
	shellBash = "bash"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "auto-completion",
		Usage: "Generate auto-completion script for your shell",
		Action: func(c *cli.Context) error {
			ctx := c.Context

			alreadyExists, err := exists(ctx, AutoCompletionFilepath)
			if err != nil {
				return fmt.Errorf("checking if file exists: %w", err)
			}

			if !alreadyExists {
				slog.DebugContext(ctx, "no auto-completion file found", slog.String("path", AutoCompletionFilepath))

				var autoCompletionFile string
				shell := guessShell(c.Context)
				switch shell {
				case shellZSH:
					autoCompletionFile = zshAutoCompletion
				case shellBash:
					autoCompletionFile = bashAutoCompletion
				default:
					return fmt.Errorf("unsupported shell: %s", shell)
				}

				if err := writeAutoCompleteFile(ctx, AutoCompletionFilepath, autoCompletionFile); err != nil {
					return fmt.Errorf("writing temp file: %w", err)
				}
			} else {
				slog.DebugContext(ctx, "auto-completion file exists, nothing to do", slog.String("path", AutoCompletionFilepath))
			}

			fmt.Println("To enable auto-completion, add the following to your shell config:")
			fmt.Printf("    PROG=%s source %s\n", c.App.Name, AutoCompletionFilepath)

			return nil
		},
	}
}

func writeAutoCompleteFile(ctx context.Context, filepath, autoCompletionFile string) error {
	slog.DebugContext(ctx, "writing auto-completion file", slog.String("filepath", filepath))

	f, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("autocomplete file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(autoCompletionFile); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}

	return nil
}

func guessShell(ctx context.Context) string {
	rawShell := os.Getenv("SHELL")

	if _, ok := os.LookupEnv("ZSH_VERSION"); ok {
		return shellZSH
	}
	if strings.Contains(rawShell, "zsh") {
		return shellZSH
	}

	if _, ok := os.LookupEnv("BASH_VERSION"); ok {
		return shellBash
	}
	if strings.Contains(rawShell, "bash") {
		return shellBash
	}

	return rawShell
}

func exists(ctx context.Context, filepath string) (bool, error) {
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("getting file info: %w", err)
	}
	return true, nil
}
