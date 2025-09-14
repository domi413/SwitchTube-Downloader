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

// videoDownloader handles the downloading of individual videos.
type videoDownloader struct {
	config   models.DownloadConfig
	progress models.ProgressInfo
	client   *Client
}

// newVideoDownloader creates a new instance of VideoDownloader.
func newVideoDownloader(
	config models.DownloadConfig,
	progress models.ProgressInfo,
	client *Client,
) *videoDownloader {
	return &videoDownloader{
		config:   config,
		progress: progress,
		client:   client,
	}
}

// downloadVideo downloads a video.
func (vd *videoDownloader) downloadVideo(videoID string, checkExists bool) error {
	video, err := vd.getMetadata(videoID)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedGetVideoInfo, err)
	}

	variants, err := vd.getVariants(videoID)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedGetVideoVariants, err)
	}

	if len(variants) == 0 {
		return errNoVariantsFound
	}

	filename := dir.CreateFilename(video.Title, variants[0].MediaType, video.Episode, vd.config)
	if checkExists && dir.OverwriteVideoIfExists(filename, vd.config) {
		return nil // Skip download
	}

	file, err := dir.CreateVideoFile(filename)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToCreateVideoFile, err)
	}

	// Download the video
	err = vd.downloadProcess(variants[0].Path, file)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToDownloadVideo, err)
	}

	return nil
}

// getMetadata retrieves video metadata from the API.
func (vd *videoDownloader) getMetadata(videoID string) (*models.Video, error) {
	fullURL, err := url.JoinPath(baseURL, videoAPI, videoID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errFailedConstructURL, err)
	}

	resp, err := vd.client.makeRequest(fullURL)
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

// getVariants retrieves available video variants from the API.
func (vd *videoDownloader) getVariants(videoID string) ([]videoVariant, error) {
	fullURL, err := url.JoinPath(baseURL, videoAPI, videoID, "video_variants")
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errFailedConstructURL, err)
	}

	resp, err := vd.client.makeRequest(fullURL)
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

	return variants, nil
}

// downloadProcess handles the actual file download.
func (vd *videoDownloader) downloadProcess(endpoint string, file *os.File) error {
	fullURL, err := url.JoinPath(baseURL, endpoint)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedConstructURL, err)
	}

	resp, err := vd.client.makeRequest(fullURL)
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

	currentItem := vd.progress.CurrentItem
	if currentItem == 0 {
		currentItem = 1
	}

	totalItems := vd.progress.TotalItems
	if totalItems == 0 {
		totalItems = 1
	}

	err = ui.ProgressBar(resp.Body, file, resp.ContentLength, file.Name(), currentItem, totalItems)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedCopyVideoData, err)
	}

	return nil
}
