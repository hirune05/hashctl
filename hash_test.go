package main

import (
	"strings"
	"testing"
)

func TestCalculateHash(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
		contents string
		wantSize int64
		wantHash string
	}{
		{
			name:     "text file",
			filename: "hello.txt",
			contents: "hello\n",
			wantSize: 6,
			wantHash: "5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03",
		},
		{
			name:     "empty file",
			filename: "empty.txt",
			contents: "",
			wantSize: 0,
			wantHash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := calculateHash(tt.filename, strings.NewReader(tt.contents))
			if err != nil {
				t.Fatalf("calculateHash() error = %v", err)
			}
			if got.Filename != tt.filename {
				t.Errorf("Filename = %q, want %q", got.Filename, tt.filename)
			}
			if got.Size != tt.wantSize {
				t.Errorf("Size = %d, want %d", got.Size, tt.wantSize)
			}
			if got.SHA256 != tt.wantHash {
				t.Errorf("SHA256 = %q, want %q", got.SHA256, tt.wantHash)
			}
		})
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errReadFailed
}

var errReadFailed = &readError{}

type readError struct{}

func (*readError) Error() string { return "read failed" }

func TestCalculateHashPropagatesReadError(t *testing.T) {
	t.Parallel()

	_, err := calculateHash("broken.txt", failingReader{})
	if err == nil {
		t.Fatal("calculateHash() error = nil, want a read error")
	}
}
