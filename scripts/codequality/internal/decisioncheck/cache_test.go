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

package decisioncheck

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestDecisionCacheRoundTripAndKeyInvalidation verifies that SQLite cache keys change when question text changes and round-trip stored responses.
func TestDecisionCacheRoundTripAndKeyInvalidation(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "nested", "decisions.db")
	cache, err := OpenDecisionCache(ctx, dbPath)
	if err != nil {
		t.Fatalf("OpenDecisionCache() unexpected error: %v", err)
	}
	defer cache.Close()
	reqV1 := DecisionRequest{
		Model: "gemini-decision-1",
		Input: "input-v1",
		Questions: map[string]WireQuestion{
			"q1": {Type: QuestionTypeNoul, Instructions: "Check v1"},
		},
	}
	reqV2 := DecisionRequest{
		Model: "gemini-decision-1",
		Input: "input-v1",
		Questions: map[string]WireQuestion{
			"q1": {Type: QuestionTypeNoul, Instructions: "Check v2"},
		},
	}
	pTrue := 0.91
	wantResp := DecisionResponse{
		Answers: map[string]WireAnswer{
			"q1": {Noul: &pTrue, Confidence: 0.95},
		},
	}

	keyV1, err := ComputeBatchCacheKey(reqV1)
	if err != nil {
		t.Fatalf("ComputeBatchCacheKey(reqV1) unexpected error: %v", err)
	}
	keyV2, err := ComputeBatchCacheKey(reqV2)
	if err != nil {
		t.Fatalf("ComputeBatchCacheKey(reqV2) unexpected error: %v", err)
	}
	_, missHit, missErr := cache.Lookup(ctx, keyV1)
	storeErr := cache.Store(ctx, keyV1, wantResp)
	gotResp, hitAfterStore, hitErr := cache.Lookup(ctx, keyV1)

	if keyV1 == keyV2 {
		t.Errorf("ComputeBatchCacheKey() produced identical keys for distinct question instructions")
	}
	if missErr != nil || missHit {
		t.Errorf("Lookup(before Store) = (hit=%v, err=%v), want (false, nil)", missHit, missErr)
	}
	if storeErr != nil {
		t.Fatalf("Store() unexpected error: %v", storeErr)
	}
	if hitErr != nil || !hitAfterStore {
		t.Fatalf("Lookup(after Store) = (hit=%v, err=%v), want (true, nil)", hitAfterStore, hitErr)
	}
	if diff := cmp.Diff(wantResp, gotResp); diff != "" {
		t.Errorf("cached response mismatch (-want +got):\n%s", diff)
	}
}

func TestDefaultCachePath(t *testing.T) {
	testCases := []struct {
		name       string
		getenv     func(string) string
		wantExact  string
		wantSuffix string
	}{
		// Uses CODEQUALITY_DECISIONS_CACHE_DIR directly when configured in the environment.
		{
			name: "explicit cache directory override",
			getenv: func(key string) string {
				if key == "CODEQUALITY_DECISIONS_CACHE_DIR" {
					return "/tmp/custom-decisions"
				}
				return ""
			},
			wantExact: filepath.Join("/tmp/custom-decisions", "decisions_v1.db"),
		},
		// Falls back to the user cache directory under khi/decisions/decisions_v1.db when unset.
		{
			name:       "default user cache directory suffix",
			getenv:     nil,
			wantSuffix: filepath.Join("khi", "decisions", "decisions_v1.db"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotPath, err := DefaultCachePath(tc.getenv)

			if err != nil {
				t.Fatalf("DefaultCachePath() unexpected error: %v", err)
			}
			if tc.wantExact != "" && gotPath != tc.wantExact {
				t.Errorf("DefaultCachePath() = %q, want %q", gotPath, tc.wantExact)
			}
			if tc.wantSuffix != "" && !strings.HasSuffix(gotPath, tc.wantSuffix) {
				t.Errorf("DefaultCachePath() = %q, want suffix %q", gotPath, tc.wantSuffix)
			}
		})
	}
}

// TestDecisionCacheCloseClosesDatabase verifies that Close shuts down the SQLite connection so subsequent lookups fail.
func TestDecisionCacheCloseClosesDatabase(t *testing.T) {
	t.Chdir(t.TempDir())
	ctx := context.Background()
	cache, err := OpenDecisionCache(ctx, "")
	if err != nil {
		t.Fatalf("OpenDecisionCache() unexpected error: %v", err)
	}

	closeErr := cache.Close()

	if closeErr != nil {
		t.Fatalf("Close() unexpected error: %v", closeErr)
	}
	if _, _, lookupErr := cache.Lookup(ctx, CacheKey{}); lookupErr == nil {
		t.Errorf("Lookup() after Close() = nil error, want closed-database error")
	}
}

// TestDecisionCacheNilClose verifies that Close on a nil DecisionCache returns nil without panicking.
func TestDecisionCacheNilClose(t *testing.T) {
	var nilCache *DecisionCache

	err := nilCache.Close()

	if err != nil {
		t.Errorf("(*DecisionCache)(nil).Close() = %v, want nil", err)
	}
}

// TestDecisionCacheNilStore verifies that Store on a nil DecisionCache is a no-op returning nil.
func TestDecisionCacheNilStore(t *testing.T) {
	ctx := context.Background()
	var nilCache *DecisionCache

	err := nilCache.Store(ctx, CacheKey{}, DecisionResponse{})

	if err != nil {
		t.Errorf("(*DecisionCache)(nil).Store() = %v, want nil", err)
	}
}
