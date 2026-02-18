package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func defaultSessionPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	dir := filepath.Join(home, ".ajent", "sessions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating sessions directory: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	base := filepath.Base(cwd)

	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generating random bytes: %w", err)
	}
	hash := hex.EncodeToString(b[:])

	ts := time.Now().Format("20060102-150405")
	name := fmt.Sprintf("%s-%s-%s.hjl", base, ts, hash)
	return filepath.Join(dir, name), nil
}
