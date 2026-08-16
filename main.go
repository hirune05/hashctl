package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

const defaultMaxUploadBytes int64 = 10 << 20 // 10 MiB

func run(ctx context.Context, args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: hashctl <local|remote|serve> [options]")
	}

	switch args[0] {
	case "local":
		if len(args) != 2 {
			return errors.New("usage: hashctl local <FILE>")
		}
		return runLocal(args[1], stdout)

	case "remote":
		flags := flag.NewFlagSet("remote", flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		serverURL := flags.String("server", "http://localhost:8080", "hashctl server URL")
		if err := flags.Parse(args[1:]); err != nil {
			return fmt.Errorf("parse remote options: %w", err)
		}
		if flags.NArg() != 1 {
			return errors.New("usage: hashctl remote --server <URL> <FILE>")
		}
		return runRemote(ctx, *serverURL, flags.Arg(0), stdout)

	case "serve":
		flags := flag.NewFlagSet("serve", flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		listen := flags.String("listen", ":8080", "HTTP listen address")
		if err := flags.Parse(args[1:]); err != nil {
			return fmt.Errorf("parse serve options: %w", err)
		}
		if flags.NArg() != 0 {
			return errors.New("usage: hashctl serve [--listen <ADDRESS>]")
		}
		maxUploadBytes, err := maxUploadBytesFromEnv()
		if err != nil {
			return err
		}
		return runServer(ctx, *listen, maxUploadBytes, stdout)

	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func runLocal(path string, stdout io.Writer) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	result, err := calculateHash(filepath.Base(path), file)
	if err != nil {
		return fmt.Errorf("calculate hash: %w", err)
	}
	if err := json.NewEncoder(stdout).Encode(result); err != nil {
		return fmt.Errorf("encode result: %w", err)
	}
	return nil
}

func maxUploadBytesFromEnv() (int64, error) {
	value := os.Getenv("MAX_UPLOAD_BYTES")
	if value == "" {
		return defaultMaxUploadBytes, nil
	}

	maxUploadBytes, err := strconv.ParseInt(value, 10, 64)
	if err != nil || maxUploadBytes <= 0 {
		return 0, fmt.Errorf("MAX_UPLOAD_BYTES must be a positive integer: %q", value)
	}
	return maxUploadBytes, nil
}

func runServer(ctx context.Context, listen string, maxUploadBytes int64, stdout io.Writer) error {
	server := &http.Server{
		Addr:              listen,
		Handler:           newHandler(maxUploadBytes),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		fmt.Fprintf(stdout, "listening on %s\n", listen)
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}
		return nil
	}
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
