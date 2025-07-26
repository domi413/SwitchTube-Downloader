package ui

import (
	"errors"
	"os"
	"testing"

	"switchtube-downloader/internal/models"
)

func TestSelectVideos(t *testing.T) {
	tests := []struct {
		name       string
		videos     []models.Video
		all        bool
		input      string
		want       []int
		wantErr    bool
		err        error
		wantPrompt string
	}{
		{
			name:       "select all with --all flag",
			videos:     []models.Video{{Title: "Video1"}, {Title: "Video2"}},
			all:        true,
			input:      "",
			want:       []int{0, 1},
			wantErr:    false,
			wantPrompt: "",
		},
		{
			name:    "select all with empty input",
			videos:  []models.Video{{Title: "Video1"}, {Title: "Video2"}},
			all:     false,
			input:   "\n",
			want:    []int{0, 1},
			wantErr: false,
			wantPrompt: "\nAvailable videos:\n1. Video1\n2. Video2\n\n" +
				"Select videos (e.g., '1-3', '1,3,5', '1 3 5', or Enter for all):\nSelection: ",
		},
		{
			name:    "select single video",
			videos:  []models.Video{{Title: "Video1"}, {Title: "Video2"}},
			all:     false,
			input:   "1\n",
			want:    []int{0},
			wantErr: false,
			wantPrompt: "\nAvailable videos:\n1. Video1\n2. Video2\n\n" +
				"Select videos (e.g., '1-3', '1,3,5', '1 3 5', or Enter for all):\nSelection: ",
		},
		{
			name:    "select range",
			videos:  []models.Video{{Title: "Video1"}, {Title: "Video2"}, {Title: "Video3"}},
			all:     false,
			input:   "1-3\n",
			want:    []int{0, 1, 2},
			wantErr: false,
			wantPrompt: "\nAvailable videos:\n1. Video1\n2. Video2\n3. Video3\n\n" +
				"Select videos (e.g., '1-3', '1,3,5', '1 3 5', or Enter for all):\nSelection: ",
		},
		{
			name:    "select multiple videos with comma",
			videos:  []models.Video{{Title: "Video1"}, {Title: "Video2"}, {Title: "Video3"}},
			all:     false,
			input:   "1, 3\n",
			want:    []int{0, 2},
			wantErr: false,
			wantPrompt: "\nAvailable videos:\n1. Video1\n2. Video2\n3. Video3\n\n" +
				"Select videos (e.g., '1-3', '1,3,5', '1 3 5', or Enter for all):\nSelection: ",
		},
		{
			name:    "select multiple videos with space",
			videos:  []models.Video{{Title: "Video1"}, {Title: "Video2"}, {Title: "Video3"}},
			all:     false,
			input:   "1 3\n",
			want:    []int{0, 2},
			wantErr: false,
			wantPrompt: "\nAvailable videos:\n1. Video1\n2. Video2\n3. Video3\n\n" +
				"Select videos (e.g., '1-3', '1,3,5', '1 3 5', or Enter for all):\nSelection: ",
		},
		{
			name:    "invalid number",
			videos:  []models.Video{{Title: "Video1"}},
			all:     false,
			input:   "abc\n",
			want:    nil,
			wantErr: true,
			err:     errInvalidNumber,
			wantPrompt: "\nAvailable videos:\n1. Video1\n\n" +
				"Select videos (e.g., '1-3', '1,3,5', '1 3 5', or Enter for all):\nSelection: ",
		},
		{
			name:    "number out of range",
			videos:  []models.Video{{Title: "Video1"}},
			all:     false,
			input:   "2\n",
			want:    nil,
			wantErr: true,
			err:     errNumberOutOfRange,
			wantPrompt: "\nAvailable videos:\n1. Video1\n\n" +
				"Select videos (e.g., '1-3', '1,3,5', '1 3 5', or Enter for all):\nSelection: ",
		},
		{
			name:    "invalid range format",
			videos:  []models.Video{{Title: "Video1"}, {Title: "Video2"}},
			all:     false,
			input:   "1-2-3\n",
			want:    nil,
			wantErr: true,
			err:     errInvalidRangeFormat,
			wantPrompt: "\nAvailable videos:\n1. Video1\n2. Video2\n\n" +
				"Select videos (e.g., '1-3', '1,3,5', '1 3 5', or Enter for all):\nSelection: ",
		},
		{
			name:    "invalid start number in range",
			videos:  []models.Video{{Title: "Video1"}, {Title: "Video2"}},
			all:     false,
			input:   "x-2\n",
			want:    nil,
			wantErr: true,
			err:     errInvalidStartNumber,
			wantPrompt: "\nAvailable videos:\n1. Video1\n2. Video2\n\n" +
				"Select videos (e.g., '1-3', '1,3,5', '1 3 5', or Enter for all):\nSelection: ",
		},
		{
			name:    "invalid end number in range",
			videos:  []models.Video{{Title: "Video1"}, {Title: "Video2"}},
			all:     false,
			input:   "1-y\n",
			want:    nil,
			wantErr: true,
			err:     errInvalidEndNumber,
			wantPrompt: "\nAvailable videos:\n1. Video1\n2. Video2\n\n" +
				"Select videos (e.g., '1-3', '1,3,5', '1 3 5', or Enter for all):\nSelection: ",
		},
		{
			name:    "range out of bounds",
			videos:  []models.Video{{Title: "Video1"}, {Title: "Video2"}},
			all:     false,
			input:   "1-3\n",
			want:    nil,
			wantErr: true,
			err:     errInvalidRange,
			wantPrompt: "\nAvailable videos:\n1. Video1\n2. Video2\n\n" +
				"Select videos (e.g., '1-3', '1,3,5', '1 3 5', or Enter for all):\nSelection: ",
		},
		{
			name:    "start greater than end in range",
			videos:  []models.Video{{Title: "Video1"}, {Title: "Video2"}},
			all:     false,
			input:   "2-1\n",
			want:    nil,
			wantErr: true,
			err:     errInvalidRange,
			wantPrompt: "\nAvailable videos:\n1. Video1\n2. Video2\n\n" +
				"Select videos (e.g., '1-3', '1,3,5', '1 3 5', or Enter for all):\nSelection: ",
		},
		{
			name:    "no valid selections",
			videos:  []models.Video{{Title: "Video1"}},
			all:     false,
			input:   ",,,\n",
			want:    nil,
			wantErr: true,
			err:     errNoValidSelectionsFound,
			wantPrompt: "\nAvailable videos:\n1. Video1\n\n" +
				"Select videos (e.g., '1-3', '1,3,5', '1 3 5', or Enter for all):\nSelection: ",
		},
		{
			name:       "empty video list",
			videos:     []models.Video{},
			all:        false,
			input:      "",
			want:       []int{},
			wantErr:    false,
			wantPrompt: "",
		},
		{
			name:    "duplicate selections",
			videos:  []models.Video{{Title: "Video1"}, {Title: "Video2"}},
			all:     false,
			input:   "1,1,1-2,2\n",
			want:    []int{0, 1},
			wantErr: false,
			wantPrompt: "\nAvailable videos:\n1. Video1\n2. Video2\n\n" +
				"Select videos (e.g., '1-3', '1,3,5', '1 3 5', or Enter for all):\nSelection: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			result, err := SelectVideos(tt.videos, tt.all)

			w.Close()

			output := make([]byte, 1000)
			n, _ := r.Read(output)
			capturedOutput := string(output[:n])

			if !equalIntSlices(result, tt.want) {
				t.Errorf("SelectVideos() = %v, want %v", result, tt.want)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("SelectVideos() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && !errors.Is(err, tt.err) {
				t.Errorf("SelectVideos() error = %v, want %v", err, tt.err)
			}

			if capturedOutput != tt.wantPrompt {
				t.Errorf("SelectVideos() prompt = %q, want %q", capturedOutput, tt.wantPrompt)
			}
		})
	}
}

// equalIntSlices compares two int slices for equality.
func equalIntSlices(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
