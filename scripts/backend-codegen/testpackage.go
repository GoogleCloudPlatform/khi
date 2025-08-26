// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// TestPackage represents a Go package containing test files.
type TestPackage struct {
	// DirectoryPath is the file system path to the package directory.
	DirectoryPath string
	// PackageName is the Go package name parsed from a test file within the directory.
	PackageName string
}

// TestPackageFinder is responsible for finding Go packages that contain tests.
type TestPackageFinder struct {
	// RootDirectoryPath is the starting directory for the search.
	RootDirectoryPath string
}

// NewTestPackageFinder creates a new TestPackageFinder.
func NewTestPackageFinder(rootPath string) *TestPackageFinder {
	return &TestPackageFinder{RootDirectoryPath: rootPath}
}

// Find scans the RootDirectoryPath and returns a list of packages that contain test files.
func (f *TestPackageFinder) Find() ([]TestPackage, error) {
	var packages []TestPackage
	processedDirs := make(map[string]bool)

	err := filepath.Walk(f.RootDirectoryPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), "_test.go") {
			dir := filepath.Dir(path)
			if !processedDirs[dir] {
				packageName, err := extractPackageName(path)
				if err != nil {
					// Ignore files where package name cannot be extracted.
					// This can happen for empty or malformed files.
					return nil
				}

				packages = append(packages, TestPackage{
					DirectoryPath: dir,
					PackageName:   packageName,
				})
				processedDirs[dir] = true
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking directory %s: %w", f.RootDirectoryPath, err)
	}

	return packages, nil
}

// extractPackageName reads a Go file and extracts its package name.
func extractPackageName(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("could not open file %s: %w", filePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	re := regexp.MustCompile(`^package\s+([a-zA-Z0-9_]+)`)
	for scanner.Scan() {
		line := scanner.Text()
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			return matches[1], nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error scanning file %s: %w", filePath, err)
	}

	return "", fmt.Errorf("package declaration not found in %s", filePath)
}
