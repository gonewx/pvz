package config

import (
	"os"
	"path/filepath"
	"runtime"
)

// getProjectRoot returns the project root directory.
// It uses runtime.Caller to find the source file location and navigates up to project root.
func getProjectRoot() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	// This file is at pkg/config/unit_config_test.go
	// Navigate up two directories to reach project root
	dir := filepath.Dir(filename)
	return filepath.Join(dir, "..", "..")
}

// checkSoundFileExists checks if a sound file exists relative to project root.
func checkSoundFileExists(path string) bool {
	projectRoot := getProjectRoot()
	fullPath := filepath.Join(projectRoot, path)
	_, err := os.Stat(fullPath)
	return err == nil
}
