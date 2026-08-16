package main

import (
	"fmt"
	"net/http"
)

func newHandler(maxUploadBytes int64) http.Handler {
	mux := http.NewServeMux()

	handleOK := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	}

	mux.HandleFunc("GET /healthz", handleOK)
	mux.HandleFunc("GET /readyz", handleOK)

	_ = maxUploadBytes // 次のPOST /hashで使用する

	return mux
}
