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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/sync/errgroup"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	storage "google.golang.org/api/storage/v1"

	"github.com/GoogleCloudPlatform/khi/scripts/vrt/internal/capture"
)

const (
	// AccessModeConsole formats object links for authenticated GCP browser viewing (works on private buckets).
	AccessModeConsole AccessMode = "console"
	// AccessModePublic formats object links for unauthenticated storage.googleapis.com access and inline PR previews.
	AccessModePublic AccessMode = "public"

	defaultConcurrency      = 16
	snapshotPrefixRoot      = "snapshots"
	reportPrefixRoot        = "reports"
	immutableCacheControl   = "public, max-age=2592000"
	reportIndexCacheControl = "no-cache"
)

// ErrNotFound is returned when a requested GCS object does not exist.
var ErrNotFound = errors.New("gcs object not found")

var _ ObjectStore = (*StorageStore)(nil)

// AccessMode controls how GCS object URLs are generated for human viewing and PR comments.
type AccessMode string

// ObjectStore abstracts raw GCS object read, write, and existence checks for testability.
type ObjectStore interface {
	// Get retrieves the full byte contents of object or returns ErrNotFound.
	Get(ctx context.Context, object string) ([]byte, error)
	// Put writes data to object with the specified Content-Type and Cache-Control headers.
	Put(ctx context.Context, object string, contentType string, cacheControl string, data []byte) error
	// Exists reports whether object is present in the bucket.
	Exists(ctx context.Context, object string) (bool, error)
}

// StorageStore implements ObjectStore using Google Cloud Storage JSON API v1 with Application Default Credentials.
type StorageStore struct {
	service *storage.Service
	bucket  string
}

// Client manages VRT snapshot caches and visual diff report artifacts in a GCS bucket.
type Client struct {
	store      ObjectStore
	bucket     string
	accessMode AccessMode
	workers    int
}

// ParseAccessMode validates and normalizes an access mode string (defaults to AccessModeConsole when empty).
func ParseAccessMode(raw string) (AccessMode, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", string(AccessModeConsole):
		return AccessModeConsole, nil
	case string(AccessModePublic):
		return AccessModePublic, nil
	default:
		return "", fmt.Errorf("invalid gcs access mode %q: expected console or public", raw)
	}
}

// NewStorageStore creates a GCS-backed ObjectStore targeting bucket using optional Google API client options.
func NewStorageStore(ctx context.Context, bucket string, opts ...option.ClientOption) (*StorageStore, error) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(bucket), "gs://")
	trimmed = strings.Trim(trimmed, "/")
	if trimmed == "" {
		return nil, errors.New("gcs bucket name is required")
	}
	svc, err := storage.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize GCS storage service: %w", err)
	}
	return &StorageStore{
		service: svc,
		bucket:  trimmed,
	}, nil
}

