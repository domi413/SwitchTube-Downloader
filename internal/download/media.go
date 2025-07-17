// Package media handles the downloading of videos and channels from SwitchTube.
package media

import (
	"errors"
	"fmt"
	"strings"

	"switch-tube-downloader/internal/token"
)

const (
	baseURL       = "https://tube.switch.ch/"
	videoAPI      = "api/v1/browse/videos/"
	channelAPI    = "api/v1/browse/channels/"
	videoPrefix   = "videos/"
	channelPrefix = "channels/"
)

type mediaType int

const (
	unknown mediaType = iota
	video
	channel
)

var (
	errInvalidURL              = errors.New("invalid url")
	errFailedToGetToken        = errors.New("failed to get token")
	errFailedToExtractType     = errors.New("failed to extract type")
	errFailedToDownloadVideo   = errors.New("failed to download video")
	errFailedToDownloadChannel = errors.New("failed to download channel")
	errCouldNotDetermineType   = errors.New("could not determine if channel or video")
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

// ExtractIDAndType extracts the id and determines if it's a video or channel.
func extractIDAndType(input string) (string, mediaType, error) {
	input = strings.TrimSpace(input)

	// If input doesn't start with baseURL, return as unknown ExtractIDAndType
	if !strings.HasPrefix(input, baseURL) {
		return input, unknown, nil
	}

	id := strings.TrimPrefix(baseURL, input)

	switch {
	case strings.HasPrefix(id, videoPrefix):
		return strings.TrimPrefix(id, videoPrefix), video, nil
	case strings.HasPrefix(id, channelPrefix):
		return strings.TrimPrefix(id, channelPrefix), channel, nil
	default:
		return id, unknown, errInvalidURL
	}
}
