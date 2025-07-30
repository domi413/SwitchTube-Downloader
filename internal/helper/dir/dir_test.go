package dir

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"switchtube-downloader/internal/models"
)

func TestCreateFilename(t *testing.T) {
	tests := []struct {
		name      string
		title     string
		mediaType string
		episodeNr string
		config    models.DownloadConfig
		want      string
	}{
		{
			name:      "basic video",
			title:     "Test Video",
			mediaType: "video/mp4",
			episodeNr: "",
			config:    models.DownloadConfig{UseEpisode: false},
			want:      "Test_Video.mp4",
		},
		{
			name:      "video with episode number",
			title:     "Test Video",
			mediaType: "video/mp4",
			episodeNr: "E01",
			config:    models.DownloadConfig{UseEpisode: true},
			want:      "E01_Test_Video.mp4",
		},
		{
			name:      "invalid media type",
			title:     "Test Video",
			mediaType: "invalid",
			episodeNr: "",
			config:    models.DownloadConfig{UseEpisode: false},
			want:      "Test_Video.mp4",
		},
		{
			name:      "video with invalid characters",
			title:     "Test/Video:With*Invalid?Chars",
			mediaType: "video/mp4",
			episodeNr: "",
			config:    models.DownloadConfig{UseEpisode: false},
			want:      "Test-Video-WithInvalidChars.mp4",
		},
		{
			name:      "video with output path",
			title:     "Test Video",
			mediaType: "video/mp4",
			episodeNr: "",
			config:    models.DownloadConfig{Output: "output", UseEpisode: false},
			want:      filepath.Join("output", "Test_Video.mp4"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CreateFilename(tt.title, tt.mediaType, tt.episodeNr, tt.config)
			if got != tt.want {
				t.Errorf("CreateFilename() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOverwriteVideoIfExists(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		config      models.DownloadConfig
		wantPrompt  string
		promptInput string
		wantValue   bool
		createFile  bool // Whether to create the file to simulate existing file
	}{
		{
			name:        "video exists, overwrite",
			filename:    "existing_video.mp4",
			config:      models.DownloadConfig{},
			wantPrompt:  "File existing_video.mp4 already exists. Overwrite? (y/N): ",
			promptInput: "y\n",
			wantValue:   false,
			createFile:  true,
		},
		{
			name:        "video exists, do not overwrite",
			filename:    "existing_video.mp4",
			config:      models.DownloadConfig{},
			wantPrompt:  "File existing_video.mp4 already exists. Overwrite? (y/N): ",
			promptInput: "\n",
			wantValue:   true,
			createFile:  true,
		},
		{
			name:        "video does not exist",
			filename:    "non_existing_video.mp4",
			config:      models.DownloadConfig{},
			wantPrompt:  "",
			promptInput: "",
			wantValue:   false,
			createFile:  false,
		},
		{
			name:        "video exists, force-flag set",
			filename:    "existing_video.mp4",
			config:      models.DownloadConfig{Force: true},
			wantPrompt:  "",
			promptInput: "",
			wantValue:   false,
			createFile:  true,
		},
		{
			name:        "video does not exist, force-flag set",
			filename:    "non_existing_video.mp4",
			config:      models.DownloadConfig{Force: true},
			wantPrompt:  "",
			promptInput: "",
			wantValue:   false,
			createFile:  false,
		},
		{
			name:        "video exists, skip-flag set",
			filename:    "existing_video.mp4",
			config:      models.DownloadConfig{Skip: true},
			wantPrompt:  "",
			promptInput: "",
			wantValue:   true,
			createFile:  true,
		},
		{
			name:        "video does not exist, skip-flag set",
			filename:    "non_existing_video.mp4",
			config:      models.DownloadConfig{Skip: true},
			wantPrompt:  "",
			promptInput: "",
			wantValue:   false,
			createFile:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			filename := filepath.Join(tempDir, tt.filename)

			if tt.createFile {
				if err := os.MkdirAll(filepath.Dir(filename), dirPermissions); err != nil {
					t.Fatalf("Failed to create directory: %v", err)
				}

				if _, err := os.Create(filename); err != nil {
					t.Fatalf("Failed to create file: %v", err)
				}
			}

			tmpFile, err := os.CreateTemp(t.TempDir(), "test-input")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err = tmpFile.WriteString(tt.promptInput); err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}

			if _, err = tmpFile.Seek(0, 0); err != nil {
				t.Fatalf("Failed to seek temp file: %v", err)
			}

			oldStdin := os.Stdin
			os.Stdin = tmpFile

			defer func() { os.Stdin = oldStdin }()

			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			defer func() { os.Stdout = oldStdout }()

			got := OverwriteVideoIfExists(filename, tt.config)

			w.Close()

			output := make([]byte, 1000)
			n, _ := r.Read(output)
			capturedOutput := string(output[:n])

			if tt.wantPrompt != "" {
				adjustedOutput := strings.ReplaceAll(
					capturedOutput,
					tempDir+string(os.PathSeparator),
					"",
				)
				if adjustedOutput != tt.wantPrompt {
					t.Errorf(
						"OverwriteVideoIfExists() prompt = %q, want %q",
						adjustedOutput,
						tt.wantPrompt,
					)
				}
			} else if capturedOutput != "" {
				t.Errorf("OverwriteVideoIfExists() prompt = %q, want empty", capturedOutput)
			}

			if got != tt.wantValue {
				t.Errorf("OverwriteVideoIfExists() = %v, want %v", got, tt.wantValue)
			}
		})
	}
}

func TestCreateVideoFile(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		wantErr    bool
		err        error
		createFile bool
	}{
		{
			name:       "create new video",
			filename:   "test_video.mp4",
			wantErr:    false,
			createFile: false,
		},
		{
			name:       "create video in subdirectory",
			filename:   filepath.Join("sub", "test_video.mp4"),
			wantErr:    false,
			createFile: false,
		},
		{
			name:       "video already exists",
			filename:   "existing_video.mp4",
			wantErr:    false,
			createFile: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			filename := filepath.Join(tempDir, tt.filename)

			if tt.createFile {
				if err := os.MkdirAll(filepath.Dir(filename), dirPermissions); err != nil {
					t.Fatalf("Failed to create directory: %v", err)
				}

				if _, err := os.Create(filename); err != nil {
					t.Fatalf("Failed to create file: %v", err)
				}
			}

			fd, err := CreateVideoFile(filename)
			if fd != nil {
				defer fd.Close()
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateVideoFile() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && !errors.Is(err, tt.err) {
				t.Errorf("CreateVideoFile() error = %v, want %v", err, tt.err)
			}

			if !tt.wantErr {
				if _, err := os.Stat(filename); os.IsNotExist(err) {
					t.Errorf("CreateVideoFile() did not create file %q", filename)
				}
			}
		})
	}
}

func TestCreateChannelFolder(t *testing.T) {
	tests := []struct {
		name        string
		channelName string
		config      models.DownloadConfig
		wantFolder  string
		wantErr     bool
		err         error
	}{
		{
			name:        "basic folder creation",
			channelName: "Test Channel",
			config:      models.DownloadConfig{Output: ""},
			wantFolder:  "Test Channel",
			wantErr:     false,
		},
		{
			name:        "folder with slashes",
			channelName: "Test/Channel",
			config:      models.DownloadConfig{Output: "output"},
			wantFolder:  filepath.Join("output", "Test - Channel"),
			wantErr:     false,
		},
		{
			name:        "empty channel name",
			channelName: "",
			config:      models.DownloadConfig{Output: ""},
			wantFolder:  ".",
			wantErr:     false,
		},
		{
			name:        "folder in specific output directory",
			channelName: "Test Channel",
			config:      models.DownloadConfig{Output: "test_path"},
			wantFolder:  filepath.Join("test_path", "Test Channel"),
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			tt.config.Output = filepath.Join(tempDir, tt.config.Output)

			folder, err := CreateChannelFolder(tt.channelName, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateChannelFolder() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && !errors.Is(err, tt.err) {
				t.Errorf("CreateChannelFolder() error = %v, want %v", err, tt.err)
			}

			if folder != filepath.Join(tempDir, tt.wantFolder) {
				t.Errorf(
					"CreateChannelFolder() folder = %q, want %q",
					folder,
					filepath.Join(tempDir, tt.wantFolder),
				)
			}

			if !tt.wantErr {
				if _, err := os.Stat(folder); os.IsNotExist(err) {
					t.Errorf("CreateChannelFolder() did not create folder %q", folder)
				}
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "basic sanitization",
			in:   "Test Video",
			want: "Test Video",
		},
		{
			name: "invalid characters",
			in:   "Test/Video:With*Invalid?Chars<>\\|",
			want: "Test-Video-WithInvalidChars-",
		},
		{
			name: "multiple dashes",
			in:   "Test///Video",
			want: "Test-Video",
		},
		{
			name: "leading and trailing spaces",
			in:   "  Test Video  ",
			want: "Test Video",
		},
		{
			name: "empty string",
			in:   "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeFilename(tt.in); got != tt.want {
				t.Errorf("sanitizeFilename() = %q, want %q", got, tt.want)
			}
		})
	}
}
