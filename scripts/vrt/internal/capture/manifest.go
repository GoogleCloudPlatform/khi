// Copyright 2026 Google LLC
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

package capture

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// ManifestVersion is the current schema version of VRT snapshot manifests.
	ManifestVersion = 1
	// ManifestFileName is the filename of the snapshot manifest stored at the root of a snapshot directory.
	ManifestFileName = "manifest.json"
	// ImagesDirName is the subdirectory storing captured PNG files inside a snapshot directory.
	ImagesDirName = "images"
)

var nonFingerprintCharRe = regexp.MustCompile(`[^a-z0-9._-]+`)

// SnapshotEntry holds metadata and capture status for a single visual snapshot.
type SnapshotEntry struct {
	// ID is the namespaced snapshot identifier (e.g., "story/header--default").
	ID string `json:"id"`
	// StoryID is the raw Storybook story ID when captured from Storybook.
	StoryID string `json:"storyId"`
	// Title is the Storybook hierarchy title.
	Title string `json:"title"`
	// Name is the human-readable variant name.
	Name string `json:"name"`
	// ImportPath is the source path of the story file.
	ImportPath string `json:"importPath"`
	// ImageRelPath is the slash-separated path of the PNG relative to the snapshot root.
	ImageRelPath string `json:"imageRelPath,omitempty"`
	// SHA256 is the lowercase hex SHA-256 digest of the captured PNG bytes.
	SHA256 string `json:"sha256,omitempty"`
	// Width is the captured PNG width in pixels.
	Width int `json:"width,omitempty"`
	// Height is the captured PNG height in pixels.
	Height int `json:"height,omitempty"`
	// DurationMs is the capture elapsed time in milliseconds.
	DurationMs int64 `json:"durationMs"`
	// Skipped reports whether the story opted out of VRT via parameters.vrt.skip.
	Skipped bool `json:"skipped,omitempty"`
	// DiffPixelRatio overrides the default diff tolerance for this story when positive.
	DiffPixelRatio float64 `json:"diffPixelRatio,omitempty"`
	// Error holds the failure description when capture or runtime verification failed.
	Error string `json:"error,omitempty"`
}

// Manifest describes a complete set of captured screenshots for a commit and builder environment.
type Manifest struct {
	// Version is the manifest schema version.
	Version int `json:"version"`
	// CommitSHA is the Git commit SHA that produced these snapshots.
	CommitSHA string `json:"commitSha,omitempty"`
	// BuilderFingerprint identifies the browser and OS environment used for rendering.
	BuilderFingerprint string `json:"builderFingerprint"`
	// CapturedAt is the UTC RFC3339 timestamp when capture finished.
	CapturedAt string `json:"capturedAt"`
	// ViewportWidth is the default viewport width in pixels.
	ViewportWidth int `json:"viewportWidth"`
	// ViewportHeight is the default viewport height in pixels.
	ViewportHeight int `json:"viewportHeight"`
	// Entries maps namespaced snapshot IDs (e.g., "story/<id>") to their capture metadata.
	Entries map[string]SnapshotEntry `json:"entries"`
}

// StorySnapshotID returns the namespaced snapshot key for a Storybook story ID.
func StorySnapshotID(storyID string) string {
	return "story/" + storyID
}

// ImageRelPathForID returns the relative PNG path within a snapshot directory for a snapshot ID.
func ImageRelPathForID(snapshotID string) string {
	return ImagesDirName + "/" + snapshotID + ".png"
}

// SanitizeFingerprint normalizes a Chromium product string and OS codename into a GCS-safe cache key segment.
func SanitizeFingerprint(browserVersion string, osCodename string) string {
	combined := strings.ToLower(strings.TrimSpace(browserVersion))
	if codename := strings.ToLower(strings.TrimSpace(osCodename)); codename != "" {
		combined = combined + "-" + codename
	}
	sanitized := nonFingerprintCharRe.ReplaceAllString(combined, "-")
	sanitized = strings.Trim(sanitized, "-")
	if sanitized == "" {
		return "unknown"
	}
	return sanitized
}

// WriteManifest writes manifest as formatted JSON to path, creating parent directories as needed.
func WriteManifest(path string, manifest Manifest) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create manifest directory: %w", err)
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write manifest %s: %w", path, err)
	}
	return nil
}

// ReadManifest reads and decodes a snapshot Manifest from path.
func ReadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("failed to read manifest %s: %w", path, err)
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("failed to parse manifest %s: %w", path, err)
	}
	if manifest.Entries == nil {
		manifest.Entries = map[string]SnapshotEntry{}
	}
	return manifest, nil
}
