package file

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"switch-tube-downloader/internal/helper/ui"
)

const (
	// File and directory permissions.
	dirPermissions = 0o744

	minMediaTypeParts = 2
)

var (
	errFailedCreateFolder  = errors.New("failed to create folder")
	errFileCreationAborted = errors.New("file creation aborted")
)

// CreateVideoFile creates a sanitized filename from video title and media type,
// and opens the file for writing.
func CreateVideoFile(
	title string,
	mediaType string,
	episodeNr string,
	useEpisode bool,
	force bool,
) (*os.File, error) {
	filename := createFilename(title, mediaType, episodeNr, useEpisode)

	if !force {
		_, err := os.Stat(filename)
		if !os.IsNotExist(err) {
			if !ui.Confirm("File %s already exists. Overwrite?", filename) {
				return nil, fmt.Errorf("%w", errFileCreationAborted)
			}
		}
	}

	err := os.MkdirAll(filepath.Dir(filename), dirPermissions)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errFailedCreateFolder, err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}

	return file, nil
}

// CreateChannelFolder creates a folder for the channel using its name.
func CreateChannelFolder(channelName string) (string, error) {
	folderName := strings.ReplaceAll(channelName, "/", " - ")
	folderName = filepath.Clean(folderName)

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
