package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/pelletier/go-toml/v2"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

// Config contains reusable configuration for the llm-tools.
// Keys must be set in kebab-case to be able to be used as
// defaults for flags.
type Config struct {
	OpenaiAPIKey string `toml:"openai-api-key,omitempty"`
}

var Filepath = func() string {
	homedir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("getting user home dir: %v", err))
	}
	return path.Join(homedir, ".llm-tools", "config.toml")
}()

var FolderPath = path.Dir(Filepath)

type configKey struct{}

func AltSrcInputSource(flags []cli.Flag) cli.BeforeFunc {
	return altsrc.InitInputSourceWithContext(flags, func(c *cli.Context) (altsrc.InputSourceContext, error) {
		if exists, err := Exists(c.Context); err != nil {
			return nil, fmt.Errorf("checking if config file exists: %w", err)
		} else if !exists {
			// No file, nothing to load.
			return altsrc.NewMapInputSource("", nil), nil
		}
		return altsrc.NewTomlSourceFromFile(Filepath)
	})
}

func WithConfig(ctx context.Context, cfg Config) context.Context {
	return context.WithValue(ctx, configKey{}, cfg)
}

func FromContext(ctx context.Context) Config {
	cfg, ok := ctx.Value(configKey{}).(Config)
	if !ok {
		// Not to worry if there is no config.
		return Config{}
	}

	return cfg
}

func Load(ctx context.Context) (Config, error) {
	logger := slog.With(slog.String("filepath", Filepath))

	if exists, err := Exists(ctx); err != nil {
		return Config{}, fmt.Errorf("checking if config file exists: %w", err)
	} else if !exists {
		logger.DebugContext(ctx, "config file does not exist, nothing to load")
		return Config{}, nil
	}

	logger.DebugContext(ctx, "loading config file")
	f, err := os.Open(Filepath)
	if err != nil {
		return Config{}, fmt.Errorf("opening config file: %w", err)
	}
	defer f.Close()
	var cfg Config
	if err := toml.NewDecoder(f).Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decoding config file: %w", err)
	}

	logger.DebugContext(ctx, "config file loaded")

	return cfg, nil
}

func Exists(ctx context.Context) (bool, error) {
	if _, err := os.Stat(Filepath); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("getting file info: %w", err)
	}
	return true, nil
}

func folderExists(ctx context.Context) (bool, error) {
	if _, err := os.Stat(path.Dir(Filepath)); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("getting file info: %w", err)
	}
	return true, nil
}

func Save(ctx context.Context, cfg Config) error {
	logger := slog.With(slog.String("filepath", Filepath))

	folderDir := path.Dir(Filepath)
	if exists, err := folderExists(ctx); err != nil {
		return fmt.Errorf("checking if config file exists: %w", err)
	} else if !exists {
		logger.DebugContext(ctx, "creating folder", slog.String("folder", folderDir))
		if err := os.MkdirAll(folderDir, 0o700); err != nil {
			return fmt.Errorf("creating config directory: %w", err)
		}
	}

	logger.DebugContext(ctx, "saving config file")
	f, err := os.Create(Filepath)
	if err != nil {
		return fmt.Errorf("creating config file: %w", err)
	}
	defer f.Close()

	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		return fmt.Errorf("encoding config file: %w", err)
	}
	logger.DebugContext(ctx, "config file saved")

	return nil
}
