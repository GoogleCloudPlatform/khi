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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

const testAssetBaseURL = "http://127.0.0.1:8080/api/v1/mcp-assets/"

func TestAssetStore_Issue(t *testing.T) {
	store := NewAssetStore(testAssetBaseURL, time.Hour)
	inputPNG := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	before := time.Now()

	grant := store.Issue(inputPNG, "image/png", ".png")

	fileName, found := strings.CutPrefix(grant.URL, testAssetBaseURL)
	if !found {
		t.Fatalf("Issue().URL = %q, want prefix %q", grant.URL, testAssetBaseURL)
	}
	token, hasExt := strings.CutSuffix(fileName, ".png")
	if !hasExt || token == "" {
		t.Fatalf("Issue() file name = %q, want non-empty <token>.png", fileName)
	}
	if grant.ExpiresAt.Before(before.Add(time.Hour)) {
		t.Errorf("Issue().ExpiresAt = %v, want at least %v", grant.ExpiresAt, before.Add(time.Hour))
	}

	// Mutating the caller slice must not alter the stored asset payload.
	inputPNG[0] = 0x00
	gotEntry, err := store.lookup(fileName)
	if err != nil {
		t.Fatalf("lookup() unexpected error = %v", err)
	}
	wantBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if diff := cmp.Diff(wantBytes, gotEntry.data); diff != "" {
		t.Errorf("lookup() bytes mismatch (-want +got):\n%s", diff)
	}
}

func TestAssetStore_IssueReturnsDistinctURLs(t *testing.T) {
	store := NewAssetStore(testAssetBaseURL, time.Hour)
	pngBytes := []byte{0x89, 0x50, 0x4E, 0x47}

	first := store.Issue(pngBytes, "image/png", ".png")
	second := store.Issue(pngBytes, "image/png", ".png")

	if first.URL == second.URL {
		t.Errorf("Issue() returned the same URL %q twice, want distinct URLs", first.URL)
	}
}

