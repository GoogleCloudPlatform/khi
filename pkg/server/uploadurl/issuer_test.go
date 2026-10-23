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

package uploadurl

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

const testBaseURL = "http://127.0.0.1:8080/api/v1/file-upload/"

func TestIssuer_Issue(t *testing.T) {
	issuer := NewIssuer(testBaseURL, 1024, time.Hour)
	before := time.Now()

	grant := issuer.Issue("upload-token-1", "field-1")

	urlToken, found := strings.CutPrefix(grant.URL, testBaseURL)
	if !found || urlToken == "" {
		t.Fatalf("Issue().URL = %q, want %q followed by a URL token", grant.URL, testBaseURL)
	}
	wantGrant := Grant{FieldID: "field-1", MaxSizeBytes: 1024}
	if diff := cmp.Diff(wantGrant, grant, cmpopts.IgnoreFields(Grant{}, "URL", "ExpiresAt")); diff != "" {
		t.Errorf("Issue() grant mismatch (-want +got):\n%s", diff)
	}
	if grant.ExpiresAt.Before(before.Add(time.Hour)) {
		t.Errorf("Issue().ExpiresAt = %v, want at least %v", grant.ExpiresAt, before.Add(time.Hour))
	}

	got, err := issuer.claim(urlToken)
	if err != nil {
		t.Fatalf("claim() returned an unexpected error: %v", err)
	}
	want := grantTarget{uploadTokenID: "upload-token-1", fieldID: "field-1", expiresAt: grant.ExpiresAt}
	if diff := cmp.Diff(want, got, cmp.AllowUnexported(grantTarget{})); diff != "" {
		t.Errorf("claim() mismatch (-want +got):\n%s", diff)
	}
}

func TestIssuer_IssueReturnsDistinctURLs(t *testing.T) {
	issuer := NewIssuer(testBaseURL, 1024, time.Hour)

	first := issuer.Issue("upload-token-1", "field-1")
	second := issuer.Issue("upload-token-1", "field-1")

	if first.URL == second.URL {
		t.Errorf("Issue() returned the same URL %q twice, want distinct URLs", first.URL)
	}
}

func TestIssuer_Claim(t *testing.T) {
	testCases := []struct {
		name          string
		ttl           time.Duration
		urlToken      func(grant Grant) string
		wantErr       error
		wantSecondErr error
	}{
		{
			name: "issued URL token is claimed once",
			ttl:  time.Hour,
			urlToken: func(grant Grant) string {
				return strings.TrimPrefix(grant.URL, testBaseURL)
			},
			wantSecondErr: ErrUnknownURL,
		},
		{
			name: "URL token never issued is unknown",
			ttl:  time.Hour,
			urlToken: func(grant Grant) string {
				return "not-issued"
			},
			wantErr:       ErrUnknownURL,
			wantSecondErr: ErrUnknownURL,
		},
		{
			name: "URL token past its expiry is expired and then forgotten",
			ttl:  0,
			urlToken: func(grant Grant) string {
				return strings.TrimPrefix(grant.URL, testBaseURL)
			},
			wantErr:       ErrExpiredURL,
			wantSecondErr: ErrUnknownURL,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			issuer := NewIssuer(testBaseURL, 1024, tc.ttl)
			urlToken := tc.urlToken(issuer.Issue("upload-token-1", "field-1"))

			if _, err := issuer.claim(urlToken); !errors.Is(err, tc.wantErr) {
				t.Errorf("first claim() error = %v, want %v", err, tc.wantErr)
			}
			if _, err := issuer.claim(urlToken); !errors.Is(err, tc.wantSecondErr) {
				t.Errorf("second claim() error = %v, want %v", err, tc.wantSecondErr)
			}
		})
	}
}

func TestIssuer_IssueSweepsExpiredTokens(t *testing.T) {
	testCases := []struct {
		name         string
		ttl          time.Duration
		wantMapLen   int
		wantClaimErr error
	}{
		{
			name:         "sweeps expired URL tokens",
			ttl:          0,
			wantMapLen:   1,
			wantClaimErr: ErrUnknownURL,
		},
		{
			name:       "keeps unexpired URL tokens",
			ttl:        time.Hour,
			wantMapLen: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			issuer := NewIssuer(testBaseURL, 1024, tc.ttl)
			firstURLToken := strings.TrimPrefix(issuer.Issue("upload-token-1", "field-1").URL, testBaseURL)
			issuer.Issue("upload-token-2", "field-2")

			if got := len(issuer.targetsByURLToken); got != tc.wantMapLen {
				t.Errorf("len(targetsByURLToken) = %d, want %d", got, tc.wantMapLen)
			}
			if _, err := issuer.claim(firstURLToken); !errors.Is(err, tc.wantClaimErr) {
				t.Errorf("claim() of first URL token error = %v, want %v", err, tc.wantClaimErr)
			}
		})
	}
}
