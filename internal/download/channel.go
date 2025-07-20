package download

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"switch-tube-downloader/internal/prompt"
)

// channelInfo represents channel metadata.
type channelInfo struct {
	Name string `json:"name"`
}

var (
	errInvalidRange            = errors.New("invalid range")
	errInvalidNumber           = errors.New("invalid number")
	errInvalidEndNumber        = errors.New("invalid end number")
	errNumberOutOfRange        = errors.New("number out of range")
	errInvalidRangeFormat      = errors.New("invalid range format")
	errInvalidStartNumber      = errors.New("invalid start number")
	errFailedSelectVideos      = errors.New("failed to select videos")
	errFailedCreateFolder      = errors.New("failed to create folder")
	errNoValidSelectionsFound  = errors.New("no valid selections found")
	errFailedChangeDirectory   = errors.New("failed to change directory")
	errFailedGetChannelVideos  = errors.New("failed to get channel videos")
	errFailedDecodeChannelVids = errors.New("failed to decode channel videos")
	errFailedGetChannelInfo    = errors.New("failed to get channel information")
	errFailedDecodeChannelMeta = errors.New("failed to decode channel metadata")
)

// downloadChannel downloads selected videos from a channel.
func downloadChannel(channelID string, token string, useEpisode bool, force bool, all bool) error {
	channelInfo, err := getChannelInfo(channelID, token)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedGetChannelInfo, err)
	}

	videos, err := getChannelVideos(channelID, token)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedGetChannelVideos, err)
	}

	if len(videos) == 0 {
		fmt.Println("No videos found in this channel")

		return nil
	}

	fmt.Printf("Found %d videos in channel\n", len(videos))

	selectedIndices, err := selectVideos(videos, all)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedSelectVideos, err)
	}

	if len(selectedIndices) == 0 {
		fmt.Println("No videos selected for download")

		return nil
	}

	folderName, err := createChannelFolder(channelInfo.Name)
	if err != nil {
		return err
	}

	cleanup, err := changeDirToFolder(folderName)
	if err != nil {
		return err
	}
	defer cleanup()

	fmt.Printf("Downloading to folder: %s\n", folderName)
	downloadSelectedVideos(videos, selectedIndices, token, useEpisode, force)

	return nil
}

// downloadSelectedVideos downloads the selected videos and reports results.
func downloadSelectedVideos(
	videos []video,
	selectedIndices []int,
	token string,
	useEpisode bool,
	force bool,
) {
	var failed []string

	for i, videoIndex := range selectedIndices {
		video := videos[videoIndex]

		err := downloadVideo(video.ID, token, i+1, len(selectedIndices), useEpisode, force)
		if err != nil {
			fmt.Printf("\nFailed: %s - %v\n", video.Title, err)
			failed = append(failed, video.Title)
		}
	}

	fmt.Printf("\nDownload complete! %d/%d videos successful\n",
		len(selectedIndices)-len(failed), len(selectedIndices))

	if len(failed) > 0 {
		fmt.Println("Failed downloads:")

		for _, title := range failed {
			fmt.Printf("  - %s\n", title)
		}
	}
}

// changeDirToFolder changes the directory to the specified folder and returns a
// cleanup function.
func changeDirToFolder(folderName string) (func(), error) {
	originalDir, _ := os.Getwd()

	err := os.Chdir(folderName)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errFailedChangeDirectory, err)
	}

	cleanup := func() {
		err := os.Chdir(originalDir)
		if err != nil {
			fmt.Printf(
				"Warning: failed to change back to original directory %s: %v\n",
				originalDir,
				err,
			)
		}
	}

	return cleanup, nil
}

// createChannelFolder creates a download folder using the channel name.
func createChannelFolder(channelName string) (string, error) {
	folderName := strings.ReplaceAll(channelName, "/", " - ")
	folderName = filepath.Clean(folderName)

	err := os.MkdirAll(folderName, dirPermissions)
	if err != nil {
		return "", fmt.Errorf("%w: %w", errFailedCreateFolder, err)
	}

	return folderName, nil
}

