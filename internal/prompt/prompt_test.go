package prompt

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestInput(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
		input  string
		want   string
	}{
		{
			name:   "basic input",
			prompt: "Enter value: ",
			input:  "test-value\n",
			want:   "test-value",
		},
		{
			name:   "empty input",
			prompt: "Enter value: ",
			input:  "\n",
			want:   "",
		},
		{
			name:   "input with leading/trailing spaces",
			prompt: "Enter value: ",
			input:  "  spaced-value  \n",
			want:   "spaced-value",
		},
		{
			name:   "multiline input stops at first newline",
			prompt: "Enter value: ",
			input:  "first-line\nsecond-line\n",
			want:   "first-line",
		},
		{
			name:   "input with tabs",
			prompt: "Enter value: ",
			input:  "\tvalue-with-tabs\t\n",
			want:   "value-with-tabs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file for stdin simulation
			tmpFile, err := os.CreateTemp("", "test-input")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			// Write test input
			if _, err := tmpFile.WriteString(tt.input); err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			tmpFile.Seek(0, 0)

			// Redirect stdin
			oldStdin := os.Stdin
			os.Stdin = tmpFile
			defer func() { os.Stdin = oldStdin }()

			// Capture stdout to verify prompt is printed
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			defer func() { os.Stdout = oldStdout }()

			// Test the function
			result := Input(tt.prompt)

			// Close writer and read captured output
			w.Close()
			output := make([]byte, 1000)
			n, _ := r.Read(output)
			capturedOutput := string(output[:n])

			if result != tt.want {
				t.Errorf("Input() = %v, want %v", result, tt.want)
			}

			if !strings.Contains(capturedOutput, tt.prompt) {
				t.Errorf(
					"Input() did not print prompt. Got: %v, expected to contain: %v",
					capturedOutput,
					tt.prompt,
				)
			}
		})
	}
}

func TestConfirm(t *testing.T) {
	tests := []struct {
		name   string
		format string
		args   []interface{}
		input  string
		want   bool
	}{
		{name: "yes lowercase", format: "Continue?", input: "y\n", want: true},
		{name: "yes uppercase", format: "Continue?", input: "Y\n", want: true},
		{name: "yes full word lowercase", format: "Continue?", input: "yes\n", want: true},
		{name: "yes full word uppercase", format: "Continue?", input: "YES\n", want: true},
		{name: "yes with spaces", format: "Continue?", input: "  y  \n", want: true},
		{name: "no lowercase", format: "Continue?", input: "n\n", want: false},
		{name: "no uppercase", format: "Continue?", input: "N\n", want: false},
		{name: "no full word", format: "Continue?", input: "no\n", want: false},
		{name: "empty input defaults to no", format: "Continue?", input: "\n", want: false},
		{name: "invalid input defaults to no", format: "Continue?", input: "💀\n", want: false},
		{
			name:   "format with single argument",
			format: "Delete file %s?",
			args:   []any{"test.txt"},
			input:  "y\n",
			want:   true,
		},
		{
			name:   "format with multiple arguments",
			format: "Delete %d files from %s?",
			args:   []any{5, "/tmp"},
			input:  "n\n",
			want:   false,
		},
		{
			name:   "format with no arguments but percent sign",
			format: "Continue with 100% certainty?",
			input:  "yes\n",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file for stdin simulation
			tmpFile, err := os.CreateTemp("", "test-input")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			// Write test input
			if _, err := tmpFile.WriteString(tt.input); err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			tmpFile.Seek(0, 0)

			// Redirect stdin
			oldStdin := os.Stdin
			os.Stdin = tmpFile
			defer func() { os.Stdin = oldStdin }()

			// Capture stdout to verify prompt is printed
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			defer func() { os.Stdout = oldStdout }()

			// Test the function
			var result bool
			if tt.args != nil {
				result = Confirm(tt.format, tt.args...)
			} else {
				result = Confirm("%s", tt.format)
			}

			// Close writer and read captured output
			w.Close()
			output := make([]byte, 1000)
			n, _ := r.Read(output)
			capturedOutput := string(output[:n])

			if result != tt.want {
				t.Errorf("Confirm() = %v, want %v", result, tt.want)
			}

			// Build expected prompt
			var expectedBase string
			if tt.args != nil {
				expectedBase = fmt.Sprintf(tt.format, tt.args...)
			} else {
				expectedBase = tt.format
			}
			expectedPrompt := expectedBase + " (y/N): "

			if !strings.Contains(capturedOutput, expectedPrompt) {
				t.Errorf(
					"Confirm() did not print expected prompt. Got: %v, expected to contain: %v",
					capturedOutput,
					expectedPrompt,
				)
			}
		})
	}
}

func TestConfirmPromptFormat(t *testing.T) {
	// Test that Confirm appends the correct suffix to the prompt
	tmpFile, err := os.CreateTemp("", "test-input")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("n\n")
	tmpFile.Seek(0, 0)

	oldStdin := os.Stdin
	os.Stdin = tmpFile
	defer func() { os.Stdin = oldStdin }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	Confirm("Test prompt")

	w.Close()
	output := make([]byte, 1000)
	n, _ := r.Read(output)
	capturedOutput := string(output[:n])

	expectedPrompt := "Test prompt (y/n): "
	if capturedOutput != expectedPrompt {
		t.Errorf("Confirm() prompt format = %v, want %v", capturedOutput, expectedPrompt)
	}
}

func TestInputEmptyPrompt(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-input")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("test\n")
	tmpFile.Seek(0, 0)

	oldStdin := os.Stdin
	os.Stdin = tmpFile
	defer func() { os.Stdin = oldStdin }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	result := Input("")

	w.Close()
	output := make([]byte, 1000)
	n, _ := r.Read(output)
	capturedOutput := string(output[:n])

	if result != "test" {
		t.Errorf("Input() = %v, want test", result)
	}

	if capturedOutput != "" {
		t.Errorf("Input() with empty prompt should print nothing, got: %v", capturedOutput)
	}
}
