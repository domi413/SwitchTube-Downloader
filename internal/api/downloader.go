// Package api provides constants and functions for interacting with the
// SWITCHtube API
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"switch-tube-downloader/internal/auth"
)

const baseURL = "https://tube.switch.ch"

// Video represents a video from the API
type Video struct {
	ID    string  `json:"id"`
	Title *string `json:"title"`
}

// String returns a string representation of the video
func (v Video) String() string {
	if v.Title != nil {
		return *v.Title
	}
	return v.ID
}

// Channel represents a channel from the API
type Channel struct {
	ID    string  `json:"id"`
	Title *string `json:"title"`
}

// String returns a string representation of the channel
func (c Channel) String() string {
	if c.Title != nil {
		return *c.Title
	}
	return c.ID
}

// createHTTPClient creates an HTTP client with authentication
func createHTTPClient(token string) *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

// makeAuthenticatedRequest makes an HTTP request with authentication
func makeAuthenticatedRequest(client *http.Client, url, token string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Token "+token)
	return client.Do(req)
}

// extractID extracts the ID from a URL or returns the input if it's already an ID
func extractID(input string) string {
	input = strings.TrimSpace(input)
	if !strings.HasPrefix(input, baseURL+"/") {
		return input
	}
	parts := strings.Split(input, "/")
	if len(parts) >= 2 {
		id := parts[len(parts)-1]
		if id == "" && len(parts) >= 3 {
			id = parts[len(parts)-2]
		}
		return id
	}
	return input
}

// getVideoDownloadInfo gets the download path and extension for a video
func getVideoDownloadInfo(client *http.Client, videoID, token string) (string, string, error) {
	url := fmt.Sprintf("%s/api/v1/browse/videos/%s/video_variants", baseURL, videoID)
	resp, err := makeAuthenticatedRequest(client, url, token)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var variants []struct {
		Path      string `json:"path"`
		MediaType string `json:"media_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&variants); err != nil {
		return "", "", err
	}

	if len(variants) == 0 {
		return "", "", fmt.Errorf("no variants found")
	}

	parts := strings.Split(variants[0].MediaType, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid media type: %s", variants[0].MediaType)
	}

	return variants[0].Path, parts[1], nil
}

// downloadFileWithProgress downloads a file with progress bar
func downloadFileWithProgress(client *http.Client, path, filename, token string, currentVideo, totalVideos int) error {
	resp, err := makeAuthenticatedRequest(client, baseURL+path, token)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()

	totalSize := resp.ContentLength
	var written int64
	barLength := 50
	startTime := time.Now()

	for {
		buffer := make([]byte, 32*1024)
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			if _, err := out.Write(buffer[:n]); err != nil {
				return err
			}
			written += int64(n)
			percent := float64(written) / float64(totalSize) * 100
			bar := strings.Repeat("#", int(float64(barLength)*percent/100)) + strings.Repeat("-", int(barLength)-int(float64(barLength)*percent/100))

			// Calculate real-time speed
			elapsed := time.Since(startTime).Seconds()
			speed := float64(written) / elapsed / (1024 * 1024 / 8) // Mb/s

			fmt.Printf("\r\x1b[2K[%d/%d] Downloading %s %3.0f%% [%s] (%.2f MB/s)",
				currentVideo, totalVideos, filename, percent, bar, speed)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// downloadSingleVideo downloads one video
func downloadSingleVideo(client *http.Client, video Video, token string, currentVideo, totalVideos int) error {
	path, extension, err := getVideoDownloadInfo(client, video.ID, token)
	if err != nil {
		return fmt.Errorf("failed to get download info: %w", err)
	}

	filename := fmt.Sprintf("%s.%s", video.String(), extension)
	filename = strings.ReplaceAll(filename, "/", " - ")

	if err := downloadFileWithProgress(client, path, filename, token, currentVideo, totalVideos); err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}

	return nil
}

// getChannelInfo gets channel information including name
func getChannelInfo(client *http.Client, channelID, token string) (*Channel, error) {
	url := fmt.Sprintf("%s/api/v1/browse/channels/%s", baseURL, channelID)
	resp, err := makeAuthenticatedRequest(client, url, token)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var channel Channel
	if err := json.NewDecoder(resp.Body).Decode(&channel); err != nil {
		return nil, err
	}

	return &channel, nil
}

// DownloadVideo downloads a single video by ID or URL
func DownloadVideo(videoInput string, name string) {
	videoID := extractID(videoInput)
	token := auth.GetToken()
	if token == "" {
		fmt.Println("No token found. Please create a token first.")
		return
	}

	client := createHTTPClient(token)

	url := fmt.Sprintf("%s/api/v1/browse/videos/%s", baseURL, videoID)
	resp, err := makeAuthenticatedRequest(client, url, token)
	if err != nil {
		fmt.Printf("Error getting video info: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error getting video info: HTTP %d\n", resp.StatusCode)
		return
	}

	var video Video
	if err := json.NewDecoder(resp.Body).Decode(&video); err != nil {
		fmt.Printf("Error decoding video info: %v\n", err)
		return
	}

	if name != "" {
		video.Title = &name
	}

	if err := downloadSingleVideo(client, video, token, 1, 1); err != nil {
		fmt.Printf("Error downloading video: %v\n", err)
	}
}

// DownloadChannel downloads all videos from a channel by ID or URL
func DownloadChannel(channelInput string, name string) {
	channelID := extractID(channelInput)
	token := auth.GetToken()
	if token == "" {
		fmt.Println("No token found. Please create a token first.")
		return
	}

	client := createHTTPClient(token)

	url := fmt.Sprintf("%s/api/v1/browse/channels/%s/videos", baseURL, channelID)
	resp, err := makeAuthenticatedRequest(client, url, token)
	if err != nil {
		fmt.Printf("Error getting channel videos: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error getting channel videos: HTTP %d\n", resp.StatusCode)
		return
	}

	var videos []Video
	if err := json.NewDecoder(resp.Body).Decode(&videos); err != nil {
		fmt.Printf("Error decoding channel videos: %v\n", err)
		return
	}

	fmt.Printf("Found %d videos in channel\n", len(videos))

	var folderName string
	if name != "" {
		folderName = name
	} else {
		channelInfo, err := getChannelInfo(client, channelID, token)
		if err != nil {
			folderName = fmt.Sprintf("channel_%s", channelID)
		} else {
			folderName = channelInfo.String()
			folderName = strings.ReplaceAll(folderName, "/", " - ")
		}
	}

	if err := os.MkdirAll(folderName, 0o755); err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		return
	}
	originalDir, _ := os.Getwd()
	os.Chdir(folderName)
	defer os.Chdir(originalDir)

	fmt.Printf("Downloading to folder: %s\n", folderName)

	var failed []Video
	for i, video := range videos {
		if err := downloadSingleVideo(client, video, token, i+1, len(videos)); err != nil {
			fmt.Printf("Failed: %v\n", err)
			failed = append(failed, video)
		}
	}

	fmt.Println("Download complete")
	if len(failed) > 0 {
		fmt.Printf("\n%d videos failed:\n", len(failed))
		for _, video := range failed {
			fmt.Printf("- %s\n", video.String())
		}
	}
}
