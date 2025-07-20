package download

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
)

// videoVariant represents a video download variant.
type videoVariant struct {
	Path      string `json:"path"`
	MediaType string `json:"mediaType"`
}

var (
	errNoVariantsFound        = errors.New("no video variants found")
	errFailedDecodeVariants   = errors.New("failed to decode variants")
	errFailedGetVideoVariants = errors.New("failed to get video variants")
	errFailedGetVideoInfo     = errors.New("failed to get video information")
	errFailedDecodeVideoMeta  = errors.New("failed to decode video metadata")
)

// downloadVideo downloads a video.
func downloadVideo(
	videoID string,
	token string,
	currentItem int,
	totalItems int,
	useEpisode bool,
	force bool,
) error {
	videoData, err := getVideoMetadata(videoID, token)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedGetVideoInfo, err)
	}

	variants, err := getVideoVariants(videoID, token)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedGetVideoVariants, err)
	}

	filename := createFilename(
		videoData.Title,
		variants[0].MediaType,
		videoData.Episode,
		useEpisode,
	)

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
func getVideoMetadata(videoID, token string) (*video, error) {
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
		return nil, fmt.Errorf(
			"%w: status %d: %s",
			errHTTPNotOK,
			resp.StatusCode,
			http.StatusText(resp.StatusCode),
		)
	}

	var videoData video

	err = json.NewDecoder(resp.Body).Decode(&videoData)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errFailedDecodeVideoMeta, err)
	}

	return &videoData, nil
}

// getVideoVariants retrieves available video variants from the API.
func getVideoVariants(videoID, token string) ([]videoVariant, error) {
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
		return nil, fmt.Errorf(
			"%w: status %d: %s",
			errHTTPNotOK,
			resp.StatusCode,
			http.StatusText(resp.StatusCode),
		)
	}

	var variants []videoVariant

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
func createFilename(title string, mediaType string, episodeNr string, useEpisode bool) string {
	// Extract extension from media type (e.g., "video/mp4" -> "mp4")
	parts := strings.Split(mediaType, "/")

	extension := "mp4" // default fallback
	if len(parts) >= minMediaTypeParts {
		extension = parts[1]
	}

	sanitizedTitle := sanitizeFilename(title)
	sanitizedTitle = strings.ReplaceAll(sanitizedTitle, " ", "_")

	// Add episode prefix if episode flag is set
	var filename string
	if useEpisode && episodeNr != "" {
		filename = fmt.Sprintf("%s_%s.%s", episodeNr, sanitizedTitle, extension)
	} else {
		filename = fmt.Sprintf("%s.%s", sanitizedTitle, extension)
	}

	return filepath.Clean(filename)
}

// sanitizeFilename removes or replaces characters that are invalid in filenames.
func sanitizeFilename(filename string) string {
	replacements := map[string]string{
		"/":  "-",
		"\\": "-",
		":":  "-",
		"*":  "",
		"?":  "",
		"\"": "",
		"<":  "",
		">":  "",
		"|":  "-",
	}

	sanitized := filename
	for invalid, replacement := range replacements {
		sanitized = strings.ReplaceAll(sanitized, invalid, replacement)
	}

	sanitized = strings.TrimSpace(sanitized)
	for strings.Contains(sanitized, "--") {
		sanitized = strings.ReplaceAll(sanitized, "--", "-")
	}

	return sanitized
}
