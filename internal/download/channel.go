package download

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"switchtube-downloader/internal/helper/dir"
	"switchtube-downloader/internal/helper/ui"
	"switchtube-downloader/internal/models"
)

// channelInfo represents channel metadata.
type channelInfo struct {
	Name string `json:"name"`
}

var (
	errFailedDecodeChannelMeta     = errors.New("failed to decode channel metadata")
	errFailedDecodeChannelVids     = errors.New("failed to decode channel videos")
	errFailedGetChannelInfo        = errors.New("failed to get channel information")
	errFailedGetChannelVideos      = errors.New("failed to get channel videos")
	errFailedSelectVideos          = errors.New("failed to select videos")
	errFailedToCreateChannelFolder = errors.New("failed to create channel folder")
)

// downloadChannel downloads selected videos from a channel.
func downloadChannel(channelID string, token string, config models.DownloadConfig) error {
	channelInfo, err := getChannelInfo(channelID, token)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedGetChannelInfo, err)
	}

	videos, err := getChannelVideos(channelID, token)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedGetChannelVideos, err)
	}

	if len(videos) == 0 {
		fmt.Println("No videos found in this channel")

		return nil
	}

	fmt.Printf("Found %d videos in channel\n", len(videos))

	selectedIndices, err := ui.SelectVideos(videos, config.All)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedSelectVideos, err)
	}

	if len(selectedIndices) == 0 {
		fmt.Println("No videos selected for download")

		return nil
	}

	folderName, err := dir.CreateChannelFolder(channelInfo.Name, config)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToCreateChannelFolder, err)
	}

	config.Output = folderName

	fmt.Printf("Downloading to folder: %s\n", folderName)
	downloadSelectedVideos(videos, selectedIndices, token, config)

	return nil
}

// downloadSelectedVideos downloads the selected videos and reports results.
func downloadSelectedVideos(
	videos []models.Video,
	selectedIndices []int,
	token string,
	config models.DownloadConfig,
) {
	var failed []string

	for i, videoIndex := range selectedIndices {
		video := videos[videoIndex]

		err := downloadVideo(video.ID, token, i+1, len(selectedIndices), config)
		if err != nil && !errors.Is(err, dir.ErrFileCreationAborted) {
			fmt.Printf("\nFailed: %s - %v\n", video.Title, err)
			failed = append(failed, video.Title)
		}
	}

	fmt.Printf("\nDownload complete! %d/%d videos successful\n",
		len(selectedIndices)-len(failed), len(selectedIndices))

	if len(failed) > 0 {
		fmt.Println("Failed downloads:")

		for _, title := range failed {
			fmt.Printf("  - %s\n", title)
		}
	}
}

// getChannelVideos retrieves all videos from a channel.
func getChannelVideos(channelID, token string) ([]models.Video, error) {
	fullURL, err := url.JoinPath(baseURL, channelAPI, channelID, "videos")
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

	var videos []models.Video
	if err = json.NewDecoder(resp.Body).Decode(&videos); err != nil {
		return nil, fmt.Errorf("%w: %w", errFailedDecodeChannelVids, err)
	}

	return videos, nil
}

// getChannelInfo retrieves channel metadata from the API.
func getChannelInfo(channelID, token string) (*channelInfo, error) {
	fullURL, err := url.JoinPath(baseURL, channelAPI, channelID)
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

	var channelData channelInfo
	if err = json.NewDecoder(resp.Body).Decode(&channelData); err != nil {
		return nil, fmt.Errorf("%w: %w", errFailedDecodeChannelMeta, err)
	}

	return &channelData, nil
}
