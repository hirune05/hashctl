package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func run(args []string, stdout io.Writer) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: hashctl local <FILE>")
	}

	switch args[0] {
	case "local":
		return runLocal(args[1], stdout)
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

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
