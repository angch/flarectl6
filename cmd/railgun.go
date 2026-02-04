package cmd

import (
	"github.com/spf13/cobra"
)

var railgunCmd = &cobra.Command{
	Use:     "railgun",
	Aliases: []string{"r"},
	Short:   "Railgun information",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return ensureClient()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

func init() {
	rootCmd.AddCommand(railgunCmd)
}
