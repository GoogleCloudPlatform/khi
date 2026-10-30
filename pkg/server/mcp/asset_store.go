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

package mcp

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DefaultAssetTTL defines how long an issued MCP asset URL remains valid.
const DefaultAssetTTL = 30 * time.Minute

var (
	// ErrUnknownAsset indicates that an asset token was never issued or has already been removed.
	ErrUnknownAsset = errors.New("unknown asset URL")
	// ErrExpiredAsset indicates that an asset token was accessed after its expiration time.
	ErrExpiredAsset = errors.New("asset URL has expired")
)

// AssetGrant describes a short-lived HTTP GET URL issued for an MCP-generated asset.
type AssetGrant struct {
	// URL is the HTTP GET address that serves the stored asset.
	URL string
	// ExpiresAt is the timestamp after which the URL stops serving the asset.
	ExpiresAt time.Time
}

type assetEntry struct {
	data        []byte
	contentType string
	extension   string
	expiresAt   time.Time
}

type assetErrorResponse struct {
	Error string `json:"error"`
}

// AssetStore stores MCP-generated assets in memory with a TTL and serves them over HTTP GET and HEAD.
type AssetStore struct {
	baseURL string
	ttl     time.Duration

	mu             sync.Mutex
	entriesByToken map[string]assetEntry
}

// NewAssetStore returns an AssetStore whose issued URLs append "<token><extension>" to baseURL.
//
// baseURL must end with the route path mapped to ServeAsset, including the trailing slash.
func NewAssetStore(baseURL string, ttl time.Duration) *AssetStore {
	return &AssetStore{
		baseURL:        baseURL,
		ttl:            ttl,
		entriesByToken: make(map[string]assetEntry),
	}
}

// Issue stores data with its MIME contentType and file extension until the configured TTL elapses and returns its HTTP GET URL.
func (s *AssetStore) Issue(data []byte, contentType string, extension string) AssetGrant {
	if extension != "" && !strings.HasPrefix(extension, ".") {
		extension = "." + extension
	}
	token := rand.Text()
	now := time.Now()
	expiresAt := now.Add(s.ttl)

	s.mu.Lock()
	defer s.mu.Unlock()
	// Sweeping expired entries on issue bounds memory growth without a background goroutine.
	for issuedToken, entry := range s.entriesByToken {
		if !now.Before(entry.expiresAt) {
			delete(s.entriesByToken, issuedToken)
		}
	}
	s.entriesByToken[token] = assetEntry{
		data:        bytes.Clone(data),
		contentType: contentType,
		extension:   extension,
		expiresAt:   expiresAt,
	}

	return AssetGrant{
		URL:       s.baseURL + token + extension,
		ExpiresAt: expiresAt,
	}
}

// lookup returns the stored asset for fileName ("<token><extension>") while keeping unexpired entries readable across repeated GET requests.
func (s *AssetStore) lookup(fileName string) (assetEntry, error) {
	token, ext, hasDot := strings.Cut(fileName, ".")
	if token == "" {
		return assetEntry{}, ErrUnknownAsset
	}
	if hasDot {
		ext = "." + ext
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	entry, found := s.entriesByToken[token]
	if !found || entry.extension != ext {
		return assetEntry{}, ErrUnknownAsset
	}
	if !time.Now().Before(entry.expiresAt) {
		delete(s.entriesByToken, token)
		return assetEntry{}, ErrExpiredAsset
	}
	return entry, nil
}

// ServeAsset serves the asset stored for fileName over HTTP GET or HEAD.
func (s *AssetStore) ServeAsset(w http.ResponseWriter, r *http.Request, fileName string) {
	entry, err := s.lookup(fileName)
	if err != nil {
		status := http.StatusNotFound
		if errors.Is(err, ErrExpiredAsset) {
			status = http.StatusGone
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if encodeErr := json.NewEncoder(w).Encode(assetErrorResponse{
			Error: err.Error(),
		}); encodeErr != nil {
			slog.WarnContext(r.Context(), "failed to write MCP asset error response", "error", encodeErr)
		}
		return
	}

	w.Header().Set("Content-Type", entry.contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(entry.data)))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	if _, err := w.Write(entry.data); err != nil {
		slog.WarnContext(r.Context(), "failed to write MCP asset response", "error", err)
	}
}
