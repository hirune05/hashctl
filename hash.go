package main

import (
	"errors"
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
	return HashResult{}, errors.New("TODO: implement calculateHash")
}
