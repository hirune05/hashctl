package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRunLocal(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(path, []byte("hello\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	if err := run([]string{"local", path}, &stdout); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	var got HashResult
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, stdout.String())
	}

	if got.Filename != "hello.txt" {
		t.Errorf("Filename = %q, want %q", got.Filename, "hello.txt")
	}
	if got.Size != 6 {
		t.Errorf("Size = %d, want 6", got.Size)
	}
	const wantHash = "5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03"
	if got.SHA256 != wantHash {
		t.Errorf("SHA256 = %q, want %q", got.SHA256, wantHash)
	}
}

func TestRunRejectsInvalidArguments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "no command"},
		{name: "local without file", args: []string{"local"}},
		{name: "local with too many files", args: []string{"local", "a", "b"}},
		{name: "unknown command", args: []string{"unknown"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := run(tt.args, &bytes.Buffer{}); err == nil {
				t.Fatal("run() error = nil, want an argument error")
			}
		})
	}
}

func TestRunLocalRejectsMissingFile(t *testing.T) {
	t.Parallel()

	if err := run([]string{"local", filepath.Join(t.TempDir(), "missing.txt")}, &bytes.Buffer{}); err == nil {
		t.Fatal("run() error = nil, want a file-open error")
	}
}
