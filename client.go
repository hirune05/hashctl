package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func runRemote(ctx context.Context, serverURL, path string, stdout io.Writer) error {
	client := &http.Client{Timeout: 30 * time.Second}
	return runRemoteWithClient(ctx, client, serverURL, path, stdout)
}

func runRemoteWithClient(ctx context.Context, client *http.Client, serverURL, path string, stdout io.Writer) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	pipeReader, pipeWriter := io.Pipe()
	multipartWriter := multipart.NewWriter(pipeWriter)
	contentType := multipartWriter.FormDataContentType()

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(serverURL, "/")+"/hash",
		pipeReader,
	)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("Content-Type", contentType)

	go func() {
		part, err := multipartWriter.CreateFormFile("file", filepath.Base(path))
		if err == nil {
			_, err = io.Copy(part, file)
		}
		if closeErr := multipartWriter.Close(); err == nil {
			err = closeErr
		}
		pipeWriter.CloseWithError(err)
	}()

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
		return fmt.Errorf("server returned %s: %s", response.Status, strings.TrimSpace(string(message)))
	}

	var result HashResult
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if err := json.NewEncoder(stdout).Encode(result); err != nil {
		return fmt.Errorf("encode result: %w", err)
	}
	return nil
}
