package cmd

import (
	"fmt"

	media "switch-tube-downloader/internal/download"

	"github.com/spf13/cobra"
)

// init initializes the download command and adds it to the root command with its flags.
func init() {
	rootCmd.AddCommand(downloadCmd)
	downloadCmd.Flags().StringP("name", "n", "", "Custom name for the download")
}

var downloadCmd = &cobra.Command{
	Use:   "download <video_url|channel_url|video_id|channel_id>",
	Short: "Download a video or channel",
	Long:  "Download a video or channel. Automatically detects if input is a video or channel.",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name, err := cmd.Flags().GetString("name")
		if err != nil {
			fmt.Printf("Error getting name flag: %v\n", err)

			return
		}

		err = media.Download(args[0], name)
		if err != nil {
			fmt.Printf("Error: %v\n", err)

			return
		}
	},
}