// Get downloads the object contents from GCS.
func (s *StorageStore) Get(ctx context.Context, object string) ([]byte, error) {
	resp, err := s.service.Objects.Get(s.bucket, object).Context(ctx).Download()
	if err != nil {
		if isNotFoundErr(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to download gs://%s/%s: %w", s.bucket, object, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read gs://%s/%s: %w", s.bucket, object, err)
	}
	return data, nil
}

// Put uploads data to object in GCS with the given Content-Type and Cache-Control metadata.
func (s *StorageStore) Put(ctx context.Context, object string, contentType string, cacheControl string, data []byte) error {
	obj := &storage.Object{
		Name:         object,
		ContentType:  contentType,
		CacheControl: cacheControl,
	}
	_, err := s.service.Objects.Insert(s.bucket, obj).
		Context(ctx).
		Media(bytes.NewReader(data), googleapi.ContentType(contentType)).
		Do()
	if err != nil {
		return fmt.Errorf("failed to upload gs://%s/%s: %w", s.bucket, object, err)
	}
	return nil
}

// Exists checks object metadata to determine whether object exists in GCS.
func (s *StorageStore) Exists(ctx context.Context, object string) (bool, error) {
	_, err := s.service.Objects.Get(s.bucket, object).Context(ctx).Fields("name").Do()
	if err != nil {
		if isNotFoundErr(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat gs://%s/%s: %w", s.bucket, object, err)
	}
	return true, nil
}

// NewClient constructs a VRT GCS Client backed by store.
func NewClient(store ObjectStore, bucket string, mode AccessMode, workers int) *Client {
	trimmed := strings.TrimPrefix(strings.TrimSpace(bucket), "gs://")
	trimmed = strings.Trim(trimmed, "/")
	if mode == "" {
		mode = AccessModeConsole
	}
	if workers <= 0 {
		workers = defaultConcurrency
	}
	return &Client{
		store:      store,
		bucket:     trimmed,
		accessMode: mode,
		workers:    workers,
	}
}

// AccessMode returns the configured link formatting mode.
func (c *Client) AccessMode() AccessMode {
	return c.accessMode
}

// SnapshotPrefix returns the GCS object prefix for a builder fingerprint and commit SHA.
func SnapshotPrefix(fingerprint string, commitSHA string) string {
	return path.Join(snapshotPrefixRoot, fingerprint, commitSHA)
}

// ReportPrefix returns the GCS object prefix for a PR diff report run.
func ReportPrefix(headSHA string, runID string) string {
	return path.Join(reportPrefixRoot, headSHA, runID)
}

// ObjectURL returns the browser-accessible HTTPS URL for objectPath under the configured AccessMode.
func (c *Client) ObjectURL(objectPath string) string {
	escaped := escapeObjectPath(objectPath)
	if c.accessMode == AccessModePublic {
		return fmt.Sprintf("https://storage.googleapis.com/%s/%s", c.bucket, escaped)
	}
	return fmt.Sprintf("https://storage.cloud.google.com/%s/%s", c.bucket, escaped)
}

// HasSnapshot reports whether a completed snapshot manifest exists in GCS for fingerprint and commitSHA.
func (c *Client) HasSnapshot(ctx context.Context, fingerprint string, commitSHA string) (bool, error) {
	manifestObj := path.Join(SnapshotPrefix(fingerprint, commitSHA), capture.ManifestFileName)
	return c.store.Exists(ctx, manifestObj)
}

// DownloadSnapshot fetches a cached snapshot manifest and all referenced PNG files into destDir.
func (c *Client) DownloadSnapshot(ctx context.Context, fingerprint string, commitSHA string, destDir string) (capture.Manifest, error) {
	prefix := SnapshotPrefix(fingerprint, commitSHA)
	manifestObj := path.Join(prefix, capture.ManifestFileName)
	raw, err := c.store.Get(ctx, manifestObj)
	if err != nil {
		return capture.Manifest{}, err
	}

	var manifest capture.Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return capture.Manifest{}, fmt.Errorf("failed to decode cached manifest %s: %w", manifestObj, err)
	}
	if manifest.Entries == nil {
		manifest.Entries = map[string]capture.SnapshotEntry{}
	}

	if err := capture.WriteManifest(filepath.Join(destDir, capture.ManifestFileName), manifest); err != nil {
		return capture.Manifest{}, err
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(c.workers)
	for _, entry := range manifest.Entries {
		if entry.ImageRelPath == "" {
			continue
		}
		entry := entry
		g.Go(func() error {
			objName := path.Join(prefix, entry.ImageRelPath)
			data, err := c.store.Get(gctx, objName)
			if err != nil {
				return err
			}
			localPath := filepath.Join(destDir, filepath.FromSlash(entry.ImageRelPath))
			if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
				return fmt.Errorf("failed to create snapshot dir: %w", err)
			}
			return os.WriteFile(localPath, data, 0o600)
		})
	}
	if err := g.Wait(); err != nil {
		return capture.Manifest{}, err
	}
	return manifest, nil
}

// UploadSnapshot uploads all snapshot PNGs first and writes manifest.json last as an atomic commit marker.
func (c *Client) UploadSnapshot(ctx context.Context, fingerprint string, commitSHA string, srcDir string, manifest capture.Manifest) error {
	prefix := SnapshotPrefix(fingerprint, commitSHA)
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(c.workers)

	for _, entry := range manifest.Entries {
		if entry.ImageRelPath == "" {
			continue
		}
		entry := entry
		g.Go(func() error {
			localPath := filepath.Join(srcDir, filepath.FromSlash(entry.ImageRelPath))
			data, err := os.ReadFile(localPath)
			if err != nil {
				return fmt.Errorf("failed to read snapshot image %s: %w", localPath, err)
			}
			objName := path.Join(prefix, entry.ImageRelPath)
			return c.store.Put(gctx, objName, "image/png", immutableCacheControl, data)
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot manifest: %w", err)
	}
	manifestBytes = append(manifestBytes, '\n')
	manifestObj := path.Join(prefix, capture.ManifestFileName)
	return c.store.Put(ctx, manifestObj, "application/json", immutableCacheControl, manifestBytes)
}

// UploadDirectory recursively uploads all files inside srcDir under GCS prefix with appropriate Content-Type headers.
func (c *Client) UploadDirectory(ctx context.Context, prefix string, srcDir string) error {
	var relPaths []string
	err := filepath.WalkDir(srcDir, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(srcDir, p)
		if err != nil {
			return err
		}
		relPaths = append(relPaths, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to enumerate report directory %s: %w", srcDir, err)
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(c.workers)
	for _, rel := range relPaths {
		rel := rel
		g.Go(func() error {
			localPath := filepath.Join(srcDir, filepath.FromSlash(rel))
			data, err := os.ReadFile(localPath)
			if err != nil {
				return fmt.Errorf("failed to read artifact %s: %w", localPath, err)
			}
			contentType, cacheControl := contentHeadersForPath(rel)
			return c.store.Put(gctx, path.Join(prefix, rel), contentType, cacheControl, data)
		})
	}
	return g.Wait()
}

func isNotFoundErr(err error) bool {
	var apiErr *googleapi.Error
	return errors.As(err, &apiErr) && apiErr.Code == http.StatusNotFound
}

func escapeObjectPath(objectPath string) string {
	parts := strings.Split(strings.TrimPrefix(objectPath, "/"), "/")
	for i, seg := range parts {
		parts[i] = url.PathEscape(seg)
	}
	return strings.Join(parts, "/")
}

func contentHeadersForPath(relPath string) (string, string) {
	switch strings.ToLower(path.Ext(relPath)) {
	case ".html":
		return "text/html; charset=utf-8", reportIndexCacheControl
	case ".json":
		return "application/json; charset=utf-8", reportIndexCacheControl
	case ".png":
		return "image/png", immutableCacheControl
	default:
		return "application/octet-stream", reportIndexCacheControl
	}
}
