package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
)

func newHandler(maxUploadBytes int64) http.Handler {
	mux := http.NewServeMux()

	handleOK := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	}

	mux.HandleFunc("GET /healthz", handleOK)
	mux.HandleFunc("GET /readyz", handleOK)
	mux.HandleFunc("POST /hash", handleHash(maxUploadBytes))

	return mux
}

func handleHash(maxUploadBytes int64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Allow some room for multipart headers while still bounding the full body.
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes+(1<<20))

		multipartReader, err := r.MultipartReader()
		if err != nil {
			http.Error(w, "request must be multipart/form-data", http.StatusBadRequest)
			return
		}

		for {
			part, err := multipartReader.NextPart()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				http.Error(w, "read multipart request", http.StatusBadRequest)
				return
			}

			if part.FormName() != "file" || part.FileName() == "" {
				part.Close()
				continue
			}

			result, err := calculateHash(
				filepath.Base(part.FileName()),
				io.LimitReader(part, maxUploadBytes+1),
			)
			part.Close()
			if err != nil {
				http.Error(w, "calculate hash", http.StatusInternalServerError)
				return
			}
			if result.Size > maxUploadBytes {
				http.Error(w, "file exceeds upload limit", http.StatusRequestEntityTooLarge)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(result); err != nil {
				return
			}
			return
		}

		http.Error(w, "multipart field file is required", http.StatusBadRequest)
	}
}
