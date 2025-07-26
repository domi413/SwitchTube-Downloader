package download

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractIDAndType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantID   string
		wantType mediaType
		wantErr  bool
		err      error
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
			err:      errInvalidURL,
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

			if tt.wantErr && !errors.Is(err, tt.err) {
				t.Errorf("extractIDAndType() error = %v, want %v", err, tt.err)
			}
		})
	}
}

func TestMakeRequest(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		token      string
		server     func(*http.Request) *http.Response
		wantStatus int
		wantErr    bool
		err        string
	}{
		{
			name:  "successful request",
			url:   "http://example.com",
			token: "mock-token",
			server: func(req *http.Request) *http.Response {
				if req.Header.Get(headerAuthorization) != "Token mock-token" {
					return &http.Response{StatusCode: http.StatusUnauthorized}
				}
				return &http.Response{StatusCode: http.StatusOK}
			},
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name:  "invalid URL",
			url:   "://invalid-url",
			token: "mock-token",
			server: func(req *http.Request) *http.Response {
				return nil // Won't be called
			},
			wantStatus: 0,
			wantErr:    true,
			err:        "failed to create request",
		},
		{
			name:  "server error",
			url:   "http://example.com",
			token: "mock-token",
			server: func(req *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError}
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start test server
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					resp := tt.server(r)
					if resp == nil {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}

					w.WriteHeader(resp.StatusCode)
				}),
			)
			defer server.Close()

			// Use server URL if not testing invalid URL
			url := tt.url
			if !tt.wantErr {
				url = server.URL
			}

			// Make request
			resp, err := makeRequest(url, tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("makeRequest() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && !strings.Contains(err.Error(), tt.err) {
				t.Errorf("makeRequest() error = %v, want contains %q", err, tt.err)
			}

			if resp != nil {
				defer resp.Body.Close()

				if resp.StatusCode != tt.wantStatus {
					t.Errorf("makeRequest() status = %d, want %d", resp.StatusCode, tt.wantStatus)
				}
			}
		})
	}
}
