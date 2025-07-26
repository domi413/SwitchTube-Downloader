package download

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"switchtube-downloader/internal/helper/dir"
	"switchtube-downloader/internal/helper/ui"
	"switchtube-downloader/internal/models"
)

// videoVariant represents a video download variant.
type videoVariant struct {
	Path      string `json:"path"`
	MediaType string `json:"mediaType"`
}

var (
	errFailedConstructURL      = errors.New("failed to construct URL")
	errFailedCopyVideoData     = errors.New("failed to copy video data")
	errFailedDecodeVariants    = errors.New("failed to decode variants")
	errFailedDecodeVideoMeta   = errors.New("failed to decode video metadata")
	errFailedFetchVideoStream  = errors.New("failed to fetch video stream")
	errFailedGetVideoInfo      = errors.New("failed to get video information")
	errFailedGetVideoVariants  = errors.New("failed to get video variants")
	errFailedToCreateVideoFile = errors.New("failed to create video file")
	errHTTPNotOK               = errors.New("HTTP request failed with non-OK status")
	errNoVariantsFound         = errors.New("no video variants found")
)

// downloadVideo downloads a video.
func downloadVideo(
	videoID string,
	token string,
	currentItem int,
	totalItems int,
	config models.DownloadConfig,
) error {
	videoData, err := getVideoMetadata(videoID, token)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedGetVideoInfo, err)
	}

	variants, err := getVideoVariants(videoID, token)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedGetVideoVariants, err)
	}

	file, err := dir.CreateVideoFile(
		videoData.Title,
		variants[0].MediaType,
		videoData.Episode,
		config,
	)
	if errors.Is(err, dir.ErrFileCreationAborted) {
		return fmt.Errorf("%w", dir.ErrFileCreationAborted)
	} else if err != nil {
		return fmt.Errorf("%w: %w", errFailedToCreateVideoFile, err)
	}

	// Download the video
	err = downloadProcess(variants[0].Path, token, file, file.Name(), currentItem, totalItems)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToDownloadVideo, err)
	}

	return nil
}

// getVideoMetadata retrieves video metadata from the API.
func getVideoMetadata(videoID, token string) (*models.Video, error) {
	fullURL, err := url.JoinPath(baseURL, videoAPI, videoID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errFailedConstructURL, err)
	}

	resp, err := makeRequest(fullURL, token)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errFailedFetchVideoStream, err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
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

	var videoData models.Video
	if err = json.NewDecoder(resp.Body).Decode(&videoData); err != nil {
		return nil, fmt.Errorf("%w: %w", errFailedDecodeVideoMeta, err)
	}

	return &videoData, nil
}

// getVideoVariants retrieves available video variants from the API.
func getVideoVariants(videoID, token string) ([]videoVariant, error) {
	fullURL, err := url.JoinPath(baseURL, videoAPI, videoID, "video_variants")
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errFailedConstructURL, err)
	}

	resp, err := makeRequest(fullURL, token)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errFailedFetchVideoStream, err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
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
	if err = json.NewDecoder(resp.Body).Decode(&variants); err != nil {
		return nil, fmt.Errorf("%w: %w", errFailedDecodeVariants, err)
	}

	if len(variants) == 0 {
		return nil, errNoVariantsFound
	}

	return variants, nil
}

// downloadProcess handles the actual file download.
func downloadProcess(
	endpoint string,
	token string,
	file *os.File,
	filename string,
	currentItem int,
	totalItems int,
) error {
	fullURL, err := url.JoinPath(baseURL, endpoint)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedConstructURL, err)
	}

	resp, err := makeRequest(fullURL, token)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedFetchVideoStream, err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Warning: failed to close response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"%w: status %d: %s",
			errHTTPNotOK,
			resp.StatusCode,
			http.StatusText(resp.StatusCode),
		)
	}

	err = ui.ProgressBar(resp.Body, file, resp.ContentLength, filename, currentItem, totalItems)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedCopyVideoData, err)
	}

	return nil
}
