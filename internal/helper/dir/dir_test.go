package dir

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"switchtube-downloader/internal/models"
)

func TestCreateVideoFile(t *testing.T) {
	tests := []struct {
		name       string
		title      string
		mediaType  string
		episodeNr  string
		config     models.DownloadConfig
		input      string
		wantFile   string
		wantErr    bool
		err        error
		wantPrompt string
		createFile bool // Whether to create a file to simulate existing file
	}{
		{
			name:       "basic video file creation",
			title:      "Test Video",
			mediaType:  "video/mp4",
			episodeNr:  "",
			config:     models.DownloadConfig{},
			input:      "",
			wantFile:   "Test_Video.mp4",
			wantErr:    false,
			wantPrompt: "",
		},
		{
			name:       "video file with episode number",
			title:      "Test Video",
			mediaType:  "video/mp4",
			episodeNr:  "69",
			config:     models.DownloadConfig{UseEpisode: true},
			input:      "",
			wantFile:   "69_Test_Video.mp4",
			wantErr:    false,
			wantPrompt: "",
		},
		{
			name:       "download video to specific output dir",
			title:      "Test Video",
			mediaType:  "video/mp4",
			episodeNr:  "",
			config:     models.DownloadConfig{Output: "test_path"},
			input:      "",
			wantFile:   filepath.Join("test_path", "Test_Video.mp4"),
			wantErr:    false,
			wantPrompt: "",
		},
		{
			name:       "video file with invalid characters",
			title:      "Test/Video:With*Invalid?Chars",
			mediaType:  "video/mp4",
			episodeNr:  "",
			config:     models.DownloadConfig{},
			input:      "",
			wantFile:   "Test-Video-WithInvalidChars.mp4",
			wantErr:    false,
			wantPrompt: "",
		},
		{
			name:       "existing file with overwrite confirmation",
			title:      "Test Video",
			mediaType:  "video/mp4",
			episodeNr:  "",
			config:     models.DownloadConfig{},
			input:      "y\n",
			wantFile:   "Test_Video.mp4",
			wantErr:    false,
			wantPrompt: "File Test_Video.mp4 already exists. Overwrite? (y/N): ",
			createFile: true,
		},
		{
			name:       "existing file aborted",
			title:      "Test Video",
			mediaType:  "video/mp4",
			episodeNr:  "",
			config:     models.DownloadConfig{},
			input:      "n\n",
			wantFile:   "",
			wantErr:    true,
			err:        ErrFileCreationAborted,
			wantPrompt: "File Test_Video.mp4 already exists. Overwrite? (y/N): ",
			createFile: true,
		},
		{
			name:       "existing file with skip",
			title:      "Test Video",
			mediaType:  "video/mp4",
			episodeNr:  "",
			config:     models.DownloadConfig{Skip: true},
			input:      "",
			wantFile:   "",
			wantErr:    true,
			err:        ErrFileCreationAborted,
			wantPrompt: "",
			createFile: true,
		},
		{
			name:       "existing file with force",
			title:      "Test Video",
			mediaType:  "video/mp4",
			episodeNr:  "",
			config:     models.DownloadConfig{Force: true},
			input:      "",
			wantFile:   "Test_Video.mp4",
			wantErr:    false,
			wantPrompt: "",
			createFile: true,
		},
		{
			name:       "invalid media type fallback",
			title:      "Test Video",
			mediaType:  "invalid",
			episodeNr:  "",
			config:     models.DownloadConfig{},
			input:      "",
			wantFile:   "Test_Video.mp4",
			wantErr:    false,
			wantPrompt: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			tt.config.Output = filepath.Join(tempDir, tt.config.Output)

			// Simulate existing file
			if tt.createFile {
				filename := filepath.Join(
					tempDir,
					createFilename(tt.title, tt.mediaType, tt.episodeNr, tt.config.UseEpisode),
				)

				if err := os.MkdirAll(filepath.Dir(filename), dirPermissions); err != nil {
					t.Fatalf("Failed to create directory: %v", err)
				}

				if _, err := os.Create(filename); err != nil {
					t.Fatalf("Failed to create file: %v", err)
				}
			}

			// Simulate stdin for ui.Confirm
			tmpFile, err := os.CreateTemp(t.TempDir(), "test-input")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err = tmpFile.WriteString(tt.input); err != nil {
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

			fd, err := CreateVideoFile(tt.title, tt.mediaType, tt.episodeNr, tt.config)
			if fd != nil {
				defer fd.Close()
			}

			w.Close()

			output := make([]byte, 1000)
			n, _ := r.Read(output)
			capturedOutput := string(output[:n])

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateVideoFile() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && !errors.Is(err, tt.err) {
				t.Errorf("CreateVideoFile() error = %v, want %v", err, tt.err)
			}

			if !tt.wantErr {
				if fd == nil {
					t.Error("CreateVideoFile() returned nil file descriptor, expected non-nil")
				} else {
					gotFile := fd.Name()
					if gotFile != filepath.Join(tempDir, tt.wantFile) {
						t.Errorf("CreateVideoFile() file = %q, want %q", gotFile, filepath.Join(tempDir, tt.wantFile))
					}
				}
			}

			if tt.wantPrompt != "" {
				// Remove tempDir prefix for comparison
				adjustedOutput := strings.ReplaceAll(
					capturedOutput,
					tempDir+string(os.PathSeparator),
					"",
				)
				if adjustedOutput != tt.wantPrompt {
					t.Errorf(
						"CreateVideoFile() prompt = %q, want %q",
						adjustedOutput,
						tt.wantPrompt,
					)
				}
			} else if capturedOutput != "" {
				t.Errorf("CreateVideoFile() prompt = %q, want empty", capturedOutput)
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

func TestCreateFilename(t *testing.T) {
	tests := []struct {
		name       string
		title      string
		mediaType  string
		episodeNr  string
		useEpisode bool
		want       string
	}{
		{
			name:       "basic filename",
			title:      "Test Video",
			mediaType:  "video/mp4",
			episodeNr:  "",
			useEpisode: false,
			want:       "Test_Video.mp4",
		},
		{
			name:       "filename with episode",
			title:      "Test Video",
			mediaType:  "video/mp4",
			episodeNr:  "E01",
			useEpisode: true,
			want:       "E01_Test_Video.mp4",
		},
		{
			name:       "invalid media type",
			title:      "Test Video",
			mediaType:  "invalid",
			episodeNr:  "",
			useEpisode: false,
			want:       "Test_Video.mp4",
		},
		{
			name:       "title with invalid characters",
			title:      "Test/Video:With*Invalid?Chars",
			mediaType:  "video/mp4",
			episodeNr:  "",
			useEpisode: false,
			want:       "Test-Video-WithInvalidChars.mp4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := createFilename(tt.title, tt.mediaType, tt.episodeNr, tt.useEpisode)
			if got != tt.want {
				t.Errorf("createFilename() = %q, want %q", got, tt.want)
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
