package cmd

import (
	"fmt"

	media "switch-tube-downloader/internal/download"

	"github.com/spf13/cobra"
)

// init initializes the download command and adds it to the root command with
// its flags.
func init() {
	rootCmd.AddCommand(downloadCmd)
	downloadCmd.Flags().BoolP("force", "f", false, "Force overwrite if file already exist")
	downloadCmd.Flags().BoolP("all", "a", false, "Download the whole content of a channel")
}

var downloadCmd = &cobra.Command{
	Use:   "download <id>",
	Short: "Download a video or channel",
	Long:  "Download a video or channel. Automatically detects if input is a video or channel.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]

		force, err := cmd.Flags().GetBool("force")
		if err != nil {
			fmt.Printf("Error getting force flag: %v", err)

			return
		}

		all, err := cmd.Flags().GetBool("all")
		if err != nil {
			fmt.Printf("Error getting all flag: %v", err)

			return
		}

		err = media.Download(id, force, all)
		if err != nil {
			fmt.Printf("Error: %v\n", err)

			return
		}
	},
}
