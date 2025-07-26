// Package cmd implements the command-line interface for the
// SwitchTube-Downloader application.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "SwitchTube-Downloader",
	Short: "A CLI downloader for SwitchTube videos",

	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

// Execute runs the root command and handles any errors.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
