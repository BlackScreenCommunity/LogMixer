package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestWriteCombinedLogFile(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test_output.log")

	blocks := []LogBlock{
		{Text: "2025-01-01 10:00:00,000 First log line\n"},
		{Text: "2025-01-01 10:01:00,000 Second log line\n"},
	}

	err := writeCombinedLogFile(&outputPath, &blocks)
	if err != nil {
		t.Fatalf("writeCombinedLogFile returned error: %v", err)
	}
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	expected := "2025-01-01 10:00:00,000 First log line\n2025-01-01 10:01:00,000 Second log line\n"

	if string(content) != expected {
		t.Errorf("unexpected file content.\nGot:\n%s\nExpected:\n%s", content, expected)
	}
}

func TestGetFilters_FromFile(t *testing.T) {

	yamlContent := `
contains:
  - Session started
  - Heartbeat OK
`
	tmpFile, err := os.CreateTemp("", "filters-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	expected := FilterConfig{
		Exclude: []string{"Session started", "Heartbeat OK"},
	}

	result := getFilters(tmpFile.Name())

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected: %+v, got: %+v", expected, result)
	}
}

func TestGetIsBlockNeedsToFilter(t *testing.T) {
	type args struct {
		currentBlock strings.Builder
		filters      FilterConfig
	}
	tests := []struct {
		name     string
		args     args
		expected bool
	}{
		{
			name: "Block contains one of the filters",
			args: args{
				currentBlock: func() strings.Builder {
					var b strings.Builder
					b.WriteString("Session started: user123")
					return b
				}(),
				filters: FilterConfig{Exclude: []string{"Session started", "Heartbeat OK"}},
			},
			expected: true,
		},
		{
			name: "Block contains none of the filters",
			args: args{
				currentBlock: func() strings.Builder {
					var b strings.Builder
					b.WriteString("Some other message")
					return b
				}(),
				filters: FilterConfig{Exclude: []string{"Session started", "Heartbeat OK"}},
			},
			expected: false,
		},
		{
			name: "Block contains filter as substring",
			args: args{
				currentBlock: func() strings.Builder {
					var b strings.Builder
					b.WriteString("xxxHeartbeat OKyyy")
					return b
				}(),
				filters: FilterConfig{Exclude: []string{"Session started", "Heartbeat OK"}},
			},
			expected: true,
		},
		{
			name: "No filters provided",
			args: args{
				currentBlock: func() strings.Builder {
					var b strings.Builder
					b.WriteString("Session started: user123")
					return b
				}(),
				filters: FilterConfig{Exclude: []string{}},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getIsBlockNeedsToFilter(tt.args.currentBlock, tt.args.filters)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
