package ui

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"switch-tube-downloader/internal/models"
)

const (
	rangePartsCount = 2
)

var (
	errInvalidRange           = errors.New("invalid range")
	errInvalidNumber          = errors.New("invalid number")
	errInvalidEndNumber       = errors.New("invalid end number")
	errNumberOutOfRange       = errors.New("number out of range")
	errInvalidRangeFormat     = errors.New("invalid range format")
	errInvalidStartNumber     = errors.New("invalid start number")
	errNoValidSelectionsFound = errors.New("no valid selections found")
)

// SelectVideos displays the video list and handles user selection.
func SelectVideos(videos []models.Video, all bool) ([]int, error) {
	// If --all flag is used, select all videos
	if all {
		indices := make([]int, len(videos))
		for i := range indices {
			indices[i] = i
		}

		return indices, nil
	}

	// Display video list
	fmt.Println("\nAvailable videos:")

	for i, video := range videos {
		fmt.Printf("%d. %s\n", i+1, video.Title)
	}

	fmt.Println("\nSelect videos to download:")
	fmt.Println("Examples: '1-12', '1,3,5', '1 3 5', or press Enter for all")

	input := Input("Selection: ")
	input = strings.TrimSpace(input)

	// Empty input means select all
	if input == "" {
		indices := make([]int, len(videos))
		for i := range indices {
			indices[i] = i
		}

		return indices, nil
	}

	return ParseSelection(input, len(videos))
}

// ParseSelection parses user input and returns selected video indices.
func ParseSelection(input string, maxVideos int) ([]int, error) {
	var indices []int

	seen := make(map[int]bool)

	// Split by comma, space, or both
	parts := strings.FieldsFunc(input, func(r rune) bool {
		return r == ',' || r == ' '
	})

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Handle range (e.g., "1-5")
		if strings.Contains(part, "-") {
			var err error

			indices, err = handleRangePart(part, maxVideos, indices, seen)
			if err != nil {
				return nil, err
			}
		} else {
			var err error

			indices, err = handleSinglePart(part, maxVideos, indices, seen)
			if err != nil {
				return nil, err
			}
		}
	}

	if len(indices) == 0 {
		return nil, fmt.Errorf("%w", errNoValidSelectionsFound)
	}

	// Sort indices to maintain order
	sort.Ints(indices)

	return indices, nil
}

// handleRangePart processes a range selection like "1-5".
func handleRangePart(part string, maxVideos int, indices []int, seen map[int]bool) ([]int, error) {
	rangeParts := strings.Split(part, "-")
	if len(rangeParts) != rangePartsCount {
		return nil, fmt.Errorf("%w: %s", errInvalidRangeFormat, part)
	}

	start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errInvalidStartNumber, rangeParts[0])
	}

	end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errInvalidEndNumber, rangeParts[1])
	}

	if start < 1 || end > maxVideos || start > end {
		return nil, fmt.Errorf("%w: %d-%d (must be 1-%d)", errInvalidRange, start, end, maxVideos)
	}

	for i := start; i <= end; i++ {
		index := i - 1
		if !seen[index] {
			indices = append(indices, index)
			seen[index] = true
		}
	}

	return indices, nil
}

// handleSinglePart processes a single number selection.
func handleSinglePart(part string, maxVideos int, indices []int, seen map[int]bool) ([]int, error) {
	num, err := strconv.Atoi(part)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errInvalidNumber, part)
	}

	if num < 1 || num > maxVideos {
		return nil, fmt.Errorf("%w: %d (must be 1-%d)", errNumberOutOfRange, num, maxVideos)
	}

	index := num - 1
	if !seen[index] {
		indices = append(indices, index)
		seen[index] = true
	}

	return indices, nil
}
