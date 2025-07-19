// Package download handles the downloading of videos and channels from SwitchTube.
package download

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"switch-tube-downloader/internal/prompt"
	"switch-tube-downloader/internal/token"
)

// Video represents a video.
type video struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Episode string `json:"episode"`
}

const (
	// Base URL and API endpoints for SwitchTube.
	baseURL             = "https://tube.switch.ch/"
	videoAPI            = "api/v1/browse/videos/"
	channelAPI          = "api/v1/browse/channels/"
	videoPrefix         = "videos/"
	channelPrefix       = "channels/"
	headerAuthorization = "Authorization"

	// File and directory permissions.
	dirPermissions = 0o744

	// Magic numbers.
	minMediaTypeParts = 2
	rangePartsCount   = 2
	bufferSizeKB      = 32
	bufferSize        = bufferSizeKB * 1024
)

type mediaType int

const (
	unknownType mediaType = iota
	videoType
	channelType
)

var (
	errInvalidURL              = errors.New("invalid url")
	errFailedToGetToken        = errors.New("failed to get token")
	errFailedReadData          = errors.New("failed to read data")
	errFileCreationAborted     = errors.New("file creation aborted")
	errFailedToExtractType     = errors.New("failed to extract type")
	errFailedWriteToFile       = errors.New("failed to write to file")
	errFailedConstructURL      = errors.New("failed to construct URL")
	errFailedToDownloadVideo   = errors.New("failed to download video")
	errFailedCopyVideoData     = errors.New("failed to copy video data")
	errFailedToDownloadChannel = errors.New("failed to download channel")
	errFailedFetchVideoStream  = errors.New("failed to fetch video stream")
	errHTTPNotOK               = errors.New("HTTP request failed with non-OK status")
	errCouldNotDetermineType   = errors.New("could not determine if channel or video")
)

// Download downloads a video or a channel.
func Download(media string, useEpisode bool, force bool, all bool) error {
	id, downloadType, err := extractIDAndType(media)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToExtractType, err)
	}

	token, err := token.Get()
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToGetToken, err)
	}

	switch downloadType {
	case videoType:
		err = downloadVideo(id, token, 1, 1, useEpisode, force)
		if err != nil {
			return fmt.Errorf("%w: %w (token might be invalid)", errFailedToDownloadVideo, err)
		}
	case unknownType:
		err = downloadVideo(id, token, 1, 1, useEpisode, force)
		if err == nil {
			return nil
		}

		fallthrough
	case channelType:
		err = downloadChannel(id, token, useEpisode, force, all)
		if err != nil {
			return fmt.Errorf("%w: %w (token might be invalid)", errFailedToDownloadChannel, err)
		}
	default:
		return errCouldNotDetermineType
	}

	return nil
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

// extractIDAndType extracts the id and determines if it's a video or channel.
func extractIDAndType(input string) (string, mediaType, error) {
	input = strings.TrimSpace(input)

	// If input doesn't start with baseURL, return as unknown type
	if !strings.HasPrefix(input, baseURL) {
		return input, unknownType, nil
	}

	id := strings.TrimPrefix(input, baseURL)

	switch {
	case strings.HasPrefix(id, videoPrefix):
		return strings.TrimPrefix(id, videoPrefix), videoType, nil
	case strings.HasPrefix(id, channelPrefix):
		return strings.TrimPrefix(id, channelPrefix), channelType, nil
	default:
		return id, unknownType, errInvalidURL
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
