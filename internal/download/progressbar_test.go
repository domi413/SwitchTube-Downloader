package download

import (
	"strings"
	"testing"
	"time"
)

func TestCalculateBarLength(t *testing.T) {
	tests := []struct {
		name      string
		termWidth int
		want      int
	}{
		{
			name:      "normal terminal width",
			termWidth: 80,
			want:      progressBarLength, // Should use default length
		},
		{
			name:      "narrow terminal",
			termWidth: 50,
			want:      20, // 50 - 30 = 20
		},
		{
			name:      "very narrow terminal",
			termWidth: 30,
			want:      minBarLength, // Should use minimum
		},
		{
			name:      "extremely narrow terminal",
			termWidth: 20,
			want:      minBarLength, // Should still use minimum
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateBarLength(tt.termWidth)
			if got != tt.want {
				t.Errorf("calculateBarLength() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRenderProgressBar(t *testing.T) {
	tests := []struct {
		name      string
		percent   float64
		barLength int
		wantHash  int // number of # characters expected
		wantDash  int // number of - characters expected
	}{
		{
			name:      "0% progress",
			percent:   0,
			barLength: 10,
			wantHash:  0,
			wantDash:  10,
		},
		{
			name:      "50% progress",
			percent:   50,
			barLength: 10,
			wantHash:  5,
			wantDash:  5,
		},
		{
			name:      "100% progress",
			percent:   100,
			barLength: 10,
			wantHash:  10,
			wantDash:  0,
		},
		{
			name:      "25% progress with longer bar",
			percent:   25,
			barLength: 20,
			wantHash:  5,
			wantDash:  15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderProgressBar(tt.percent, tt.barLength)

			if len(got) != tt.barLength {
				t.Errorf("renderProgressBar() length = %v, want %v", len(got), tt.barLength)
			}

			hashCount := strings.Count(got, "#")
			dashCount := strings.Count(got, "-")

			if hashCount != tt.wantHash {
				t.Errorf("renderProgressBar() hash count = %v, want %v", hashCount, tt.wantHash)
			}

			if dashCount != tt.wantDash {
				t.Errorf("renderProgressBar() dash count = %v, want %v", dashCount, tt.wantDash)
			}

			// Check that all filled characters come before unfilled
			if tt.wantHash > 0 && tt.wantDash > 0 {
				expectedStart := strings.Repeat("#", tt.wantHash)
				if !strings.HasPrefix(got, expectedStart) {
					t.Errorf("renderProgressBar() should start with %s, got %s", expectedStart, got)
				}
			}
		})
	}
}

func TestCalculateDownloadSpeed(t *testing.T) {
	tests := []struct {
		name      string
		written   int64
		startTime time.Time
		wantSpeed float64 // approximate expected speed in Mb/s
		tolerance float64 // tolerance for floating point comparison
	}{
		{
			name:      "1 second download of 1MB",
			written:   1 * bytesPerMB,
			startTime: time.Now().Add(-1 * time.Second),
			wantSpeed: 8.0, // 1MB/s = 8 Mb/s
			tolerance: 1.0,
		},
		{
			name:      "2 seconds download of 1MB",
			written:   1 * bytesPerMB,
			startTime: time.Now().Add(-2 * time.Second),
			wantSpeed: 4.0, // 0.5MB/s = 4 Mb/s
			tolerance: 1.0,
		},
		{
			name:      "no time elapsed",
			written:   1 * bytesPerMB,
			startTime: time.Now().Add(1 * time.Second), // Future time = no elapsed time
			wantSpeed: 0.0,
			tolerance: 1.0,
		},
		{
			name:      "zero bytes written",
			written:   0,
			startTime: time.Now().Add(-1 * time.Second),
			wantSpeed: 0.0,
			tolerance: 0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateDownloadSpeed(tt.written, tt.startTime)

			if tt.wantSpeed == 0.0 {
				// For zero case, speed should be exactly 0 or very close
				if got > 1.0 {
					t.Errorf("calculateDownloadSpeed() should be near 0, got: %v", got)
				}
			} else {
				// For non-zero cases, check within tolerance
				diff := got - tt.wantSpeed
				if diff < 0 {
					diff = -diff
				}
				if diff > tt.tolerance {
					t.Errorf("calculateDownloadSpeed() = %v, want %v ± %v", got, tt.wantSpeed, tt.tolerance)
				}
			}

			// Speed should never be negative
			if got < 0 {
				t.Errorf("calculateDownloadSpeed() should not be negative, got: %v", got)
			}
		})
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		name        string
		written     int64
		total       int64
		wantWritten float64
		wantTotal   float64
	}{
		{
			name:        "1MB written of 2MB total",
			written:     1 * bytesPerMB,
			total:       2 * bytesPerMB,
			wantWritten: 1.0,
			wantTotal:   2.0,
		},
		{
			name:        "zero bytes",
			written:     0,
			total:       100 * bytesPerMB,
			wantWritten: 0.0,
			wantTotal:   100.0,
		},
		{
			name:        "fractional MB",
			written:     512 * bytesPerKB,  // 0.5 MB
			total:       1536 * bytesPerKB, // 1.5 MB
			wantWritten: 0.5,
			wantTotal:   1.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotWritten, gotTotal := formatFileSize(tt.written, tt.total)

			if gotWritten != tt.wantWritten {
				t.Errorf("formatFileSize() written = %v, want %v", gotWritten, tt.wantWritten)
			}

			if gotTotal != tt.wantTotal {
				t.Errorf("formatFileSize() total = %v, want %v", gotTotal, tt.wantTotal)
			}
		})
	}
}

func TestTruncateFilename(t *testing.T) {
	tests := []struct {
		name          string
		filename      string
		termWidth     int
		barLength     int
		wantTruncated bool
	}{
		{
			name:          "short filename",
			filename:      "test.mp4",
			termWidth:     80,
			barLength:     50,
			wantTruncated: false,
		},
		{
			name:          "long filename with normal terminal",
			filename:      "this_is_a_very_long_filename_that_should_be_truncated_because_it_exceeds_reasonable_length.mp4",
			termWidth:     80,
			barLength:     50,
			wantTruncated: true,
		},
		{
			name:          "moderate filename with narrow terminal",
			filename:      "moderately_long_filename.mp4",
			termWidth:     50,
			barLength:     20,
			wantTruncated: true,
		},
		{
			name:          "filename that fits exactly",
			filename:      "exact",
			termWidth:     60,
			barLength:     30,
			wantTruncated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateFilename(tt.filename, tt.termWidth, tt.barLength)

			if tt.wantTruncated {
				if !strings.HasSuffix(got, truncationSuffix) {
					t.Errorf(
						"truncateFilename() should end with '%s', got: %s",
						truncationSuffix,
						got,
					)
				}
				if len(got) >= len(tt.filename) {
					t.Errorf("truncateFilename() should be shorter than original")
				}
			} else {
				if got != tt.filename {
					t.Errorf("truncateFilename() should not modify filename, got: %s, want: %s", got, tt.filename)
				}
			}

			// Verify the result respects the calculated max length
			maxLength := tt.termWidth - tt.barLength - reservedSpace
			maxLength = max(maxLength, minFilenameLength)
			if len(got) > maxLength {
				t.Errorf(
					"truncateFilename() result length (%d) exceeds max length (%d)",
					len(got),
					maxLength,
				)
			}
		})
	}
}

