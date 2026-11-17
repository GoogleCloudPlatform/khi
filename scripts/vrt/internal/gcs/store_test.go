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

package gcs

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"

	"github.com/GoogleCloudPlatform/khi/scripts/vrt/internal/capture"
)

type memoryObject struct {
	data         []byte
	contentType  string
	cacheControl string
}

type memoryObjectStore struct {
	mu       sync.Mutex
	objects  map[string]memoryObject
	putOrder []string
}

func newMemoryObjectStore() *memoryObjectStore {
	return &memoryObjectStore{
		objects: map[string]memoryObject{},
	}
}

func (m *memoryObjectStore) Get(_ context.Context, object string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	obj, ok := m.objects[object]
	if !ok {
		return nil, ErrNotFound
	}
	cp := make([]byte, len(obj.data))
	copy(cp, obj.data)
	return cp, nil
}

func (m *memoryObjectStore) Put(_ context.Context, object string, contentType string, cacheControl string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]byte, len(data))
	copy(cp, data)
	m.objects[object] = memoryObject{
		data:         cp,
		contentType:  contentType,
		cacheControl: cacheControl,
	}
	m.putOrder = append(m.putOrder, object)
	return nil
}

func (m *memoryObjectStore) Exists(_ context.Context, object string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.objects[object]
	return ok, nil
}

func TestParseAccessModeAndObjectURL(t *testing.T) {
	testCases := []struct {
		name       string
		rawMode    string
		objectPath string
		wantMode   AccessMode
		wantURL    string
		wantErr    bool
	}{
		{
			name:       "empty defaults to console mode",
			rawMode:    "",
			objectPath: "reports/abc123/run-1/index.html",
			wantMode:   AccessModeConsole,
			wantURL:    "https://storage.cloud.google.com/khi-vrt-artifacts/reports/abc123/run-1/index.html",
		},
		{
			name:       "public mode uses storage.googleapis.com",
			rawMode:    "public",
			objectPath: "reports/abc123/run-1/composite/story/header--default.png",
			wantMode:   AccessModePublic,
			wantURL:    "https://storage.googleapis.com/khi-vrt-artifacts/reports/abc123/run-1/composite/story/header--default.png",
		},
		{
			name:    "rejects unsupported access mode",
			rawMode: "private",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mode, err := ParseAccessMode(tc.rawMode)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ParseAccessMode(%q) error = %v, wantErr %v", tc.rawMode, err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if mode != tc.wantMode {
				t.Errorf("ParseAccessMode(%q) = %q, want %q", tc.rawMode, mode, tc.wantMode)
			}
			client := NewClient(newMemoryObjectStore(), "gs://khi-vrt-artifacts/", mode, 4)
			gotURL := client.ObjectURL(tc.objectPath)
			if gotURL != tc.wantURL {
				t.Errorf("ObjectURL(%q) = %q, want %q", tc.objectPath, gotURL, tc.wantURL)
			}
		})
	}
}

