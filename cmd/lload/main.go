package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"lloader/cmd/lload/commands"
	"lloader/internal/app"
	"lloader/internal/models"
	"lloader/internal/ui"
)

func main() {
	cfg, err := app.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	rootCmd := &cobra.Command{
		Use:   "lload",
		Short: "TUI frontend for llama.cpp",
		Long: `A terminal user interface for managing and running llama.cpp models.

Lloader provides an interactive interface to select and run llama.cpp models
in either server mode or CLI mode.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := runTUI(cfg); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	rootCmd.PersistentFlags().StringP("config", "c", "", "config file (default is $HOME/.config/lloader/config.yaml)")
	rootCmd.PersistentFlags().StringP("models-dir", "m", "", "models directory (overrides config)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")

	rootCmd.AddCommand(
		commands.NewListCommand(cfg),
		commands.NewConfigCommand(cfg),
		commands.NewVersionCommand(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runTUI(cfg *app.Config) error {
	logger, err := app.SetupLogger(cfg)
	if err != nil {
		return fmt.Errorf("failed to setup logger: %w", err)
	}
	defer logger.Sync()

	modelList, err := models.DiscoverModels(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to discover models: %w", err)
	}

	if len(modelList) == 0 {
		return fmt.Errorf("no models found in %s", cfg.ModelsDir)
	}

	modelNames := models.GetModelNames(modelList)
	program := ui.NewProgram(modelNames, cfg, logger)
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}
