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

package simcheck

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestVectorCacheRoundTripChunkingAndPrune(t *testing.T) {
	ctx := context.Background()
	cachePath := filepath.Join(t.TempDir(), "sub", "embeddings_v1.db")
	vc, err := OpenVectorCache(ctx, cachePath)
	if err != nil {
		t.Fatalf("OpenVectorCache() failed: %v", err)
	}

	if err := vc.StoreBatch(ctx, nil); err != nil {
		t.Fatalf("empty StoreBatch() failed: %v", err)
	}

	keyA := ComputeCacheKey("gemini-embedding-2", "ast_norm", 768, "func A() {}")
	keyEmpty := ComputeCacheKey("gemini-embedding-2", "ast_norm", 768, "empty")
	entries := make([]CacheEntry, 0, sqliteQueryChunkSize+5)
	keys := make([]CacheKey, 0, sqliteQueryChunkSize+6)
	entries = append(entries,
		CacheEntry{Key: keyA, Values: []float32{3, 4}},
		CacheEntry{Key: keyEmpty, Values: nil},
	)
	keys = append(keys, keyA, keyA, keyEmpty)

	for i := range sqliteQueryChunkSize + 3 {
		k := ComputeCacheKey("gemini-embedding-2", "ast_norm", 768, fmt.Sprintf("fn_%d", i))
		entries = append(entries, CacheEntry{Key: k, Values: []float32{1, 0}})
		keys = append(keys, k)
	}

	if err := vc.StoreBatch(ctx, entries); err != nil {
		t.Fatalf("StoreBatch() failed: %v", err)
	}
	bad32 := make([]byte, 32)
	bad32[0] = 0xff
	if _, err := vc.db.ExecContext(ctx, "INSERT INTO embeddings (key_hash, vec, updated_at) VALUES (?, ?, 1), (?, ?, 1)",
		[]byte{1, 2}, []byte{0, 0, 0x80, 0x3f}, bad32, []byte{1, 2}); err != nil {
		t.Fatalf("insert malformed rows failed: %v", err)
	}
	var badKey CacheKey
	copy(badKey[:], bad32)
	var shortKey CacheKey
	shortKey[0] = 1
	shortKey[1] = 2
	keys = append(keys, badKey, shortKey)

	if err := vc.Close(); err != nil {
		t.Fatalf("Close() failed: %v", err)
	}
	if _, err := vc.LookupBatch(ctx, []CacheKey{keyA}); err == nil {
		t.Errorf("LookupBatch() after Close() succeeded, want error")
	}

	reloaded, err := OpenVectorCache(ctx, cachePath)
	if err != nil {
		t.Fatalf("reload OpenVectorCache() failed: %v", err)
	}
	defer reloaded.Close()

	emptyLookup, err := reloaded.LookupBatch(ctx, nil)
	if err != nil || len(emptyLookup) != 0 {
		t.Fatalf("LookupBatch(nil) = (%v, %v), want empty", emptyLookup, err)
	}
	if gotDedup := deduplicateKeys([]CacheKey{keyA, keyA}); len(gotDedup) != 1 {
		t.Errorf("len(deduplicateKeys) = %d, want 1", len(gotDedup))
	}

	gotMap, err := reloaded.LookupBatch(ctx, keys)
	if err != nil {
		t.Fatalf("LookupBatch() failed: %v", err)
	}
	if len(gotMap) != sqliteQueryChunkSize+4 {
		t.Errorf("len(LookupBatch) = %d, want %d", len(gotMap), sqliteQueryChunkSize+4)
	}
	if diff := cmp.Diff([]float32{0.6, 0.8}, gotMap[keyA]); diff != "" {
		t.Errorf("gotMap[keyA] mismatch (-want +got):\n%s", diff)
	}

	kept, err := reloaded.PruneOlderThan(ctx, time.Unix(0, 0))
	if err != nil || kept != 0 {
		t.Errorf("PruneOlderThan(epoch) = (%d, %v), want (0, nil)", kept, err)
	}
	pruned, err := reloaded.PruneOlderThan(ctx, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("PruneOlderThan() failed: %v", err)
	}
	if pruned != int64(sqliteQueryChunkSize+6) {
		t.Errorf("PruneOlderThan() = %d, want %d", pruned, sqliteQueryChunkSize+6)
	}
}

// TestInMemoryVectorCache verifies that OpenVectorCache initializes a usable in-memory cache when path is empty.
func TestInMemoryVectorCache(t *testing.T) {
	t.Chdir(t.TempDir())
	ctx := context.Background()

	mem, err := OpenVectorCache(ctx, "")
	if err != nil {
		t.Fatalf("OpenVectorCache(\"\") failed: %v", err)
	}
	defer mem.Close()

	got, lookupErr := mem.LookupBatch(ctx, []CacheKey{{1}})

	if lookupErr != nil {
		t.Fatalf("LookupBatch() failed: %v", lookupErr)
	}
	if len(got) != 0 {
		t.Errorf("LookupBatch() on empty cache returned %d entries, want 0", len(got))
	}
}

// TestVectorCacheNilClose verifies that Close on a nil VectorCache returns nil without panicking.
func TestVectorCacheNilClose(t *testing.T) {
	var nilCache *VectorCache

	err := nilCache.Close()

	if err != nil {
		t.Errorf("(*VectorCache)(nil).Close() = %v, want nil", err)
	}
}

func TestDecodeFloat32Blob(t *testing.T) {
	testCases := []struct {
		name string
		blob []byte
	}{
		// Rejects a nil byte slice because an empty blob cannot encode an embedding vector.
		{
			name: "nil blob returns false",
			blob: nil,
		},
		// Rejects a byte slice whose length is not a multiple of 4 bytes.
		{
			name: "unaligned length returns false",
			blob: []byte{1, 2, 3},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, ok := decodeFloat32Blob(tc.blob)

			if ok {
				t.Errorf("decodeFloat32Blob(%v) = true, want false", tc.blob)
			}
		})
	}
}

func TestDefaultCachePath(t *testing.T) {
	gotCustom, err := DefaultCachePath(func(k string) string {
		if k == "CODEQUALITY_SIMILARITY_CACHE_DIR" {
			return " /tmp/custom-codesim-cache "
		}
		return ""
	})
	if err != nil {
		t.Fatalf("DefaultCachePath(custom) failed: %v", err)
	}
	wantCustom := filepath.Join("/tmp/custom-codesim-cache", "embeddings_v1.db")
	if gotCustom != wantCustom {
		t.Errorf("DefaultCachePath(custom) = %q, want %q", gotCustom, wantCustom)
	}

	gotDefault, err := DefaultCachePath(nil)
	if err != nil {
		t.Fatalf("DefaultCachePath(nil) failed: %v", err)
	}
	if !strings.HasSuffix(filepath.ToSlash(gotDefault), "/khi/codesim/embeddings_v1.db") {
		t.Errorf("DefaultCachePath(nil) = %q, want suffix /khi/codesim/embeddings_v1.db", gotDefault)
	}
}
