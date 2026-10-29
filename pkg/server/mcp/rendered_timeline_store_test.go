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
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

const testRenderedTimelineBaseURL = "http://127.0.0.1:8080/api/v1/rendered-timelines/"

func TestRenderedTimelineStore_Issue(t *testing.T) {
	store := NewRenderedTimelineStore(testRenderedTimelineBaseURL, time.Hour)
	inputPNG := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	before := time.Now()

	grant := store.Issue(inputPNG)

	fileName, found := strings.CutPrefix(grant.URL, testRenderedTimelineBaseURL)
	if !found {
		t.Fatalf("Issue().URL = %q, want prefix %q", grant.URL, testRenderedTimelineBaseURL)
	}
	token, hasExt := strings.CutSuffix(fileName, ".png")
	if !hasExt || token == "" {
		t.Fatalf("Issue() file name = %q, want non-empty <token>.png", fileName)
	}
	if grant.ExpiresAt.Before(before.Add(time.Hour)) {
		t.Errorf("Issue().ExpiresAt = %v, want at least %v", grant.ExpiresAt, before.Add(time.Hour))
	}

	// Mutating the caller slice must not alter the stored PNG payload.
	inputPNG[0] = 0x00
	gotBytes, err := store.lookup(fileName)
	if err != nil {
		t.Fatalf("lookup() unexpected error = %v", err)
	}
	wantBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if diff := cmp.Diff(wantBytes, gotBytes); diff != "" {
		t.Errorf("lookup() bytes mismatch (-want +got):\n%s", diff)
	}
}

func TestRenderedTimelineStore_IssueReturnsDistinctURLs(t *testing.T) {
	store := NewRenderedTimelineStore(testRenderedTimelineBaseURL, time.Hour)
	pngBytes := []byte{0x89, 0x50, 0x4E, 0x47}

	first := store.Issue(pngBytes)
	second := store.Issue(pngBytes)

	if first.URL == second.URL {
		t.Errorf("Issue() returned the same URL %q twice, want distinct URLs", first.URL)
	}
}

