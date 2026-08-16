package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
)

// HashResult is shared by the CLI and the HTTP API.
type HashResult struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	SHA256   string `json:"sha256"`
}

// calculateHash reads r as a stream and returns its SHA-256 digest.
//
// Exercise checkpoint 1: implement this function without loading the whole
// input into memory. Empty input is valid and must return the SHA-256 digest
// of an empty byte sequence.
func calculateHash(filename string, r io.Reader) (HashResult, error) {
	hasher := sha256.New()
	size, err := io.Copy(hasher, r)
	if err != nil {
		return HashResult{}, err
	}
	hashString := hex.EncodeToString(hasher.Sum(nil))

	return HashResult{
		Filename: filename,
		Size:     size,
		SHA256:   hashString,
	}, nil
}
