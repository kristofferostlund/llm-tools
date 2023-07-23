package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"

	"github.com/kristofferostlund/llm-tools/prompts"
	"github.com/sashabaranov/go-openai"
	"golang.org/x/sync/errgroup"
)

var (
	packageName   = flag.String("package", "", "the package name to generate mocks for")
	interfaceFile = flag.String("interface-file", "", "filepath of file to generate mocks for")
	outputFolder  = flag.String("output-folder", "", "folder to output generated mocks to")

	model = flag.String("model", "gpt-3.5-turbo", "the GhatGPT model to use for prompts")

	debug = flag.Bool("debug", false, "enable debug mode")
)

func debugf(format string, args ...interface{}) {
	if *debug {
		log.Printf("DEBUG: "+format, args...)
	}
}

type Config struct {
	model string

	packageName   string
	interfaceFile string

	outputWriter func(ctx context.Context, parsed ParsedInterface) error
}

type ParsedInterface struct {
	Filepath string
	Content  string
}

func main() {
	flag.Parse()

	if *packageName == "" || *interfaceFile == "" {
		flag.Usage()
		return
	}

	cfg := Config{
		model:         *model,
		packageName:   *packageName,
		interfaceFile: *interfaceFile,
		outputWriter:  getFileOutputWriter(*outputFolder),
	}
	if *outputFolder == "" {
		cfg.outputWriter = getBasicWriter(os.Stdout)
		log.Printf("WARN: Will output to stdout")
	}

	ctx := context.Background()

	openaiAPIKey, ok := os.LookupEnv("OPENAI_API_KEY")
	if !ok {
		log.Fatal("OPENAI_API_KEY must be set")
	}
	client := openai.NewClient(openaiAPIKey)

	// It gets creative with the output format when the temperature is higher.
	if err := runGenerator(ctx, client, cfg); err != nil {
		log.Fatalf("running generator")
	}
}

func runGenerator(ctx context.Context, client *openai.Client, cfg Config) error {
	interfaceFileContent, err := os.ReadFile(cfg.interfaceFile)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	generationSystemPrompt, err := prompts.Get(ctx, prompts.GolangInterfaceMockGenerator)
	if err != nil {
		return fmt.Errorf("loading prompts: %w", err)
	}

	// res, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
	// 	Model: cfg.model,
	// 	Messages: []openai.ChatCompletionMessage{
	// 		{
	// 			Role:    openai.ChatMessageRoleSystem,
	// 			Content: generationSystemPrompt,
	// 		},
	// 		{
	// 			Role:    openai.ChatMessageRoleUser,
	// 			Content: cfg.packageName,
	// 		},
	// 		{
	// 			Role:    openai.ChatMessageRoleUser,
	// 			Content: string(interfaceFileContent),
	// 		},
	// 	},
	// 	Temperature: 0,
	// })
	// if err != nil {
	// 	return fmt.Errorf("generating completion: %w", err)
	// }

	stream, err := client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model: cfg.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: generationSystemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: cfg.packageName,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: string(interfaceFileContent),
			},
		},
		Stream:      true,
		Temperature: 0,
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

func parseResult(ctx context.Context, reader io.Reader, writerFunc func(ctx context.Context, parsed ParsedInterface) error) error {
	// Template:
	// filepath: `name of the file`
	// ```go
	// mock implementation
	// ```

	var parsed []ParsedInterface
	insideBlock := false

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		debugf("parsing line: %s", line)

		i := len(parsed) - 1
		switch {
		case strings.HasPrefix(line, "filepath"):
			// New file
			filename := strings.TrimPrefix(line, "filepath: ")
			filename = strings.Trim(filename, "`")

			parsed = append(parsed, ParsedInterface{
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

func getBasicWriter(w io.Writer) func(ctx context.Context, parsed ParsedInterface) error {
	return func(ctx context.Context, parsed ParsedInterface) error {
		debugf("writing to stdout for filepath: %s", parsed.Filepath)

		if _, err := fmt.Fprintf(w, "filepath: %s\n```go\n%s\n```", parsed.Filepath, parsed.Content); err != nil {
			return fmt.Errorf("writing: %w", err)
		}
		return nil
	}
}

func getFileOutputWriter(folder string) func(ctx context.Context, parsed ParsedInterface) error {
	return func(ctx context.Context, parsed ParsedInterface) error {
		debugf("writing to file: %s", parsed.Filepath)

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
