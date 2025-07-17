// Package media handles the downloading of videos and channels from SwitchTube.
package media

import (
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

type MediaType int

const (
	unknown MediaType = iota
	video
	channel
)

// Download downloads a video or a channel
func Download(media string, force bool, all bool) error {
	id, downloadType, err := extractIDAndType(media)
	if err != nil {
		return fmt.Errorf("failed to extract type: %w", err)
	}

	token, err := token.Get()
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	switch downloadType {
	case video:
		if err := downloadVideo(id, token, 1, 1, force); err != nil {
			return fmt.Errorf("failed to download video: %w", err)
		}
	case unknown:
		if err := downloadVideo(id, token, 1, 1, force); err == nil {
			return nil
		}
		fallthrough
	case channel:
		if err := downloadChannel(id, token, force, all); err != nil {
			return fmt.Errorf("failed to download channel: %w", err)
		}
	default:
		return fmt.Errorf("could not determine if channel or video")
	}

	return nil
}

// ExtractIDAndType extracts the id and determines if it's a video or channel.
func extractIDAndType(input string) (string, MediaType, error) {
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
		return id, unknown, fmt.Errorf("invalid url")
	}
}
