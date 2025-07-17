// Package media handles the downloading of videos and channels from SwitchTube.
package media

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"switch-tube-downloader/internal/prompt"
	"switch-tube-downloader/internal/token"
)

const (
	baseURL             = "https://tube.switch.ch/"
	videoAPI            = "api/v1/browse/videos/"
	channelAPI          = "api/v1/browse/channels/"
	videoPrefix         = "videos/"
	channelPrefix       = "channels/"
	headerAuthorization = "Authorization"

	// File and directory permissions.
	dirPermissions = 0o644

	// Magic numbers.
	minMediaTypeParts = 2
	rangePartsCount   = 2
	bufferSizeKB      = 32
	bufferSize        = bufferSizeKB * 1024
)

type mediaType int

const (
	unknown mediaType = iota
	video
	channel
)

// VideoInfo represents the structure of the video API response.
type VideoInfo struct {
	Title string `json:"title"`
}

// VideoVariant represents a video download variant.
type VideoVariant struct {
	Path      string `json:"path"`
	MediaType string `json:"mediaType"`
}

// ChannelInfo represents channel metadata.
type ChannelInfo struct {
	Name string `json:"name"`
}

// ChannelVideo represents a video in a channel.
type ChannelVideo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

var (
	errInvalidURL              = errors.New("invalid url")
	errFailedToGetToken        = errors.New("failed to get token")
	errFailedToExtractType     = errors.New("failed to extract type")
	errFailedToDownloadVideo   = errors.New("failed to download video")
	errFailedToDownloadChannel = errors.New("failed to download channel")
	errCouldNotDetermineType   = errors.New("could not determine if channel or video")
	errNoVariantsFound         = errors.New("no video variants found")
	errFailedGetVideoInfo      = errors.New("failed to get video information")
	errFailedGetVideoVariants  = errors.New("failed to get video variants")
	errFileCreationAborted     = errors.New("file creation aborted")
	errFailedFetchVideoStream  = errors.New("failed to fetch video stream")
	errFailedCopyVideoData     = errors.New("failed to copy video data")
	errFailedGetChannelInfo    = errors.New("failed to get channel information")
	errFailedGetChannelVideos  = errors.New("failed to get channel videos")
	errInvalidRangeFormat      = errors.New("invalid range format")
	errInvalidStartNumber      = errors.New("invalid start number")
	errInvalidEndNumber        = errors.New("invalid end number")
	errInvalidRange            = errors.New("invalid range")
	errInvalidNumber           = errors.New("invalid number")
	errNumberOutOfRange        = errors.New("number out of range")
	errNoValidSelectionsFound  = errors.New("no valid selections found")
	errFailedConstructURL      = errors.New("failed to construct URL")
	errFailedDecodeVideoMeta   = errors.New("failed to decode video metadata")
	errFailedDecodeVariants    = errors.New("failed to decode variants")
	errFailedSelectVideos      = errors.New("failed to select videos")
	errFailedCreateFolder      = errors.New("failed to create folder")
	errFailedChangeDirectory   = errors.New("failed to change directory")
	errFailedDecodeChannelVids = errors.New("failed to decode channel videos")
	errFailedWriteToFile       = errors.New("failed to write to file")
	errFailedReadData          = errors.New("failed to read data")
	errFailedDecodeChannelMeta = errors.New("failed to decode channel metadata")
	errHTTPNotOK               = errors.New("HTTP request failed with non-OK status")
)

// Download downloads a video or a channel.
func Download(media string, force bool, all bool) error {
	id, downloadType, err := extractIDAndType(media)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToExtractType, err)
	}

	token, err := token.Get()
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToGetToken, err)
	}

	switch downloadType {
	case video:
		err = downloadVideo(id, token, 1, 1, force)
		if err != nil {
			return fmt.Errorf("%w: %w", errFailedToDownloadVideo, err)
		}
	case unknown:
		err = downloadVideo(id, token, 1, 1, force)
		if err == nil {
			return nil
		}

		fallthrough
	case channel:
		err = downloadChannel(id, token, force, all)
		if err != nil {
			return fmt.Errorf("%w: %w", errFailedToDownloadChannel, err)
		}
	default:
		return errCouldNotDetermineType
	}

	return nil
}

