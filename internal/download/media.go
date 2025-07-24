// Package download handles the downloading of videos and channels from SwitchTube.
package download

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"switch-tube-downloader/internal/models"
	token "switch-tube-downloader/internal/token"
)

const (
	// Base URL and API endpoints for SwitchTube.
	baseURL             = "https://tube.switch.ch/"
	videoAPI            = "api/v1/browse/videos/"
	channelAPI          = "api/v1/browse/channels/"
	videoPrefix         = "videos/"
	channelPrefix       = "channels/"
	headerAuthorization = "Authorization"

	// Buffer size for reading data
	bufferSizeKB = 32
	bufferSize   = bufferSizeKB * 1024
)

type mediaType int

const (
	unknownType mediaType = iota
	videoType
	channelType
)

var (
	errCouldNotDetermineType   = errors.New("could not determine if channel or video")
	errFailedToDownloadChannel = errors.New("failed to download channel")
	errFailedToDownloadVideo   = errors.New("failed to download video")
	errFailedToExtractType     = errors.New("failed to extract type")
	errFailedToGetToken        = errors.New("failed to get token")
	errInvalidURL              = errors.New("invalid url")
)

// Download downloads a video or a channel.
func Download(config models.DownloadConfig) error {
	id, downloadType, err := extractIDAndType(config.Media)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToExtractType, err)
	}

	token, err := token.Get()
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToGetToken, err)
	}

	switch downloadType {
	case videoType:
		err = downloadVideo(id, token, 1, 1, config)
		if err != nil {
			return fmt.Errorf("%w: %w", errFailedToDownloadVideo, err)
		}
	case unknownType:
		err = downloadVideo(id, token, 1, 1, config)
		if err == nil {
			return nil
		}

		fallthrough
	case channelType:
		err = downloadChannel(id, token, config)
		if err != nil {
			return fmt.Errorf("%w: %w", errFailedToDownloadChannel, err)
		}
	default:
		return errCouldNotDetermineType
	}

	return nil
}

// extractIDAndType extracts the id and determines if it's a video or channel.
func extractIDAndType(input string) (string, mediaType, error) {
	input = strings.TrimSpace(input)

	// If input doesn't start with baseURL, return as unknown type
	// This is the case if the Id was passed as an argument
	if !strings.HasPrefix(input, baseURL) {
		return input, unknownType, nil
	}

	switch prefixAndID := strings.TrimPrefix(input, baseURL); {
	case strings.HasPrefix(prefixAndID, videoPrefix):
		return strings.TrimPrefix(prefixAndID, videoPrefix), videoType, nil
	case strings.HasPrefix(prefixAndID, channelPrefix):
		return strings.TrimPrefix(prefixAndID, channelPrefix), channelType, nil
	default:
		return prefixAndID, unknownType, errInvalidURL
	}
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
