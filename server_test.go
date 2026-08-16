package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpoints(t *testing.T) {
	t.Parallel()

	for _, path := range []string{"/healthz", "/readyz"} {
		path := path
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(http.MethodGet, path, nil)
			response := httptest.NewRecorder()

			newHandler(1024).ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", response.Code, http.StatusOK)
			}
			if response.Body.String() != "ok\n" {
				t.Errorf("body = %q, want %q", response.Body.String(), "ok\n")
			}
		})
	}
}

func TestHashEndpoint(t *testing.T) {
	t.Parallel()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("file", "hello.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("hello\n")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/hash", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()

	newHandler(1024).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}

	var got HashResult
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if got.Filename != "hello.txt" || got.Size != 6 {
		t.Errorf("result = %+v, want filename hello.txt and size 6", got)
	}
	const wantHash = "5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03"
	if got.SHA256 != wantHash {
		t.Errorf("SHA256 = %q, want %q", got.SHA256, wantHash)
	}
}

func TestHashEndpointRejectsInvalidUploads(t *testing.T) {
	t.Parallel()

	t.Run("missing file", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		if err := writer.WriteField("other", "value"); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}

		request := httptest.NewRequest(http.MethodPost, "/hash", &body)
		request.Header.Set("Content-Type", writer.FormDataContentType())
		response := httptest.NewRecorder()
		newHandler(1024).ServeHTTP(response, request)

		if response.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", response.Code, http.StatusBadRequest)
		}
	})

	t.Run("over upload limit", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		file, err := writer.CreateFormFile("file", "large.txt")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write([]byte("123456")); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}

		request := httptest.NewRequest(http.MethodPost, "/hash", &body)
		request.Header.Set("Content-Type", writer.FormDataContentType())
		response := httptest.NewRecorder()
		newHandler(5).ServeHTTP(response, request)

		if response.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("status = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
		}
	})
}
