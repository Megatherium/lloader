package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"lloader/internal/app"
)

func NewConfigCommand(cfg *app.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Show current configuration",
		Long:  "Display the current configuration values from config file, environment variables, and flags",
		Run: func(cmd *cobra.Command, args []string) {
			logger, err := app.SetupLogger(cfg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to setup logger: %v\n", err)
				os.Exit(1)
			}
			defer logger.Sync()

			fmt.Println("Current Configuration:")
			fmt.Println("=====================")
			fmt.Printf("Config File: %s\n", viper.ConfigFileUsed())
			fmt.Printf("Models Directory: %s\n", cfg.ModelsDir)
			fmt.Printf("Default NGL: %d\n", cfg.DefaultNGL)
			fmt.Printf("Log Level: %s\n", cfg.LogLevel)
			fmt.Printf("Log File: %s\n", cfg.LogFile)
			fmt.Printf("Server Template: %s\n", cfg.ServerTemplate)
			fmt.Printf("CLI Template: %s\n", cfg.CLITemplate)
			fmt.Println()
			fmt.Println("Environment Variables:")
			fmt.Println("=====================")
			fmt.Println("PRAMA_MODELS_DIR - Override models directory")
			fmt.Println("PRAMA_LOG_LEVEL - Set log level (debug, info, warn, error)")
			fmt.Println("PRAMA_LOG_FILE - Path to log file (empty for stdout)")
		},
	}
}