// getChannelVideos retrieves all videos from a channel.
func getChannelVideos(channelID, token string) ([]video, error) {
	endpoint := channelAPI + channelID + "/videos"

	resp, err := makeAuthenticatedRequest(endpoint, token)
	if err != nil {
		return nil, err
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			fmt.Printf("Warning: failed to close response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"%w: status %d: %s",
			errHTTPNotOK,
			resp.StatusCode,
			http.StatusText(resp.StatusCode),
		)
	}

	var videos []video

	err = json.NewDecoder(resp.Body).Decode(&videos)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errFailedDecodeChannelVids, err)
	}

	return videos, nil
}

// selectVideos displays the video list and handles user selection.
func selectVideos(videos []video, all bool) ([]int, error) {
	// If --all flag is used, select all videos
	if all {
		indices := make([]int, len(videos))
		for i := range indices {
			indices[i] = i
		}

		return indices, nil
	}

	// Display video list
	fmt.Println("\nAvailable videos:")

	for i, video := range videos {
		fmt.Printf("%d. %s\n", i+1, video.Title)
	}

	fmt.Println("\nSelect videos to download:")
	fmt.Println("Examples: '1-12', '1,3,5', '1 3 5', or press Enter for all")

	input := prompt.Input("Selection: ")
	input = strings.TrimSpace(input)

	// Empty input means select all
	if input == "" {
		indices := make([]int, len(videos))
		for i := range indices {
			indices[i] = i
		}

		return indices, nil
	}

	return parseSelection(input, len(videos))
}

// parseSelection parses user input and returns selected video indices.
func parseSelection(input string, maxVideos int) ([]int, error) {
	var indices []int

	seen := make(map[int]bool)

	// Split by comma, space, or both
	parts := strings.FieldsFunc(input, func(r rune) bool {
		return r == ',' || r == ' '
	})

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Handle range (e.g., "1-5")
		if strings.Contains(part, "-") {
			var err error

			indices, err = handleRangePart(part, maxVideos, indices, seen)
			if err != nil {
				return nil, err
			}
		} else {
			var err error

			indices, err = handleSinglePart(part, maxVideos, indices, seen)
			if err != nil {
				return nil, err
			}
		}
	}

	if len(indices) == 0 {
		return nil, fmt.Errorf("%w", errNoValidSelectionsFound)
	}

	// Sort indices to maintain order
	sort.Ints(indices)

	return indices, nil
}

// getChannelInfo retrieves channel metadata from the API.
func getChannelInfo(channelID, token string) (*channelInfo, error) {
	resp, err := makeAuthenticatedRequest(channelAPI+channelID, token)
	if err != nil {
		return nil, err
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			fmt.Printf("Warning: failed to close response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"%w: status %d: %s",
			errHTTPNotOK,
			resp.StatusCode,
			http.StatusText(resp.StatusCode),
		)
	}

	var channelData channelInfo

	err = json.NewDecoder(resp.Body).Decode(&channelData)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errFailedDecodeChannelMeta, err)
	}

	return &channelData, nil
}

// handleRangePart processes a range selection like "1-5".
func handleRangePart(part string, maxVideos int, indices []int, seen map[int]bool) ([]int, error) {
	rangeParts := strings.Split(part, "-")
	if len(rangeParts) != rangePartsCount {
		return nil, fmt.Errorf("%w: %s", errInvalidRangeFormat, part)
	}

	start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errInvalidStartNumber, rangeParts[0])
	}

	end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errInvalidEndNumber, rangeParts[1])
	}

	if start < 1 || end > maxVideos || start > end {
		return nil, fmt.Errorf("%w: %d-%d (must be 1-%d)", errInvalidRange, start, end, maxVideos)
	}

	for i := start; i <= end; i++ {
		index := i - 1
		if !seen[index] {
			indices = append(indices, index)
			seen[index] = true
		}
	}

	return indices, nil
}

// handleSinglePart processes a single number selection.
func handleSinglePart(part string, maxVideos int, indices []int, seen map[int]bool) ([]int, error) {
	num, err := strconv.Atoi(part)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errInvalidNumber, part)
	}

	if num < 1 || num > maxVideos {
		return nil, fmt.Errorf("%w: %d (must be 1-%d)", errNumberOutOfRange, num, maxVideos)
	}

	index := num - 1
	if !seen[index] {
		indices = append(indices, index)
		seen[index] = true
	}

	return indices, nil
}
