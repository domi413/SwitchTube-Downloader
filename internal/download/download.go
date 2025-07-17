package media

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const BaseURL = "https://tube.switch.ch"

type Video struct {
	ID    string  `json:"id"`
	Title *string `json:"title"`
}

// String returns the string representation of a Video, preferring the title if available, otherwise the ID.
func (v Video) String() string {
	if v.Title != nil {
		return *v.Title
	}
	return v.ID
}

// MakeRequest makes an authenticated HTTP request.
func MakeRequest(url, token string) (*http.Response, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Token "+token)
	return client.Do(req)
}

// DownloadVideo downloads a single video (used by both video and channel downloads).
func DownloadVideo(videoID, token string, currentItem, totalItems int) error {
	// Get video info
	resp, err := MakeRequest(fmt.Sprintf("%s/api/v1/browse/videos/%s", BaseURL, videoID), token)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var video Video
	if err := json.NewDecoder(resp.Body).Decode(&video); err != nil {
		return err
	}

	// Get download info
	resp, err = MakeRequest(fmt.Sprintf("%s/api/v1/browse/videos/%s/video_variants", BaseURL, videoID), token)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var variants []struct {
		Path      string `json:"path"`
		MediaType string `json:"media_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&variants); err != nil {
		return err
	}
	if len(variants) == 0 {
		return fmt.Errorf("no variants found")
	}

	// Download file
	parts := strings.Split(variants[0].MediaType, "/")
	if len(parts) < 2 {
		return fmt.Errorf("invalid media type: %s", variants[0].MediaType)
	}
	extension := parts[1]
	filename := fmt.Sprintf("%s.%s", video.String(), extension)
	filename = strings.ReplaceAll(filename, "/", " - ")

	return downloadFile(variants[0].Path, filename, token, currentItem, totalItems)
}

// downloadFile downloads a file with progress bar.
func downloadFile(path, filename, token string, currentItem, totalItems int) error {
	resp, err := MakeRequest(BaseURL+path, token)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()

	var written int64
	startTime := time.Now()

	for {
		buffer := make([]byte, 32*1024)
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			if _, err := out.Write(buffer[:n]); err != nil {
				return err
			}
			written += int64(n)
			ShowProgress(written, resp.ContentLength, filename, currentItem, totalItems, startTime)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}
