package media

import (
	"fmt"
	"strings"
	"time"
)

const (
	// Progress bar display constants.
	percentageMultiplier = 100
	progressBarLength    = 50
	bytesPerKB           = 1024
	bytesPerMB           = 1024 * 1024
	bitsPerByte          = 8
)

// ShowProgress displays a progress bar for downloading.
func ShowProgress(written, total int64, filename string, currentItem, totalItems int, startTime time.Time) {
	percent := float64(written) / float64(total) * percentageMultiplier
	barLength := progressBarLength
	filled := int(float64(barLength) * percent / percentageMultiplier)
	bar := strings.Repeat("#", filled) + strings.Repeat("-", barLength-filled)

	// Calculate speed
	elapsed := time.Since(startTime).Seconds()
	speed := float64(written) / elapsed / (bytesPerMB / bitsPerByte) // Mb/s

	// Format file sizes
	writtenMB := float64(written) / bytesPerMB
	totalMB := float64(total) / bytesPerMB

	fmt.Printf("\r[2K[%d/%d] Downloading: %s [%s] [%.0fMB/%.0fMB] (%.0f Mb/s)",
		currentItem, totalItems, filename, bar, writtenMB, totalMB, speed)
}
