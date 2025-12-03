package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

const Version = "0.1.0"

func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("lloader version %s\n", Version)
		},
	}
}
