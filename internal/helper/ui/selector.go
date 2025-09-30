package ui

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"switchtube-downloader/internal/models"
)

const rangePartsCount = 2

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
	if all || len(videos) == 0 {
		indices := make([]int, len(videos))
		for i := range indices {
			indices[i] = i
		}

		return indices, nil
	}

	fmt.Println("\nAvailable videos:")

	for i, video := range videos {
		fmt.Printf("%d. %s\n", i+1, video.Title)
	}

	fmt.Println("\nSelect videos (e.g., '1-3', '1,3,5', '1 3 5', or Enter for all):")

	input := strings.TrimSpace(Input("Selection: "))
	if input == "" {
		// If input is empty, select all videos
		indices := make([]int, len(videos))
		for i := range indices {
			indices[i] = i
		}

		return indices, nil
	}

	return parseSelection(input, len(videos))
}

// parseSelection parses user input and returns selected video indices.
func parseSelection(input string, availableVideos int) ([]int, error) {
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

		var err error
		// Handle range (e.g., "1-5")
		if strings.Contains(part, "-") {
			indices, err = handleRangeSelection(part, availableVideos, indices, seen)
			if err != nil {
				return nil, err
			}
		} else {
			indices, err = handleSingleSelection(part, availableVideos, indices, seen)
			if err != nil {
				return nil, err
			}
		}
	}

	if len(indices) == 0 {
		return nil, fmt.Errorf("%w", errNoValidSelectionsFound)
	}

	sort.Ints(indices)

	return indices, nil
}

// handleRangeSelection processes a range selection like "1-5".
func handleRangeSelection(
	part string,
	availableVideos int,
	indices []int,
	seen map[int]bool,
) ([]int, error) {
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

	if start < 1 || end > availableVideos || start > end {
		return nil, fmt.Errorf(
			"%w: %d-%d (must be 1-%d)",
			errInvalidRange,
			start,
			end,
			availableVideos,
		)
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

// handleSingleSelection processes a single number selection.
func handleSingleSelection(
	part string,
	availableVideos int,
	indices []int,
	seen map[int]bool,
) ([]int, error) {
	num, err := strconv.Atoi(part)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errInvalidNumber, part)
	}

	if num < 1 || num > availableVideos {
		return nil, fmt.Errorf("%w: %d (must be 1-%d)", errNumberOutOfRange, num, availableVideos)
	}

	index := num - 1
	if !seen[index] {
		indices = append(indices, index)
		seen[index] = true
	}

	return indices, nil
}