func TestSnapshotUploadAndDownloadRoundTrip(t *testing.T) {
	store := newMemoryObjectStore()
	client := NewClient(store, "khi-vrt-artifacts", AccessModeConsole, 4)

	hasBefore, err := client.HasSnapshot(context.Background(), "chrome-136-trixie", "sha-base")
	if err != nil {
		t.Fatalf("HasSnapshot(before) unexpected error = %v", err)
	}
	if hasBefore {
		t.Fatal("HasSnapshot(before) = true, want false")
	}

	srcDir := t.TempDir()
	imgRel := "images/story/header--default.png"
	imgAbs := filepath.Join(srcDir, filepath.FromSlash(imgRel))
	if err := os.MkdirAll(filepath.Dir(imgAbs), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	wantImg := []byte("fake-png-bytes")
	if err := os.WriteFile(imgAbs, wantImg, 0o600); err != nil {
		t.Fatalf("write png failed: %v", err)
	}

	manifest := capture.Manifest{
		Version:            capture.ManifestVersion,
		CommitSHA:          "sha-base",
		BuilderFingerprint: "chrome-136-trixie",
		Entries: map[string]capture.SnapshotEntry{
			"story/header--default": {
				ID:           "story/header--default",
				StoryID:      "header--default",
				Title:        "Header",
				Name:         "Default",
				ImageRelPath: imgRel,
				SHA256:       "abcd",
			},
			"story/skipped--one": {
				ID:      "story/skipped--one",
				StoryID: "skipped--one",
				Skipped: true,
			},
		},
	}

	if err := client.UploadSnapshot(context.Background(), "chrome-136-trixie", "sha-base", srcDir, manifest); err != nil {
		t.Fatalf("UploadSnapshot() unexpected error = %v", err)
	}

	// Verify manifest.json is uploaded last as atomic completion marker.
	wantOrder := []string{
		"snapshots/chrome-136-trixie/sha-base/images/story/header--default.png",
		"snapshots/chrome-136-trixie/sha-base/manifest.json",
	}
	if diff := cmp.Diff(wantOrder, store.putOrder); diff != "" {
		t.Errorf("upload order mismatch (-want +got):\n%s", diff)
	}
	manifestObj := store.objects["snapshots/chrome-136-trixie/sha-base/manifest.json"]
	if len(manifestObj.data) == 0 || manifestObj.data[len(manifestObj.data)-1] != '\n' {
		t.Errorf("uploaded manifest missing trailing newline: %q", string(manifestObj.data))
	}

	hasAfter, err := client.HasSnapshot(context.Background(), "chrome-136-trixie", "sha-base")
	if err != nil {
		t.Fatalf("HasSnapshot(after) unexpected error = %v", err)
	}
	if !hasAfter {
		t.Fatal("HasSnapshot(after) = false, want true")
	}

	destDir := t.TempDir()
	gotManifest, err := client.DownloadSnapshot(context.Background(), "chrome-136-trixie", "sha-base", destDir)
	if err != nil {
		t.Fatalf("DownloadSnapshot() unexpected error = %v", err)
	}
	if diff := cmp.Diff(manifest, gotManifest); diff != "" {
		t.Errorf("DownloadSnapshot() manifest mismatch (-want +got):\n%s", diff)
	}

	gotImg, err := os.ReadFile(filepath.Join(destDir, filepath.FromSlash(imgRel)))
	if err != nil {
		t.Fatalf("failed to read downloaded PNG: %v", err)
	}
	if string(gotImg) != string(wantImg) {
		t.Errorf("downloaded PNG = %q, want %q", string(gotImg), string(wantImg))
	}
}

func TestDownloadSnapshot_NotFound(t *testing.T) {
	client := NewClient(newMemoryObjectStore(), "khi-vrt-artifacts", AccessModeConsole, 2)
	_, err := client.DownloadSnapshot(context.Background(), "fp", "missing", t.TempDir())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("DownloadSnapshot() error = %v, want ErrNotFound", err)
	}
}

func TestUploadDirectory_SetsContentHeaders(t *testing.T) {
	store := newMemoryObjectStore()
	client := NewClient(store, "khi-vrt-artifacts", AccessModeConsole, 4)

	dir := t.TempDir()
	files := map[string]string{
		"index.html":         "<html></html>",
		"report.json":        "{}",
		"diff/story/a.png":   "png",
		"extra/metadata.bin": "raw",
	}
	for rel, content := range files {
		abs := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("mkdir failed: %v", err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o600); err != nil {
			t.Fatalf("write failed: %v", err)
		}
	}

	prefix := ReportPrefix("head123", "build-99")
	if err := client.UploadDirectory(context.Background(), prefix, dir); err != nil {
		t.Fatalf("UploadDirectory() unexpected error = %v", err)
	}

	gotTypes := map[string]string{}
	for k, v := range store.objects {
		gotTypes[k] = v.contentType
	}
	wantTypes := map[string]string{
		"reports/head123/build-99/index.html":         "text/html; charset=utf-8",
		"reports/head123/build-99/report.json":        "application/json; charset=utf-8",
		"reports/head123/build-99/diff/story/a.png":   "image/png",
		"reports/head123/build-99/extra/metadata.bin": "application/octet-stream",
	}
	if diff := cmp.Diff(wantTypes, gotTypes); diff != "" {
		t.Errorf("uploaded content types mismatch (-want +got):\n%s", diff)
	}
}

