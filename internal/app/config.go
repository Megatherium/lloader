package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Config struct {
	ModelsDir      string `mapstructure:"models_dir" yaml:"models_dir"`
	DefaultNGL     int    `mapstructure:"default_ngl" yaml:"default_ngl"`
	DefaultCtxSize int    `mapstructure:"default_ctx_size" yaml:"default_ctx_size"`
	LogLevel       string `mapstructure:"log_level" yaml:"log_level"`
	LogFile        string `mapstructure:"log_file" yaml:"log_file"`
	ServerTemplate string `mapstructure:"server_template" yaml:"server_template"`
	CLITemplate    string `mapstructure:"cli_template" yaml:"cli_template"`
}

func DefaultConfig() *Config {
	return &Config{
		ModelsDir:      defaultModelsDir(),
		DefaultNGL:     99,
		DefaultCtxSize: 0, // 0 lets the model choose
		LogLevel:       "info",
		LogFile:        "",
		ServerTemplate: "llama-server -m {model_path} -ngl {ngl} -c {ctx_size}",
		CLITemplate:    "llama-cli -m {model_path} -ngl {ngl} -c {ctx_size}",
	}
}

func defaultModelsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "./models"
	}
	return filepath.Join(home, "models")
}

func LoadConfig() (*Config, error) {
	cfg := DefaultConfig()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.config/lloader")
	viper.AddConfigPath("/etc/lloader")

	viper.SetDefault("models_dir", cfg.ModelsDir)
	viper.SetDefault("default_ngl", cfg.DefaultNGL)
	viper.SetDefault("default_ctx_size", cfg.DefaultCtxSize)
	viper.SetDefault("log_level", cfg.LogLevel)
	viper.SetDefault("log_file", cfg.LogFile)
	viper.SetDefault("server_template", cfg.ServerTemplate)
	viper.SetDefault("cli_template", cfg.CLITemplate)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg, nil
}

func SetupLogger(cfg *Config) (*zap.Logger, error) {
	var logger *zap.Logger
	var err error

	if cfg.LogFile != "" {
		logger, err = zap.NewProduction()
	} else {
		// For TUI applications, we want to log to stderr to avoid interfering with stdout
		config := zap.NewDevelopmentConfig()
		config.OutputPaths = []string{"stderr"}
		logger, err = config.Build()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	return logger, nil
}
