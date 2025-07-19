package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// init initializes the version command and adds it to the root command.
func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of the SwitchTube downloader",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println("TODO: Dynamically add correct version")
	},
}
