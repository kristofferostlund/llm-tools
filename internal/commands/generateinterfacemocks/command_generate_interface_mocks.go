package generateinterfacemocks

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/sashabaranov/go-openai"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
	"golang.org/x/sync/errgroup"

	"github.com/kristofferostlund/llm-tools/internal/config"
	"github.com/kristofferostlund/llm-tools/pkg/relative"
)

const (
	mainPrompt       = "./prompts/generate-go-mock-implementations.md"
	singleShotInput  = "./prompts/single-shot.example-input.md"
	singleShotOutput = "./prompts/single-shot.example-output.md"
)

type interfaceFile struct {
	Filepath string
	Content  string
}

func Command() *cli.Command {
	var (
		packageName   string
		interfaceFile string
		outputFolder  string
		model         string
		openaiAPIKey  string
	)

	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "package",
			Required:    true,
			Usage:       "the package name to generate mocks for",
			Destination: &packageName,
		},
		&cli.StringFlag{
			Name:        "interface-file",
			Required:    true,
			Usage:       "filepath of file to generate mocks for",
			Destination: &interfaceFile,
		},
		&cli.StringFlag{
			Name:        "output-folder",
			Required:    false,
			Usage:       "folder to output generated mocks to",
			Destination: &outputFolder,
		},
		&cli.StringFlag{
			Name:        "model",
			Required:    false,
			Value:       "gpt-3.5-turbo",
			Usage:       "GhatGPT model to use for prompts. gpt-3.5-turbo seems more than enough.",
			Destination: &model,
		},
		altsrc.NewStringFlag(
			&cli.StringFlag{
				Name:        "openai-api-key",
				Required:    false, // It kind of is, but altsrc seems to not work with required flags...
				Usage:       "OpenAI API key, will read from environment variables or from config file",
				EnvVars:     []string{"OPENAI_API_KEY"},
				Destination: &openaiAPIKey,
			},
		),
	}

	cmd := &cli.Command{
		Name:   "generate-interface-mocks",
		Usage:  "Generates mock implementations for Go interfaces",
		Flags:  flags,
		Before: config.AltSrcInputSource(flags),
		Action: func(c *cli.Context) error {
			ctx := c.Context

			outputWriter := folderOutputWriter(outputFolder)
			if outputFolder == "" {
				outputWriter = stdoutWriter(os.Stdout)
				slog.InfoContext(ctx, "output-folder not set, writing to stdout")
			}

			cfg := generateConfig{
				model:         model,
				packageName:   packageName,
				interfaceFile: interfaceFile,
				outputWriter:  outputWriter,
				openaiAPIKey:  openaiAPIKey,
			}

			if err := cfg.validate(ctx); err != nil {
				return fmt.Errorf("validating config: %w", err)
			}

			client := openai.NewClient(cfg.openaiAPIKey)

			if err := generate(ctx, client, cfg); err != nil {
				log.Fatalf("running generator: %v", err)
			}

			return nil
		},
	}

	return cmd
}

func generate(ctx context.Context, client *openai.Client, cfg generateConfig) error {
	interfaceFileContent, err := os.ReadFile(cfg.interfaceFile)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	prompts, err := relative.Files(ctx, mainPrompt, singleShotInput, singleShotOutput)
	if err != nil {
		return fmt.Errorf("loading prompt: %w", err)
	}

	stream, err := client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model: cfg.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: prompts[mainPrompt],
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompts[singleShotInput],
			},
			{
				Role:    openai.ChatMessageRoleAssistant,
				Content: prompts[singleShotOutput],
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: fmt.Sprintf("%s\n\n%s", cfg.packageName, string(interfaceFileContent)),
			},
		},
		Stream: true,
		// It gets creative with the output format when the temperature is higher.
		Temperature: 0, // No creativity seems to be good.
	})
	if err != nil {
		return fmt.Errorf("generating completion: %w", err)
	}
	defer stream.Close()

	reader, writer := io.Pipe()
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		if err := parseResult(ctx, reader, cfg.outputWriter); err != nil {
			return fmt.Errorf("parsing result: %w", err)
		}
		return nil
	})

	eg.Go(func() error {
		defer reader.Close()
		for {
			resp, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("receiving completion: %w", err)
			}

			if _, err := writer.Write([]byte(resp.Choices[0].Delta.Content)); err != nil {
				return fmt.Errorf("writing: %w", err)
			}
		}
	})

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("streaming result: %w", err)
	}

	return nil
}

