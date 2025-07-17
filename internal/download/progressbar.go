package media

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/term"
)

const (
	// Progress bar display constants.
	percentageMultiplier = 100
	progressBarLength    = 50
	bytesPerKB           = 1024
	bytesPerMB           = 1024 * 1024
	bitsPerByte          = 8
)

const (
	// Terminal display constants.
	defaultTerminalWidth   = 80
	reservedSpace          = 40
	terminalWidthPadding   = 30
	minBarLength           = 10
	minFilenameLength      = 10
	truncationSuffix       = "..."
	truncationSuffixLength = 3
	stdinFileDescriptor    = 0
)

const (
	// ANSI escape codes.
	clearLine = "\r\x1b[2K"
)

// ShowProgress displays a progress bar for downloading.
func ShowProgress(
	written, total int64,
	filename string,
	currentItem, totalItems int,
	startTime time.Time,
) {
	termWidth := getTerminalWidth()
	percent := float64(written) / float64(total) * percentageMultiplier
	barLength := calculateBarLength(termWidth)
	bar := renderProgressBar(percent, barLength)
	speed := calculateDownloadSpeed(written, startTime)
	writtenMB, totalMB := formatFileSize(written, total)
	truncatedFilename := truncateFilename(filename, termWidth, barLength)

	progress := formatProgressMessage(
		currentItem,
		totalItems,
		truncatedFilename,
		bar,
		writtenMB,
		totalMB,
		speed,
	)
	progress = truncateProgressMessage(progress, termWidth)

	fmt.Printf("%s%s", clearLine, progress)
}

// getTerminalWidth returns the current terminal width or default if unavailable.
func getTerminalWidth() int {
	if term.IsTerminal(stdinFileDescriptor) {
		width, _, err := term.GetSize(stdinFileDescriptor)
		if err == nil {
			return width
		}
	}

	return defaultTerminalWidth
}

// calculateBarLength determines the appropriate progress bar length based on terminal width.
func calculateBarLength(termWidth int) int {
	barLength := progressBarLength
	barLength = min(barLength, termWidth-terminalWidthPadding)
	barLength = max(barLength, minBarLength)

	return barLength
}

// renderProgressBar creates the visual progress bar string.
func renderProgressBar(percent float64, barLength int) string {
	filled := int(float64(barLength) * percent / percentageMultiplier)

	return strings.Repeat("#", filled) + strings.Repeat("-", barLength-filled)
}

// calculateDownloadSpeed computes the download speed in Mb/s.
func calculateDownloadSpeed(written int64, startTime time.Time) float64 {
	elapsed := time.Since(startTime).Seconds()
	if elapsed > 0 {
		return float64(written) / elapsed / (bytesPerMB / bitsPerByte) // Mb/s
	}

	return 0
}

// formatFileSize converts bytes to MB for display.
func formatFileSize(written, total int64) (float64, float64) {
	writtenMB := float64(written) / bytesPerMB
	totalMB := float64(total) / bytesPerMB

	return writtenMB, totalMB
}

// truncateFilename shortens the filename if it's too long for the terminal.
func truncateFilename(filename string, termWidth, barLength int) string {
	maxFilenameLength := termWidth - barLength - reservedSpace
	maxFilenameLength = max(maxFilenameLength, minFilenameLength)

	if len(filename) > maxFilenameLength {
		return filename[:maxFilenameLength-truncationSuffixLength] + truncationSuffix
	}

	return filename
}

// formatProgressMessage creates the complete progress message string.
func formatProgressMessage(
	currentItem, totalItems int,
	filename, bar string,
	writtenMB, totalMB, speed float64,
) string {
	return fmt.Sprintf(
		"[%d/%d] Downloading: %s [%s] [%.0fMB/%.0fMB] (%.0f Mb/s)",
		currentItem,
		totalItems,
		filename,
		bar,
		writtenMB,
		totalMB,
		speed,
	)
}

// truncateProgressMessage ensures the progress message fits within terminal width.
func truncateProgressMessage(progress string, termWidth int) string {
	if len(progress) > termWidth {
		maxLength := max(0, termWidth-truncationSuffixLength)

		return progress[:maxLength] + truncationSuffix
	}

	return progress
}
