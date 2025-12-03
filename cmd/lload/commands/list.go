package commands

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"lloader/internal/app"
	"lloader/internal/models"
)

func NewListCommand(cfg *app.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available models",
		Long:  "List all available llama.cpp models in the configured models directory",
		Run: func(cmd *cobra.Command, args []string) {
			logger, err := app.SetupLogger(cfg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to setup logger: %v\n", err)
				os.Exit(1)
			}
			defer logger.Sync()

			modelList, err := models.DiscoverModels(cfg, logger)
			if err != nil {
				logger.Error("Failed to discover models", zap.Error(err))
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			if len(modelList) == 0 {
				fmt.Println("No models found.")
				return
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tSIZE\tPATH")
			for _, model := range modelList {
				sizeMB := float64(model.Size) / (1024 * 1024)
				fmt.Fprintf(w, "%s\t%.2f MB\t%s\n", model.Name, sizeMB, model.Path)
			}
			w.Flush()
		},
	}
}
