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

// channelMetadata represents channel metadata.
type channelMetadata struct {
	Name string `json:"name"`
}

var (
	errFailedDecodeChannelMeta     = errors.New("failed to decode channel metadata")
	errFailedDecodeChannelVideos   = errors.New("failed to decode channel videos")
	errFailedGetChannelInfo        = errors.New("failed to get channel information")
	errFailedGetChannelVideos      = errors.New("failed to get channel videos")
	errFailedSelectVideos          = errors.New("failed to select videos")
	errFailedToCreateChannelFolder = errors.New("failed to create channel folder")
)

// channelDownloader handles the downloading of channels.
type channelDownloader struct {
	config models.DownloadConfig
	client *Client
}

// newChannelDownloader creates a new instance of channelDownloader.
func newChannelDownloader(config models.DownloadConfig, client *Client) *channelDownloader {
	return &channelDownloader{
		config: config,
		client: client,
	}
}

// downloadChannel downloads selected videos from a channel.
func (cd *channelDownloader) downloadChannel(channelID string) error {
	channelInfo, err := cd.getMetadata(channelID)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedGetChannelInfo, err)
	}

	videos, err := cd.getVideos(channelID)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedGetChannelVideos, err)
	}

	if len(videos) == 0 {
		fmt.Println("No videos found in this channel")

		return nil
	}

	fmt.Printf("Found %d videos in channel\n", len(videos))

	selectedIndices, err := ui.SelectVideos(videos, cd.config.All)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedSelectVideos, err)
	}

	if len(selectedIndices) == 0 {
		fmt.Println("No videos selected for download")

		return nil
	}

	folderName, err := dir.CreateChannelFolder(channelInfo.Name, cd.config)
	if err != nil {
		return fmt.Errorf("%w: %w", errFailedToCreateChannelFolder, err)
	}

	cd.config.Output = folderName
	fmt.Printf("Downloading to folder: %s\n", folderName)
	cd.downloadSelectedVideos(videos, selectedIndices)

	return nil
}

// getMetadata retrieves channel metadata from the API.
func (cd *channelDownloader) getMetadata(channelID string) (*channelMetadata, error) {
	fullURL, err := url.JoinPath(baseURL, channelAPI, channelID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errFailedConstructURL, err)
	}

	resp, err := cd.client.makeRequest(fullURL)
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

	var channelData channelMetadata
	if err = json.NewDecoder(resp.Body).Decode(&channelData); err != nil {
		return nil, fmt.Errorf("%w: %w", errFailedDecodeChannelMeta, err)
	}

	return &channelData, nil
}

// getVideos retrieves all videos from a channel.
func (cd *channelDownloader) getVideos(channelID string) ([]models.Video, error) {
	fullURL, err := url.JoinPath(baseURL, channelAPI, channelID, "videos")
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errFailedConstructURL, err)
	}

	resp, err := cd.client.makeRequest(fullURL)
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
		return nil, fmt.Errorf("%w: %w", errFailedDecodeChannelVideos, err)
	}

	return videos, nil
}

// downloadSelectedVideos downloads the selected videos and reports results.
func (cd *channelDownloader) downloadSelectedVideos(videos []models.Video, selectedIndices []int) {
	var failed []string

	toDownload := cd.prepareDownloads(videos, selectedIndices, &failed)

	if len(toDownload) > 0 {
		failed = append(failed, cd.processDownloads(videos, toDownload)...)
	}

	cd.printResults(len(toDownload), len(selectedIndices), failed)
}

// prepareDownloads checks which videos need to be downloaded and validates their availability.
func (cd *channelDownloader) prepareDownloads(
	videos []models.Video,
	indices []int,
	failed *[]string,
) []int {
	var toDownload []int

	for _, idx := range indices {
		video := videos[idx]
		downloader := newVideoDownloader(
			cd.config,
			models.ProgressInfo{CurrentItem: 0, TotalItems: 0},
			cd.client,
		)

		variants, err := downloader.getVariants(video.ID)
		if err != nil {
			fmt.Printf("\nFailed to get video variants for %s: %v\n", video.Title, err)
			*failed = append(*failed, video.Title)

			continue
		}

		if len(variants) == 0 {
			fmt.Printf("\nNo variants found for %s\n", video.Title)
			*failed = append(*failed, video.Title)

			continue
		}

		filename := dir.CreateFilename(video.Title, variants[0].MediaType, video.Episode, cd.config)
		if !dir.OverwriteVideoIfExists(filename, cd.config) {
			toDownload = append(toDownload, idx)
		}
	}

	return toDownload
}

// processDownloads performs the actual video downloads and returns failed video titles.
func (cd *channelDownloader) processDownloads(videos []models.Video, indices []int) []string {
	var failed []string

	for i, idx := range indices {
		video := videos[idx]
		progress := models.ProgressInfo{
			CurrentItem: i + 1,
			TotalItems:  len(indices),
		}

		downloader := newVideoDownloader(cd.config, progress, cd.client)
		if err := downloader.downloadVideo(video.ID, false); err != nil {
			fmt.Printf("\nFailed: %s - %v\n", video.Title, err)
			failed = append(failed, video.Title)
		}
	}

	return failed
}

// printResults displays the download results summary.
func (cd *channelDownloader) printResults(downloadCount, selectedCount int, failed []string) {
	fmt.Printf("\nDownload complete! %d/%d videos successful\n",
		downloadCount-len(failed), selectedCount)

	if len(failed) > 0 {
		fmt.Println("Failed downloads:")

		for _, title := range failed {
			fmt.Printf("  - %s\n", title)
		}
	}
}
