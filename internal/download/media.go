// Package download handles the downloading of videos and channels from
// SwitchTube.
package download

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"switchtube-downloader/internal/helper/dir"
	"switchtube-downloader/internal/models"
	"switchtube-downloader/internal/token"
)

const (
	// Base URL and API endpoints for SwitchTube.
	baseURL             = "https://tube.switch.ch/"
	videoAPI            = "api/v1/browse/videos/"
	channelAPI          = "api/v1/browse/channels/"
	videoPrefix         = "videos/"
	channelPrefix       = "channels/"
	headerAuthorization = "Authorization"

	defaultTimeout = 30 * time.Second
)

type mediaType int

const (
	unknownType mediaType = iota
	videoType
	channelType
)

var (
	errCouldNotDetermineType   = errors.New("could not determine if channel or video")
	errFailedDecodeResponse    = errors.New("failed to decode response")
	errFailedToCreateRequest   = errors.New("failed to create request")
	errFailedToDownloadChannel = errors.New("failed to download channel")
	errFailedToDownloadVideo   = errors.New("failed to download video")
	errFailedToExtractType     = errors.New("failed to extract type")
	errFailedToGetToken        = errors.New("failed to get token")
	errInvalidURL              = errors.New("invalid url")
)

// Client handles all API interactions.
type Client struct {
	tokenManager *token.Manager
	client       *http.Client
}

// NewClient creates a new instance of Client.
func NewClient(tm *token.Manager) *Client {
	return &Client{
		tokenManager: tm,
		client: &http.Client{
			Timeout:       defaultTimeout,
			Transport:     http.DefaultTransport,
			CheckRedirect: nil,
			Jar:           nil,
		},
	}
}

// makeRequest makes an authenticated HTTP request.
func (c *Client) makeRequest(url string) (*http.Response, error) {
	apiToken, err := c.tokenManager.Get()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errFailedToGetToken, err)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errFailedToCreateRequest, err)
	}

	req.Header.Set(headerAuthorization, "Token "+apiToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errFailedToCreateRequest, err)
	}

	return resp, nil
}

// makeRequest makes an authenticated HTTP request and decodes the response.
func (c *Client) makeJSONRequest(url string, target any) error {
	resp, err := c.makeRequest(url)
	if err != nil {
		return err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Warning: failed to close response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d: %s",
			errHTTPNotOK,
			resp.StatusCode,
			http.StatusText(resp.StatusCode))
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("%w: %w", errFailedDecodeResponse, err)
	}

	return nil
}

// Download initiates the download process based on the provided configuration.
func Download(config models.DownloadConfig) error {
	id, downloadType, err := extractIDAndType(config.Media)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToExtractType, err)
	}

	tokenMgr := token.NewTokenManager()
	client := NewClient(tokenMgr)

	// Create default progress info for single video downloads
	progress := models.ProgressInfo{
		CurrentItem: 1,
		TotalItems:  1,
	}

	switch downloadType {
	case videoType:
		downloader := newVideoDownloader(config, progress, client)
		if err = downloader.downloadVideo(id, true); err != nil {
			return fmt.Errorf("%w: %w", errFailedToDownloadVideo, err)
		}
	case unknownType:
		// If the type is unknown, we try to download as a video first.
		downloader := newVideoDownloader(config, progress, client)
		if err = downloader.downloadVideo(id, true); err == nil {
			return nil
		} else if errors.Is(err, dir.ErrCreateFile) {
			return fmt.Errorf("%w", err)
		}

		fallthrough
	case channelType:
		downloader := newChannelDownloader(config, client)
		if err = downloader.downloadChannel(id); err != nil {
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
