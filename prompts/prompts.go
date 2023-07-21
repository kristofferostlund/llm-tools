package prompts

import (
	"context"
	"fmt"
	"os"

	relative "github.com/kristofferostlund/openai-chatgpt-playground/pkg"
)

const GolangInterfaceMockGenerator = "golang-interface-mock-generation.md"

func Load(ctx context.Context, prompts ...string) (map[string]string, error) {
	loaded := make(map[string]string, len(prompts))

	for _, prompt := range prompts {
		content, err := Get(ctx, prompt)
		if err != nil {
			return nil, fmt.Errorf("getting prompt: %w", err)
		}

		loaded[prompt] = content
	}

	return loaded, nil
}

func Get(ctx context.Context, prompt string) (string, error) {
	// _, filename, _, _ := runtime.Caller(0)
	// filepath := path.Join(path.Dir(filename), "assets", prompt)
	filepath := relative.Filepath("./assets/golang-interface-mock-generation.md")

	b, err := os.ReadFile(filepath)
	if err != nil {
		return "", fmt.Errorf("reading file: %w", err)
	}

	return string(b), nil
}
