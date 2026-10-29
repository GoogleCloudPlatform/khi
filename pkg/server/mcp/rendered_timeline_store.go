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

// DefaultRenderedTimelineTTL is how long an issued rendered timeline image URL remains valid.
const DefaultRenderedTimelineTTL = 30 * time.Minute

var (
	// ErrUnknownRenderedTimeline is returned when a rendered timeline image token was never issued or has already been removed.
	ErrUnknownRenderedTimeline = errors.New("unknown rendered timeline image URL")
	// ErrExpiredRenderedTimeline is returned when a rendered timeline image token is accessed after it expires.
	ErrExpiredRenderedTimeline = errors.New("rendered timeline image URL has expired")
)

// RenderedTimelineGrant describes an issued HTTP GET URL for a rendered timeline PNG image.
type RenderedTimelineGrant struct {
	// URL is the HTTP GET address that serves the rendered PNG image.
	URL string
	// ExpiresAt is the time after which the URL stops serving the image.
	ExpiresAt time.Time
}

type renderedTimelineEntry struct {
	pngBytes  []byte
	expiresAt time.Time
}

type renderedTimelineErrorResponse struct {
	Error string `json:"error"`
}

// RenderedTimelineStore stores rendered timeline PNG images in memory with a TTL and serves them over HTTP GET.
type RenderedTimelineStore struct {
	baseURL string
	ttl     time.Duration

	mu             sync.Mutex
	entriesByToken map[string]renderedTimelineEntry
}

// NewRenderedTimelineStore returns a RenderedTimelineStore whose issued URLs append "<token>.png" to baseURL.
// baseURL must end with the path that routes to ServeImage, including the trailing slash.
func NewRenderedTimelineStore(baseURL string, ttl time.Duration) *RenderedTimelineStore {
	return &RenderedTimelineStore{
		baseURL:        baseURL,
		ttl:            ttl,
		entriesByToken: make(map[string]renderedTimelineEntry),
	}
}

// Issue stores pngBytes until the configured TTL elapses and returns its short-lived HTTP GET URL.
func (s *RenderedTimelineStore) Issue(pngBytes []byte) RenderedTimelineGrant {
	token := rand.Text()
	now := time.Now()
	expiresAt := now.Add(s.ttl)

	s.mu.Lock()
	defer s.mu.Unlock()
	// Sweeping expired entries on issue keeps memory bounded without a background goroutine.
	for issuedToken, entry := range s.entriesByToken {
		if !now.Before(entry.expiresAt) {
			delete(s.entriesByToken, issuedToken)
		}
	}
	s.entriesByToken[token] = renderedTimelineEntry{
		pngBytes:  bytes.Clone(pngBytes),
		expiresAt: expiresAt,
	}

	return RenderedTimelineGrant{
		URL:       s.baseURL + token + ".png",
		ExpiresAt: expiresAt,
	}
}

// lookup returns the stored PNG bytes for "<token>.png" while keeping unexpired entries available for repeated GET requests.
func (s *RenderedTimelineStore) lookup(fileName string) ([]byte, error) {
	token, ok := strings.CutSuffix(fileName, ".png")
	if !ok || token == "" {
		return nil, ErrUnknownRenderedTimeline
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	entry, found := s.entriesByToken[token]
	if !found {
		return nil, ErrUnknownRenderedTimeline
	}
	if !time.Now().Before(entry.expiresAt) {
		delete(s.entriesByToken, token)
		return nil, ErrExpiredRenderedTimeline
	}
	return entry.pngBytes, nil
}

// ServeImage serves the PNG image stored for fileName ("<token>.png") over HTTP GET or HEAD.
func (s *RenderedTimelineStore) ServeImage(w http.ResponseWriter, r *http.Request, fileName string) {
	pngBytes, err := s.lookup(fileName)
	if err != nil {
		status := http.StatusNotFound
		if errors.Is(err, ErrExpiredRenderedTimeline) {
			status = http.StatusGone
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if encodeErr := json.NewEncoder(w).Encode(renderedTimelineErrorResponse{
			Error: err.Error() + ". Call render_timelines to generate a new image URL.",
		}); encodeErr != nil {
			slog.WarnContext(r.Context(), "failed to write rendered timeline error response", "error", encodeErr)
		}
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(len(pngBytes)))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	if _, err := w.Write(pngBytes); err != nil {
		slog.WarnContext(r.Context(), "failed to write rendered timeline image", "error", err)
	}
}
