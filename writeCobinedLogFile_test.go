package main

import (
	"os"
	"path/filepath"
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
