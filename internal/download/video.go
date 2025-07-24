package download

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"switch-tube-downloader/internal/helper/file"
	"switch-tube-downloader/internal/helper/ui"
	"switch-tube-downloader/internal/models"
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
	errFailedReadData          = errors.New("failed to read data")
	errFailedToCreateVideoFile = errors.New("failed to create video file")
	errFailedWriteToFile       = errors.New("failed to write to file")
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

	file, err := file.CreateVideoFile(
		videoData.Title,
		variants[0].MediaType,
		videoData.Episode,
		config.UseEpisode,
		config.Force,
	)
	if err != nil {
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

	var videoData models.Video

	err = json.NewDecoder(resp.Body).Decode(&videoData)
	if err != nil {
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
		err := resp.Body.Close()
		if err != nil {
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

	err = copyWithProgress(resp.Body, file, resp.ContentLength, filename, currentItem, totalItems)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedCopyVideoData, err)
	}

	return nil
}

// TODO: Do we really need this function?
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
			ui.ShowProgress(written, total, filename, currentItem, totalItems, startTime)
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("%w: %w", errFailedReadData, err)
		}
	}

	ui.ShowProgress(written, total, filename, currentItem, totalItems, startTime)

	return nil
}
