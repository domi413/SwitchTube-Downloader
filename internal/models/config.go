package models

// DownloadConfig holds configuration options for the Download function.
type DownloadConfig struct {
	Media      string
	UseEpisode bool
	Force      bool
	All        bool
}