// downloadVideo downloads a video.
func downloadVideo(
	videoID string,
	token string,
	currentItem int,
	totalItems int,
	force bool,
) error {
	// Get video metadata
	videoData, err := getVideoMetadata(videoID, token)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedGetVideoInfo, err)
	}

	// Get video variants
	variants, err := getVideoVariants(videoID, token)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedGetVideoVariants, err)
	}

	// Create output file
	filename := createFilename(videoData.Title, variants[0].MediaType)

	file, err := createFile(filename, force)
	if err != nil {
		return err
	}

	defer func() {
		err := file.Close()
		if err != nil {
			fmt.Printf("Warning: failed to close file %s: %v\n", filename, err)
		}
	}()

	// Download the video
	err = downloadProcess(variants[0].Path, token, file, filename, currentItem, totalItems)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToDownloadVideo, err)
	}

	return nil
}

// getVideoMetadata retrieves video metadata from the API.
func getVideoMetadata(videoID, token string) (*VideoInfo, error) {
	resp, err := makeAuthenticatedRequest(videoAPI+videoID, token)
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
		return nil, fmt.Errorf("%w: status %d", errHTTPNotOK, resp.StatusCode)
	}

	var videoData VideoInfo

	err = json.NewDecoder(resp.Body).Decode(&videoData)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errFailedDecodeVideoMeta, err)
	}

	return &videoData, nil
}

// getVideoVariants retrieves available video variants from the API.
func getVideoVariants(videoID, token string) ([]VideoVariant, error) {
	endpoint := videoAPI + videoID + "/video_variants"

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
		return nil, fmt.Errorf("%w: status %d", errHTTPNotOK, resp.StatusCode)
	}

	var variants []VideoVariant

	err = json.NewDecoder(resp.Body).Decode(&variants)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errFailedDecodeVariants, err)
	}

	if len(variants) == 0 {
		return nil, errNoVariantsFound
	}

	return variants, nil
}

// createFilename creates a sanitized filename from video title and media type.
func createFilename(title, mediaType string) string {
	// Extract extension from media type (e.g., "video/mp4" -> "mp4")
	parts := strings.Split(mediaType, "/")

	extension := "mp4" // default fallback
	if len(parts) >= minMediaTypeParts {
		extension = parts[1]
	}

	// Sanitize title for filename
	sanitizedTitle := strings.ReplaceAll(title, " ", "_")
	filename := fmt.Sprintf("%s.%s", sanitizedTitle, extension)

	return filepath.Clean(filename)
}

// downloadChannel downloads selected videos from a channel.
func downloadChannel(channelID string, token string, force bool, all bool) error {
	// Get channel info for folder name
	channelInfo, err := getChannelInfo(channelID, token)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedGetChannelInfo, err)
	}

	// Get channel videos
	videos, err := getChannelVideos(channelID, token)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedGetChannelVideos, err)
	}

	if len(videos) == 0 {
		fmt.Println("No videos found in this channel")

		return nil
	}

	fmt.Printf("Found %d videos in channel\n", len(videos))

	// Select videos to download
	selectedIndices, err := selectVideos(videos, all)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedSelectVideos, err)
	}

	if len(selectedIndices) == 0 {
		fmt.Println("No videos selected for download")

		return nil
	}

	// Create download folder using channel name
	folderName := strings.ReplaceAll(channelInfo.Name, "/", " - ")
	folderName = filepath.Clean(folderName)

	err = os.MkdirAll(folderName, dirPermissions)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedCreateFolder, err)
	}

	originalDir, _ := os.Getwd()

	err = os.Chdir(folderName)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedChangeDirectory, err)
	}

	defer func() {
		err := os.Chdir(originalDir)
		if err != nil {
			fmt.Printf(
				"Warning: failed to change back to original directory %s: %v\n",
				originalDir,
				err,
			)
		}
	}()

	fmt.Printf("Downloading to folder: %s\n", folderName)

	// Download selected videos
	var failed []string

	for i, videoIndex := range selectedIndices {
		video := videos[videoIndex]

		err := downloadVideo(video.ID, token, i+1, len(selectedIndices), force)
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

	return nil
}

