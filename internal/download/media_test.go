package download

import (
	"errors"
	"testing"
)

func TestExtractIDAndType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantID   string
		wantType mediaType
		wantErr  bool
		errType  error
	}{
		{
			name:     "video URL",
			input:    baseURL + videoPrefix + "123",
			wantID:   "123",
			wantType: videoType,
			wantErr:  false,
		},
		{
			name:     "channel URL",
			input:    baseURL + channelPrefix + "abc",
			wantID:   "abc",
			wantType: channelType,
			wantErr:  false,
		},
		{
			name:     "ID only (unknown type)",
			input:    "123",
			wantID:   "123",
			wantType: unknownType,
			wantErr:  false,
		},
		{
			name:     "invalid URL",
			input:    baseURL + "invalid/123",
			wantID:   "invalid/123",
			wantType: unknownType,
			wantErr:  true,
			errType:  errInvalidURL,
		},
		{
			name:     "input with spaces",
			input:    "  " + baseURL + videoPrefix + "123  ",
			wantID:   "123",
			wantType: videoType,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, downloadType, err := extractIDAndType(tt.input)

			if id != tt.wantID {
				t.Errorf("extractIDAndType() id = %q, want %q", id, tt.wantID)
			}

			if downloadType != tt.wantType {
				t.Errorf("extractIDAndType() type = %v, want %v", downloadType, tt.wantType)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("extractIDAndType() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && !errors.Is(err, tt.errType) {
				t.Errorf("extractIDAndType() error = %v, want %v", err, tt.errType)
			}
		})
	}
}