type generateConfig struct {
	model        string
	openaiAPIKey string

	packageName   string
	interfaceFile string

	outputWriter func(ctx context.Context, parsed interfaceFile) error
}

func (c generateConfig) validate(ctx context.Context) error {
	var errs *multierror.Error
	if c.packageName == "" {
		errs = multierror.Append(errs, errors.New("package name is required"))
	}
	if c.interfaceFile == "" {
		errs = multierror.Append(errs, errors.New("interface file is required"))
	}
	if c.outputWriter == nil {
		errs = multierror.Append(errs, errors.New("output writer is required"))
	}
	if c.model == "" {
		errs = multierror.Append(errs, errors.New("model is required"))
	}
	if c.openaiAPIKey == "" {
		errs = multierror.Append(errs, errors.New("openai api key is required"))
	}

	if err := errs.ErrorOrNil(); err != nil {
		return fmt.Errorf("validating config: %w", err)
	}
	return nil
}

func parseResult(ctx context.Context, reader io.Reader, writerFunc func(ctx context.Context, parsed interfaceFile) error) error {
	// Template:
	// filepath: `name of the file`
	// ```go
	// mock implementation
	// ```

	var parsed []interfaceFile
	insideBlock := false

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		slog.DebugContext(ctx, "parsing line", slog.String("line", line))

		i := len(parsed) - 1
		switch {
		case strings.HasPrefix(line, "filepath"):
			// New file
			filename := strings.TrimPrefix(line, "filepath: ")
			filename = strings.Trim(filename, "`")

			parsed = append(parsed, interfaceFile{
				Filepath: filename,
				Content:  "",
			})
		case strings.HasPrefix(line, "```go"):
			insideBlock = true
		case strings.HasPrefix(line, "```"):
			insideBlock = false
			if i < 0 {
				return fmt.Errorf("unexpected end of block")
			}

			if err := writerFunc(ctx, parsed[i]); err != nil {
				return fmt.Errorf("writing: %w", err)
			}
		case insideBlock:
			if i < 0 {
				return fmt.Errorf("unexpected line inside block")
			}
			parsed[i].Content += line + "\n"
		case !insideBlock:
			// ignore line
		}
	}

	return nil
}

func stdoutWriter(w io.Writer) func(ctx context.Context, parsed interfaceFile) error {
	return func(ctx context.Context, parsed interfaceFile) error {
		slog.DebugContext(ctx, "writing to stdout", slog.String("filepath", parsed.Filepath))

		if _, err := fmt.Fprintf(w, "filepath: %s\n```go\n%s\n```", parsed.Filepath, parsed.Content); err != nil {
			return fmt.Errorf("writing: %w", err)
		}
		return nil
	}
}

func folderOutputWriter(folder string) func(ctx context.Context, parsed interfaceFile) error {
	return func(ctx context.Context, parsed interfaceFile) error {
		slog.DebugContext(ctx, "writing to file", slog.String("filepath", parsed.Filepath))

		fullPath := path.Join(folder, parsed.Filepath)
		dir := path.Dir(fullPath)

		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return fmt.Errorf("creating dir (%s): %w", dir, err)
		}

		f, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY, os.ModePerm)
		if err != nil {
			return fmt.Errorf("opening file (%s): %w", fullPath, err)
		}
		defer f.Close()

		if _, err := f.WriteString(parsed.Content); err != nil {
			return fmt.Errorf("writing file (%s): %w", fullPath, err)
		}

		return nil
	}
}