func TestAssetStore_Lookup(t *testing.T) {
	pngBytes := []byte{0x89, 0x50, 0x4E, 0x47}

	type lookupResult struct {
		Data        []byte
		ContentType string
		Err         error
	}
	type lookupOutcome struct {
		First  lookupResult
		Second lookupResult
	}

	testCases := []struct {
		name        string
		ttl         time.Duration
		contentType string
		extension   string
		fileName    func(grant AssetGrant) string
		want        lookupOutcome
	}{
		{
			name:        "issued PNG URL can be read repeatedly before expiry",
			ttl:         time.Hour,
			contentType: "image/png",
			extension:   ".png",
			fileName: func(grant AssetGrant) string {
				return strings.TrimPrefix(grant.URL, testAssetBaseURL)
			},
			want: lookupOutcome{
				First:  lookupResult{Data: pngBytes, ContentType: "image/png"},
				Second: lookupResult{Data: pngBytes, ContentType: "image/png"},
			},
		},
		{
			name:        "extension without leading dot is normalized",
			ttl:         time.Hour,
			contentType: "image/svg+xml",
			extension:   "svg",
			fileName: func(grant AssetGrant) string {
				return strings.TrimPrefix(grant.URL, testAssetBaseURL)
			},
			want: lookupOutcome{
				First:  lookupResult{Data: pngBytes, ContentType: "image/svg+xml"},
				Second: lookupResult{Data: pngBytes, ContentType: "image/svg+xml"},
			},
		},
		{
			name:        "extensionless asset can be read when issued without extension",
			ttl:         time.Hour,
			contentType: "application/octet-stream",
			extension:   "",
			fileName: func(grant AssetGrant) string {
				return strings.TrimPrefix(grant.URL, testAssetBaseURL)
			},
			want: lookupOutcome{
				First:  lookupResult{Data: pngBytes, ContentType: "application/octet-stream"},
				Second: lookupResult{Data: pngBytes, ContentType: "application/octet-stream"},
			},
		},
		{
			name:        "file name never issued is unknown",
			ttl:         time.Hour,
			contentType: "image/png",
			extension:   ".png",
			fileName: func(grant AssetGrant) string {
				return "not-issued.png"
			},
			want: lookupOutcome{
				First:  lookupResult{Err: ErrUnknownAsset},
				Second: lookupResult{Err: ErrUnknownAsset},
			},
		},
		{
			name:        "token requested without its issued extension is unknown",
			ttl:         time.Hour,
			contentType: "image/png",
			extension:   ".png",
			fileName: func(grant AssetGrant) string {
				return strings.TrimSuffix(strings.TrimPrefix(grant.URL, testAssetBaseURL), ".png")
			},
			want: lookupOutcome{
				First:  lookupResult{Err: ErrUnknownAsset},
				Second: lookupResult{Err: ErrUnknownAsset},
			},
		},
		{
			name:        "token requested with mismatched extension is unknown",
			ttl:         time.Hour,
			contentType: "image/png",
			extension:   ".png",
			fileName: func(grant AssetGrant) string {
				return strings.TrimSuffix(strings.TrimPrefix(grant.URL, testAssetBaseURL), ".png") + ".svg"
			},
			want: lookupOutcome{
				First:  lookupResult{Err: ErrUnknownAsset},
				Second: lookupResult{Err: ErrUnknownAsset},
			},
		},
		{
			name:        "bare extension without token is unknown",
			ttl:         time.Hour,
			contentType: "image/png",
			extension:   ".png",
			fileName: func(grant AssetGrant) string {
				return ".png"
			},
			want: lookupOutcome{
				First:  lookupResult{Err: ErrUnknownAsset},
				Second: lookupResult{Err: ErrUnknownAsset},
			},
		},
		{
			name:        "expired asset URL returns ErrExpiredAsset then ErrUnknownAsset",
			ttl:         0,
			contentType: "image/png",
			extension:   ".png",
			fileName: func(grant AssetGrant) string {
				return strings.TrimPrefix(grant.URL, testAssetBaseURL)
			},
			want: lookupOutcome{
				First:  lookupResult{Err: ErrExpiredAsset},
				Second: lookupResult{Err: ErrUnknownAsset},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := NewAssetStore(testAssetBaseURL, tc.ttl)
			targetFile := tc.fileName(store.Issue(pngBytes, tc.contentType, tc.extension))

			firstEntry, firstErr := store.lookup(targetFile)
			secondEntry, secondErr := store.lookup(targetFile)

			got := lookupOutcome{
				First: lookupResult{
					Data:        firstEntry.data,
					ContentType: firstEntry.contentType,
					Err:         firstErr,
				},
				Second: lookupResult{
					Data:        secondEntry.data,
					ContentType: secondEntry.contentType,
					Err:         secondErr,
				},
			}
			if diff := cmp.Diff(tc.want, got, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("lookup() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAssetStore_IssueSweepsExpiredTokens(t *testing.T) {
	pngBytes := []byte{0x89, 0x50, 0x4E, 0x47}

	type sweepOutcome struct {
		StoredTokenCount int
		FirstLookupErr   error
	}

	testCases := []struct {
		name string
		ttl  time.Duration
		want sweepOutcome
	}{
		{
			name: "sweeps expired asset tokens on subsequent issue",
			ttl:  0,
			want: sweepOutcome{
				StoredTokenCount: 1,
				FirstLookupErr:   ErrUnknownAsset,
			},
		},
		{
			name: "keeps unexpired asset tokens on subsequent issue",
			ttl:  time.Hour,
			want: sweepOutcome{
				StoredTokenCount: 2,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := NewAssetStore(testAssetBaseURL, tc.ttl)
			firstFileName := strings.TrimPrefix(store.Issue(pngBytes, "image/png", ".png").URL, testAssetBaseURL)
			store.Issue(pngBytes, "image/png", ".png")

			_, lookupErr := store.lookup(firstFileName)
			got := sweepOutcome{
				StoredTokenCount: len(store.entriesByToken),
				FirstLookupErr:   lookupErr,
			}
			if diff := cmp.Diff(tc.want, got, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("Issue() sweep mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAssetStore_ServeAsset(t *testing.T) {
	pngBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	svgBytes := []byte("<svg xmlns=\"http://www.w3.org/2000/svg\"></svg>")

	type serveOutcome struct {
		StatusCode    int
		ContentType   string
		ContentLength string
		Body          []byte
	}

	testCases := []struct {
		name        string
		ttl         time.Duration
		payload     []byte
		contentType string
		extension   string
		method      string
		fileName    func(grant AssetGrant) string
		want        serveOutcome
	}{
		{
			name:        "GET returns PNG payload with image/png headers",
			ttl:         time.Hour,
			payload:     pngBytes,
			contentType: "image/png",
			extension:   ".png",
			method:      http.MethodGet,
			fileName: func(grant AssetGrant) string {
				return strings.TrimPrefix(grant.URL, testAssetBaseURL)
			},
			want: serveOutcome{
				StatusCode:    http.StatusOK,
				ContentType:   "image/png",
				ContentLength: "8",
				Body:          pngBytes,
			},
		},
		{
			name:        "GET returns SVG payload with image/svg+xml headers",
			ttl:         time.Hour,
			payload:     svgBytes,
			contentType: "image/svg+xml",
			extension:   ".svg",
			method:      http.MethodGet,
			fileName: func(grant AssetGrant) string {
				return strings.TrimPrefix(grant.URL, testAssetBaseURL)
			},
			want: serveOutcome{
				StatusCode:    http.StatusOK,
				ContentType:   "image/svg+xml",
				ContentLength: "46",
				Body:          svgBytes,
			},
		},
		{
			name:        "HEAD returns asset headers with empty body",
			ttl:         time.Hour,
			payload:     pngBytes,
			contentType: "image/png",
			extension:   ".png",
			method:      http.MethodHead,
			fileName: func(grant AssetGrant) string {
				return strings.TrimPrefix(grant.URL, testAssetBaseURL)
			},
			want: serveOutcome{
				StatusCode:    http.StatusOK,
				ContentType:   "image/png",
				ContentLength: "8",
				Body:          []byte{},
			},
		},
		{
			name:        "GET unknown token returns 404 JSON error",
			ttl:         time.Hour,
			payload:     pngBytes,
			contentType: "image/png",
			extension:   ".png",
			method:      http.MethodGet,
			fileName: func(grant AssetGrant) string {
				return "missing.png"
			},
			want: serveOutcome{
				StatusCode:  http.StatusNotFound,
				ContentType: "application/json",
				Body:        []byte("{\"error\":\"unknown asset URL\"}\n"),
			},
		},
		{
			name:        "GET expired token returns 410 JSON error",
			ttl:         0,
			payload:     pngBytes,
			contentType: "image/png",
			extension:   ".png",
			method:      http.MethodGet,
			fileName: func(grant AssetGrant) string {
				return strings.TrimPrefix(grant.URL, testAssetBaseURL)
			},
			want: serveOutcome{
				StatusCode:  http.StatusGone,
				ContentType: "application/json",
				Body:        []byte("{\"error\":\"asset URL has expired\"}\n"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := NewAssetStore(testAssetBaseURL, tc.ttl)
			grant := store.Issue(tc.payload, tc.contentType, tc.extension)
			targetFile := tc.fileName(grant)

			req := httptest.NewRequest(tc.method, testAssetBaseURL+targetFile, nil)
			rec := httptest.NewRecorder()

			store.ServeAsset(rec, req, targetFile)

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
				t.Errorf("ServeAsset() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