func TestRenderedTimelineStore_Lookup(t *testing.T) {
	pngBytes := []byte{0x89, 0x50, 0x4E, 0x47}

	testCases := []struct {
		name            string
		ttl             time.Duration
		fileName        func(grant RenderedTimelineGrant) string
		wantFirstBytes  []byte
		wantFirstErr    error
		wantSecondBytes []byte
		wantSecondErr   error
	}{
		{
			name: "issued image URL can be read repeatedly before expiry",
			ttl:  time.Hour,
			fileName: func(grant RenderedTimelineGrant) string {
				return strings.TrimPrefix(grant.URL, testRenderedTimelineBaseURL)
			},
			wantFirstBytes:  pngBytes,
			wantSecondBytes: pngBytes,
		},
		{
			name: "file name never issued is unknown",
			ttl:  time.Hour,
			fileName: func(grant RenderedTimelineGrant) string {
				return "not-issued.png"
			},
			wantFirstErr:  ErrUnknownRenderedTimeline,
			wantSecondErr: ErrUnknownRenderedTimeline,
		},
		{
			name: "token without .png extension is unknown",
			ttl:  time.Hour,
			fileName: func(grant RenderedTimelineGrant) string {
				return strings.TrimSuffix(strings.TrimPrefix(grant.URL, testRenderedTimelineBaseURL), ".png")
			},
			wantFirstErr:  ErrUnknownRenderedTimeline,
			wantSecondErr: ErrUnknownRenderedTimeline,
		},
		{
			name: "bare .png without token is unknown",
			ttl:  time.Hour,
			fileName: func(grant RenderedTimelineGrant) string {
				return ".png"
			},
			wantFirstErr:  ErrUnknownRenderedTimeline,
			wantSecondErr: ErrUnknownRenderedTimeline,
		},
		{
			name: "expired image URL returns ErrExpiredRenderedTimeline then ErrUnknownRenderedTimeline",
			ttl:  0,
			fileName: func(grant RenderedTimelineGrant) string {
				return strings.TrimPrefix(grant.URL, testRenderedTimelineBaseURL)
			},
			wantFirstErr:  ErrExpiredRenderedTimeline,
			wantSecondErr: ErrUnknownRenderedTimeline,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := NewRenderedTimelineStore(testRenderedTimelineBaseURL, tc.ttl)
			targetFile := tc.fileName(store.Issue(pngBytes))

			gotFirst, firstErr := store.lookup(targetFile)
			if !errors.Is(firstErr, tc.wantFirstErr) {
				t.Errorf("first lookup() error = %v, want %v", firstErr, tc.wantFirstErr)
			}
			if diff := cmp.Diff(tc.wantFirstBytes, gotFirst); diff != "" {
				t.Errorf("first lookup() bytes mismatch (-want +got):\n%s", diff)
			}

			gotSecond, secondErr := store.lookup(targetFile)
			if !errors.Is(secondErr, tc.wantSecondErr) {
				t.Errorf("second lookup() error = %v, want %v", secondErr, tc.wantSecondErr)
			}
			if diff := cmp.Diff(tc.wantSecondBytes, gotSecond); diff != "" {
				t.Errorf("second lookup() bytes mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRenderedTimelineStore_IssueSweepsExpiredTokens(t *testing.T) {
	pngBytes := []byte{0x89, 0x50, 0x4E, 0x47}

	testCases := []struct {
		name          string
		ttl           time.Duration
		wantMapLen    int
		wantLookupErr error
	}{
		{
			name:          "sweeps expired image tokens on subsequent issue",
			ttl:           0,
			wantMapLen:    1,
			wantLookupErr: ErrUnknownRenderedTimeline,
		},
		{
			name:       "keeps unexpired image tokens on subsequent issue",
			ttl:        time.Hour,
			wantMapLen: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := NewRenderedTimelineStore(testRenderedTimelineBaseURL, tc.ttl)
			firstFileName := strings.TrimPrefix(store.Issue(pngBytes).URL, testRenderedTimelineBaseURL)
			store.Issue(pngBytes)

			if got := len(store.entriesByToken); got != tc.wantMapLen {
				t.Errorf("len(entriesByToken) = %d, want %d", got, tc.wantMapLen)
			}
			if _, err := store.lookup(firstFileName); !errors.Is(err, tc.wantLookupErr) {
				t.Errorf("lookup() of first image error = %v, want %v", err, tc.wantLookupErr)
			}
		})
	}
}

func TestRenderedTimelineStore_ServeImage(t *testing.T) {
	pngBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

	type serveOutcome struct {
		StatusCode    int
		ContentType   string
		ContentLength string
		Body          []byte
	}

	testCases := []struct {
		name     string
		ttl      time.Duration
		method   string
		fileName func(grant RenderedTimelineGrant) string
		want     serveOutcome
	}{
		{
			name:   "GET returns PNG payload with image/png headers",
			ttl:    time.Hour,
			method: http.MethodGet,
			fileName: func(grant RenderedTimelineGrant) string {
				return strings.TrimPrefix(grant.URL, testRenderedTimelineBaseURL)
			},
			want: serveOutcome{
				StatusCode:    http.StatusOK,
				ContentType:   "image/png",
				ContentLength: "8",
				Body:          pngBytes,
			},
		},
		{
			name:   "HEAD returns image/png headers with empty body",
			ttl:    time.Hour,
			method: http.MethodHead,
			fileName: func(grant RenderedTimelineGrant) string {
				return strings.TrimPrefix(grant.URL, testRenderedTimelineBaseURL)
			},
			want: serveOutcome{
				StatusCode:    http.StatusOK,
				ContentType:   "image/png",
				ContentLength: "8",
				Body:          []byte{},
			},
		},
		{
			name:   "GET unknown token returns 404 JSON error",
			ttl:    time.Hour,
			method: http.MethodGet,
			fileName: func(grant RenderedTimelineGrant) string {
				return "missing.png"
			},
			want: serveOutcome{
				StatusCode:  http.StatusNotFound,
				ContentType: "application/json",
				Body:        []byte("{\"error\":\"unknown rendered timeline image URL. Call render_timelines to generate a new image URL.\"}\n"),
			},
		},
		{
			name:   "GET expired token returns 410 JSON error",
			ttl:    0,
			method: http.MethodGet,
			fileName: func(grant RenderedTimelineGrant) string {
				return strings.TrimPrefix(grant.URL, testRenderedTimelineBaseURL)
			},
			want: serveOutcome{
				StatusCode:  http.StatusGone,
				ContentType: "application/json",
				Body:        []byte("{\"error\":\"rendered timeline image URL has expired. Call render_timelines to generate a new image URL.\"}\n"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := NewRenderedTimelineStore(testRenderedTimelineBaseURL, tc.ttl)
			grant := store.Issue(pngBytes)
			targetFile := tc.fileName(grant)

			req := httptest.NewRequest(tc.method, testRenderedTimelineBaseURL+targetFile, nil)
			rec := httptest.NewRecorder()

			store.ServeImage(rec, req, targetFile)

			got := serveOutcome{
				StatusCode:    rec.Code,
				ContentType:   rec.Header().Get("Content-Type"),
				ContentLength: rec.Header().Get("Content-Length"),
				Body:          rec.Body.Bytes(),
			}
			if got.Body == nil {
				got.Body = []byte{}
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ServeImage() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
