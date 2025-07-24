// Package dir provides functions to create video files and channel folders.
package dir

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"switch-tube-downloader/internal/helper/ui"
	"switch-tube-downloader/internal/models"
)

const (
	// File and directory permissions.
	dirPermissions = 0o755

	// Minimum number of parts in a media type string (e.g., "video/mp4" has 2
	// parts).
	minMediaTypeParts = 2
)

var (
	// ErrCreateFile is returned when file creation fails.
	ErrCreateFile = errors.New("failed to create file")

	// ErrFileCreationAborted is returned when the user aborts file creation.
	ErrFileCreationAborted = errors.New("file creation aborted")

	errFailedCreateFolder = errors.New("failed to create folder")
)

// CreateVideoFile creates a sanitized filename from video title and media type,
// and opens the file for writing.
func CreateVideoFile(
	title string,
	mediaType string,
	episodeNr string,
	config models.DownloadConfig,
) (*os.File, error) {
	filename := createFilename(title, mediaType, episodeNr, config.UseEpisode)

	if config.Output != "" {
		filename = filepath.Join(config.Output, filename)
	}

	if !config.Force {
		_, err := os.Stat(filename)
		if err == nil {
			if config.Skip || !ui.Confirm("File %s already exists. Overwrite?", filename) {
				return nil, fmt.Errorf("%w", ErrFileCreationAborted)
			}
		}
	}

	err := os.MkdirAll(filepath.Dir(filename), dirPermissions)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errFailedCreateFolder, err)
	}

	fd, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateFile, err)
	}

	return fd, nil
}

// CreateChannelFolder creates a folder for the channel using its name.
func CreateChannelFolder(channelName string, config models.DownloadConfig) (string, error) {
	folderName := strings.ReplaceAll(channelName, "/", " - ")
	folderName = filepath.Clean(folderName)

	if config.Output != "" {
		folderName = filepath.Join(config.Output, folderName)
	}

	err := os.MkdirAll(folderName, dirPermissions)
	if err != nil {
		return "", fmt.Errorf("%w: %w", errFailedCreateFolder, err)
	}

	return folderName, nil
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

// sanitizeFilename removes or replaces characters that are invalid in
// filenames.
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