// getChannelVideos retrieves all videos from a channel.
func getChannelVideos(channelID, token string) ([]ChannelVideo, error) {
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
		return nil, fmt.Errorf("%w: status %d", errHTTPNotOK, resp.StatusCode)
	}

	var videos []ChannelVideo

	err = json.NewDecoder(resp.Body).Decode(&videos)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errFailedDecodeChannelVids, err)
	}

	return videos, nil
}

// selectVideos displays the video list and handles user selection.
func selectVideos(videos []ChannelVideo, all bool) ([]int, error) {
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

// createFile creates a file with the given name, handling overwrites based on force flag.
func createFile(filename string, force bool) (*os.File, error) {
	_, err := os.Stat(filename)
	if !os.IsNotExist(err) && !force {
		if !prompt.Confirm("File %s already exists. Overwrite?", filename) {
			return nil, fmt.Errorf("%w", errFileCreationAborted)
		}
	}

	file, err := os.Create(filepath.Clean(filename))
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}

	return file, nil
}

// downloadProcess handles the actual file download.
func downloadProcess(
	path, token string,
	out *os.File,
	filename string,
	currentItem, totalItems int,
) error {
	resp, err := makeAuthenticatedRequest(path, token)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedFetchVideoStream, err)
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			fmt.Printf("Warning: failed to close response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", errHTTPNotOK, resp.StatusCode)
	}

	err = copyWithProgress(resp.Body, out, resp.ContentLength, filename, currentItem, totalItems)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedCopyVideoData, err)
	}

	return nil
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
		index := i - 1 // Convert to 0-based index
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

	index := num - 1 // Convert to 0-based index
	if !seen[index] {
		indices = append(indices, index)
		seen[index] = true
	}

	return indices, nil
}

// copyWithProgress copies data from src to dst while showing download progress.
func copyWithProgress(
	src io.Reader,
	dst io.Writer,
	total int64,
	filename string,
	currentItem, totalItems int,
) error {
	buffer := make([]byte, bufferSize)

	var written int64

	startTime := time.Now()

	for {
		n, err := src.Read(buffer)
		if n > 0 {
			_, writeErr := dst.Write(buffer[:n])
			if writeErr != nil {
				return fmt.Errorf("%w: %w", errFailedWriteToFile, writeErr)
			}

			written += int64(n)
			ShowProgress(written, total, filename, currentItem, totalItems, startTime)
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("%w: %w", errFailedReadData, err)
		}
	}

	ShowProgress(written, total, filename, currentItem, totalItems, startTime)

	return nil
}

// getChannelInfo retrieves channel metadata from the API.
func getChannelInfo(channelID, token string) (*ChannelInfo, error) {
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
		return nil, fmt.Errorf("%w: status %d", errHTTPNotOK, resp.StatusCode)
	}

	var channelData ChannelInfo

	err = json.NewDecoder(resp.Body).Decode(&channelData)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errFailedDecodeChannelMeta, err)
	}

	return &channelData, nil
}

// extractIDAndType extracts the id and determines if it's a video or channel.
func extractIDAndType(input string) (string, mediaType, error) {
	input = strings.TrimSpace(input)

	// If input doesn't start with baseURL, return as unknown type
	if !strings.HasPrefix(input, baseURL) {
		return input, unknown, nil
	}

	id := strings.TrimPrefix(input, baseURL)

	switch {
	case strings.HasPrefix(id, videoPrefix):
		return strings.TrimPrefix(id, videoPrefix), video, nil
	case strings.HasPrefix(id, channelPrefix):
		return strings.TrimPrefix(id, channelPrefix), channel, nil
	default:
		return id, unknown, errInvalidURL
	}
}

func makeAuthenticatedRequest(endpoint string, token string) (*http.Response, error) {
	fullURL, err := url.JoinPath(baseURL, endpoint)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errFailedConstructURL, err)
	}

	return makeRequest(fullURL, token)
}

// makeRequest makes an authenticated HTTP request.
func makeRequest(url string, token string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set(headerAuthorization, "Token "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	return resp, nil
}