func TestFormatProgressMessage(t *testing.T) {
	tests := []struct {
		name        string
		currentItem int
		totalItems  int
		filename    string
		bar         string
		writtenMB   float64
		totalMB     float64
		speed       float64
	}{
		{
			name:        "basic progress message",
			currentItem: 1,
			totalItems:  3,
			filename:    "test.mp4",
			bar:         "#####-----",
			writtenMB:   50.0,
			totalMB:     100.0,
			speed:       8.5,
		},
		{
			name:        "single item",
			currentItem: 1,
			totalItems:  1,
			filename:    "video.mkv",
			bar:         "##########",
			writtenMB:   200.0,
			totalMB:     200.0,
			speed:       12.3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatProgressMessage(
				tt.currentItem,
				tt.totalItems,
				tt.filename,
				tt.bar,
				tt.writtenMB,
				tt.totalMB,
				tt.speed,
			)

			// Check that the message contains expected components
			if !strings.Contains(got, tt.filename) {
				t.Errorf("formatProgressMessage() should contain filename '%s'", tt.filename)
			}

			if !strings.Contains(got, tt.bar) {
				t.Errorf("formatProgressMessage() should contain progress bar '%s'", tt.bar)
			}

			expectedPrefix := "[" + string(
				rune(tt.currentItem+'0'),
			) + "/" + string(
				rune(tt.totalItems+'0'),
			) + "]"
			if !strings.Contains(got, expectedPrefix) {
				t.Errorf("formatProgressMessage() should contain item counter '%s'", expectedPrefix)
			}

			// Check that it follows the expected format
			if !strings.Contains(got, "Downloading:") {
				t.Errorf("formatProgressMessage() should contain 'Downloading:'")
			}

			if !strings.Contains(got, "MB") {
				t.Errorf("formatProgressMessage() should contain file size in MB")
			}

			if !strings.Contains(got, "Mb/s") {
				t.Errorf("formatProgressMessage() should contain speed in Mb/s")
			}
		})
	}
}

func TestTruncateProgressMessage(t *testing.T) {
	tests := []struct {
		name      string
		progress  string
		termWidth int
		want      string
	}{
		{
			name:      "message fits",
			progress:  "Short message",
			termWidth: 50,
			want:      "Short message",
		},
		{
			name:      "message too long",
			progress:  "This is a very long progress message that exceeds terminal width",
			termWidth: 20,
			want:      "This is a very l...",
		},
		{
			name:      "exact fit",
			progress:  "Exactly twenty ch",
			termWidth: 17,
			want:      "Exactly twenty ch",
		},
		{
			name:      "very narrow terminal",
			progress:  "Long message",
			termWidth: 5,
			want:      "Lo...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateProgressMessage(tt.progress, tt.termWidth)

			if len(got) > tt.termWidth {
				t.Errorf(
					"truncateProgressMessage() result length (%d) exceeds terminal width (%d)",
					len(got),
					tt.termWidth,
				)
			}

			if len(tt.progress) <= tt.termWidth {
				// Should not be modified
				if got != tt.progress {
					t.Errorf(
						"truncateProgressMessage() should not modify message that fits, got: %s, want: %s",
						got,
						tt.progress,
					)
				}
			} else {
				// Should be truncated and end with suffix
				if !strings.HasSuffix(got, truncationSuffix) {
					t.Errorf("truncateProgressMessage() should end with '%s', got: %s", truncationSuffix, got)
				}
			}
		})
	}
}
