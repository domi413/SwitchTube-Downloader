// Package cli provides the CLI interface for the application
package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"switch-tube-downloader/internal/api"
	"switch-tube-downloader/internal/auth"
)

func showHelp() {
	fmt.Println("Usage: switch-tube-downloader [options]")
	fmt.Println("Options:")
	fmt.Println("  -v, --video <id|url> [<name>]     Download video with id and optionally rename it")
	fmt.Println("  -c, --channel <id|url> [<name>]   Download channel with id/url and optionally rename it")
	fmt.Println("  -t, --token                       Create and save token")
	fmt.Println("  -h, --help                        Show help")
	fmt.Println("  --version                         Show version")
}

func parseArgs(args []string) {
	switch args[0] {
	case "-v", "--video":
		if len(args) < 2 || len(args) > 3 {
			showHelp()
			return
		}

		name := ""
		if len(args) == 3 {
			name = args[2]
		}
		handleVideoDownload(args[1], name)
	case "-c", "--channel":
		if len(args) < 2 || len(args) > 3 {
			showHelp()
			return
		}

		name := ""
		if len(args) == 3 {
			name = args[2]
		}
		handleChannelDownload(args[1], name)
	case "-t", "--token":
		handleTokenCreation()
	case "--version":
		fmt.Println("VERSION XY")
	case "-h", "--help":
		fallthrough
	default:
		showHelp()
	}
}

func handleVideoDownload(videoID string, name string) {
	token := auth.GetToken()
	if token == "" {
		auth.CreateToken()
	}

	api.DownloadVideo(videoID, name)
}

func handleChannelDownload(channelID string, name string) {
	token := auth.GetToken()
	if token == "" {
		auth.CreateToken()
	}

	api.DownloadChannel(channelID, name)
}

func handleTokenCreation() {
	if auth.GetToken() != "" {
		fmt.Println("Token already exists, do you want to overwrite it? (y/n)")
		reader := bufio.NewReader(os.Stdin)

		answer, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			return
		}

		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			return
		}
	}

	if _, err := auth.CreateToken(); err != nil {
		fmt.Printf("Error creating token: %v\n", err)
		return
	}
}

func Run() {
	args := os.Args[1:]
	if len(args) == 0 {
		showHelp()
		return
	}
	parseArgs(args)
}