func TestNewClient_DefaultsAndAccessMode(t *testing.T) {
	c := NewClient(newMemoryObjectStore(), "gs://demo-bucket/", "", 0)
	if c.AccessMode() != AccessModeConsole {
		t.Errorf("AccessMode() = %q, want %q", c.AccessMode(), AccessModeConsole)
	}
	if c.workers != defaultConcurrency {
		t.Errorf("default workers = %d, want %d", c.workers, defaultConcurrency)
	}
	cNeg := NewClient(newMemoryObjectStore(), "demo", AccessModePublic, -5)
	if cNeg.AccessMode() != AccessModePublic || cNeg.workers != defaultConcurrency {
		t.Errorf("negative workers client = (mode=%q, workers=%d), want (public, %d)", cNeg.AccessMode(), cNeg.workers, defaultConcurrency)
	}
	cCustom := NewClient(newMemoryObjectStore(), "demo", AccessModePublic, 3)
	if cCustom.workers != 3 {
		t.Errorf("custom workers = %d, want 3", cCustom.workers)
	}
}

func TestDownloadSnapshot_InitializesNilEntries(t *testing.T) {
	store := newMemoryObjectStore()
	_ = store.Put(context.Background(), "snapshots/fp/sha1/manifest.json", "application/json", "", []byte(`{"version":1}`))
	client := NewClient(store, "demo", AccessModeConsole, 2)

	got, err := client.DownloadSnapshot(context.Background(), "fp", "sha1", t.TempDir())
	if err != nil {
		t.Fatalf("DownloadSnapshot() unexpected error = %v", err)
	}
	if got.Entries == nil {
		t.Fatal("DownloadSnapshot() Entries is nil, want empty map")
	}
}

func TestStorageStore_HTTPIntegration(t *testing.T) {
	if _, err := NewStorageStore(context.Background(), "  gs:/// "); err == nil {
		t.Fatal("NewStorageStore(empty) expected error, got nil")
	}

	var mu sync.Mutex
	objects := map[string][]byte{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/b/test-bucket/o"):
			name := r.URL.Query().Get("name")
			if name == "" {
				name = "uploaded.txt"
			}
			body, _ := io.ReadAll(r.Body)
			objects[name] = body
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"name": name})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/b/test-bucket/o/"):
			idx := strings.Index(r.URL.Path, "/o/")
			name := r.URL.Path[idx+3:]
			data, ok := objects[name]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":{"code":404,"message":"not found"}}`))
				return
			}
			if r.URL.Query().Get("alt") == "media" {
				_, _ = w.Write(data)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"name": name})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	store, err := NewStorageStore(
		context.Background(),
		"gs://test-bucket/",
		option.WithEndpoint(srv.URL),
		option.WithoutAuthentication(),
	)
	if err != nil {
		t.Fatalf("NewStorageStore() unexpected error = %v", err)
	}

	exists, err := store.Exists(context.Background(), "missing.png")
	if err != nil || exists {
		t.Fatalf("Exists(missing) = (%v, %v), want (false, nil)", exists, err)
	}
	if _, err := store.Get(context.Background(), "missing.png"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(missing) error = %v, want ErrNotFound", err)
	}

	if err := store.Put(context.Background(), "hello.txt", "text/plain", "no-cache", []byte("world")); err != nil {
		t.Fatalf("Put() unexpected error = %v", err)
	}
	// Seed direct key matching GET lookup path.
	mu.Lock()
	objects["hello.txt"] = []byte("world")
	mu.Unlock()

	exists, err = store.Exists(context.Background(), "hello.txt")
	if err != nil || !exists {
		t.Fatalf("Exists(hello.txt) = (%v, %v), want (true, nil)", exists, err)
	}
	got, err := store.Get(context.Background(), "hello.txt")
	if err != nil || string(got) != "world" {
		t.Fatalf("Get(hello.txt) = (%q, %v), want (world, nil)", string(got), err)
	}
}

func TestHelpers_IsNotFoundAndEscapeObjectPath(t *testing.T) {
	if !isNotFoundErr(&googleapi.Error{Code: http.StatusNotFound}) {
		t.Error("isNotFoundErr(404) = false, want true")
	}
	if isNotFoundErr(&googleapi.Error{Code: http.StatusInternalServerError}) {
		t.Error("isNotFoundErr(500) = true, want false")
	}
	if isNotFoundErr(errors.New("plain error")) {
		t.Error("isNotFoundErr(plain) = true, want false")
	}
	if got, want := escapeObjectPath("reports/my report/a#b.html"), "reports/my%20report/a%23b.html"; got != want {
		t.Errorf("escapeObjectPath() = %q, want %q", got, want)
	}
}
