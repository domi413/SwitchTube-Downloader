// Package media handles the downloading of videos and channels from SwitchTube.
package media

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"switch-tube-downloader/internal/token"
)

// ExtractIDAndType extracts ID and determines if it's a video or channel.
func ExtractIDAndType(input string) (string, string) {
	input = strings.TrimSpace(input)

	// If it's a URL, parse it
	if strings.HasPrefix(input, BaseURL) {
		if strings.Contains(input, "/videos/") {
			parts := strings.Split(input, "/videos/")
			if len(parts) > 1 {
				return strings.Split(parts[1], "/")[0], "video"
			}
		}
		if strings.Contains(input, "/channels/") {
			parts := strings.Split(input, "/channels/")
			if len(parts) > 1 {
				return strings.Split(parts[1], "/")[0], "channel"
			}
		}
	}

	// If it's just an ID, we need to determine type by trying video first
	return input, "unknown"
}

// Download downloads a video or channel based on the input.
func Download(input, customName string) error {
	id, downloadType := ExtractIDAndType(input)

	tokenStr, err := token.Get()
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	// Try as video first if type is unknown
	if downloadType == "unknown" {
		if err := DownloadVideo(id, tokenStr, 1, 1); err == nil {
			fmt.Printf("\n✅ Successfully downloaded video: %s\n", id)
			return nil
		}
		// If video fails, try as channel
		downloadType = "channel"
	}

	switch downloadType {
	case "video":
		if err := DownloadVideo(id, tokenStr, 1, 1); err != nil {
			return fmt.Errorf("failed to download video: %w", err)
		}
		fmt.Printf("\n✅ Successfully downloaded video\n")

	case "channel":
		if err := downloadChannel(id, tokenStr, customName); err != nil {
			return fmt.Errorf("failed to download channel: %w", err)
		}

	default:
		return fmt.Errorf("could not determine if input is video or channel")
	}

	return nil
}

// downloadChannel downloads all videos from a channel.
func downloadChannel(channelID, token, customName string) error {
	// Get channel videos
	resp, err := MakeRequest(fmt.Sprintf("%s/api/v1/browse/channels/%s/videos", BaseURL, channelID), token)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var videos []Video
	if err := json.NewDecoder(resp.Body).Decode(&videos); err != nil {
		return err
	}

	fmt.Printf("📹 Found %d videos in channel\n", len(videos))

	// Create folder
	folderName := customName
	if folderName == "" {
		folderName = fmt.Sprintf("channel_%s", channelID)
	}
	folderName = strings.ReplaceAll(folderName, "/", " - ")

	if err := os.MkdirAll(folderName, 0o755); err != nil {
		return err
	}

	originalDir, _ := os.Getwd()
	os.Chdir(folderName)
	defer os.Chdir(originalDir)

	fmt.Printf("📁 Downloading to folder: %s\n", folderName)

	// Download each video
	var failed []string
	for i, video := range videos {
		if err := DownloadVideo(video.ID, token, i+1, len(videos)); err != nil {
			fmt.Printf("\n❌ Failed: %s\n", video.String())
			failed = append(failed, video.String())
		}
	}

	fmt.Printf("\n✅ Download complete! %d/%d videos\n", len(videos)-len(failed), len(videos))
	return nil
}
