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

package cel

import (
	"context"
	"regexp/syntax"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/GoogleCloudPlatform/khi/pkg/common/worker"
	"github.com/RoaringBitmap/roaring/v2"
)

// TrigramProgressCallback receives streaming progress updates during Trigram index building.
type TrigramProgressCallback = worker.ProgressCallback

// TrigramIndex provides fast regular expression and substring candidate search over interned struct IDs using Roaring Bitmaps.
type TrigramIndex struct {
	// trigramToBitmap maps each lowercase 3-rune string to a Roaring Bitmap of StructIDs containing it.
	trigramToBitmap map[string]*roaring.Bitmap
	// allStructIDs contains all indexed StructIDs.
	allStructIDs *roaring.Bitmap
	// mu protects candidateCache.
	mu sync.RWMutex
	// candidateCache stores cached candidate query results (*roaring.Bitmap or nil) for concurrent evaluators.
	candidateCache map[string]*roaring.Bitmap
}

// NewTrigramIndex creates an empty TrigramIndex.
func NewTrigramIndex() *TrigramIndex {
	return &TrigramIndex{
		trigramToBitmap: make(map[string]*roaring.Bitmap),
		allStructIDs:    roaring.NewBitmap(),
		candidateCache:  make(map[string]*roaring.Bitmap),
	}
}

type yamlEntry struct {
	id   uint32
	yaml string
}

// processYAMLEntryChunk processes a slice of yamlEntry, returning worker-local trigram-to-StructIDs mapping.
func processYAMLEntryChunk(chunk []yamlEntry, onProcessed func(int)) map[string][]uint32 {
	localTrigrams := make(map[string][]uint32)
	var buf [12]byte

	for _, entry := range chunk {
		yamlStr := entry.yaml
		id := entry.id
		if len(yamlStr) < 3 {
			onProcessed(1)
			continue
		}

		var r0, r1, r2 rune
		var count int

		for offset := 0; offset < len(yamlStr); {
			r, size := utf8.DecodeRuneInString(yamlStr[offset:])
			offset += size
			rLower := unicode.ToLower(r)

			if count == 0 {
				r0 = rLower
				count = 1
				continue
			}
			if count == 1 {
				r1 = rLower
				count = 2
				continue
			}

			r2 = rLower
			n := utf8.EncodeRune(buf[0:], r0)
			n += utf8.EncodeRune(buf[n:], r1)
			n += utf8.EncodeRune(buf[n:], r2)

			triKey := string(buf[:n])
			if ids, exists := localTrigrams[triKey]; exists {
				if len(ids) == 0 || ids[len(ids)-1] != id {
					localTrigrams[triKey] = append(ids, id)
				}
			} else {
				localTrigrams[triKey] = []uint32{id}
			}

			r0 = r1
			r1 = r2
		}

		onProcessed(1)
	}

	return localTrigrams
}

// mergeTrigramChunk merges worker-local struct ID slices for the given slice of trigrams into Roaring Bitmaps.
func mergeTrigramChunk(chunk []string, results []map[string][]uint32, onProcessed func(int)) map[string]*roaring.Bitmap {
	localMerged := make(map[string]*roaring.Bitmap, len(chunk))
	for _, tri := range chunk {
		totalCount := 0
		for _, res := range results {
			totalCount += len(res[tri])
		}
		if totalCount == 0 {
			onProcessed(1)
			continue
		}

		allIDs := make([]uint32, 0, totalCount)
		for _, res := range results {
			if ids, ok := res[tri]; ok {
				allIDs = append(allIDs, ids...)
			}
		}

		bm := roaring.NewBitmap()
		bm.AddMany(allIDs)
		localMerged[tri] = bm
		onProcessed(1)
	}
	return localMerged
}

// BuildFromStructYAMLs indexes trigrams from pre-serialized struct YAML strings concurrently using Roaring Bitmaps.
func (t *TrigramIndex) BuildFromStructYAMLs(ctx context.Context, structYAMLs map[uint32]string, onProgress TrigramProgressCallback) error {
	if len(structYAMLs) == 0 {
		return nil
	}

	entries := make([]yamlEntry, 0, len(structYAMLs))
	for id, yaml := range structYAMLs {
		if id == 0 {
			continue
		}
		t.allStructIDs.Add(id)
		entries = append(entries, yamlEntry{id: id, yaml: yaml})
	}

	if len(entries) == 0 {
		return nil
	}

	// Phase 1: Worker-Local Construction (0 locks)
	results, err := worker.ParallelChunkMap(
		ctx,
		entries,
		func(ctx context.Context, workerIdx int, chunk []yamlEntry, onProcessed func(int)) (map[string][]uint32, error) {
			return processYAMLEntryChunk(chunk, onProcessed), nil
		},
		onProgress,
		worker.ProgressOptions{
			MessageFmt:  "Building text search index (%d/%d)...",
			MinProgress: 0.0,
			MaxProgress: 0.70,
		},
	)
	if err != nil {
		return err
	}

	// Collect unique trigram keys from all workers
	uniqueTrigramMap := make(map[string]struct{})
	for _, res := range results {
		for tri := range res {
			uniqueTrigramMap[tri] = struct{}{}
		}
	}

	uniqueTrigrams := make([]string, 0, len(uniqueTrigramMap))
	for tri := range uniqueTrigramMap {
		uniqueTrigrams = append(uniqueTrigrams, tri)
	}

	if len(uniqueTrigrams) == 0 {
		return nil
	}

	// Phase 2: Parallel Partition Merge (0 locks)
	mergedBitmaps, err := worker.ParallelChunkMap(
		ctx,
		uniqueTrigrams,
		func(ctx context.Context, workerIdx int, chunk []string, onProcessed func(int)) (map[string]*roaring.Bitmap, error) {
			return mergeTrigramChunk(chunk, results, onProcessed), nil
		},
		onProgress,
		worker.ProgressOptions{
			MessageFmt:  "Finalizing text search index (%d/%d)...",
			MinProgress: 0.70,
			MaxProgress: 1.00,
		},
	)
	if err != nil {
		return err
	}

	for _, localMerged := range mergedBitmaps {
		for tri, bm := range localMerged {
			t.trigramToBitmap[tri] = bm
		}
	}

	return nil
}

// FindCandidateStructs returns a Roaring Bitmap containing candidate StructIDs whose serialized YAML could match the regex pattern.
// If the regex is unconstrained by trigrams (e.g. wildcards or <3 char literals), it returns nil, meaning all structs are candidates.
// If the pattern cannot match any indexed struct, it returns an empty Roaring Bitmap.
func (t *TrigramIndex) FindCandidateStructs(pattern string) *roaring.Bitmap {
	if t == nil {
		return roaring.NewBitmap()
	}

	t.mu.RLock()
	if cached, ok := t.candidateCache[pattern]; ok {
		t.mu.RUnlock()
		return cached
	}
	t.mu.RUnlock()

	t.mu.Lock()
	defer t.mu.Unlock()
	if cached, ok := t.candidateCache[pattern]; ok {
		return cached
	}

	syn, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		empty := roaring.NewBitmap()
		t.candidateCache[pattern] = empty
		return empty
	}

	syn = syn.Simplify()
	query := RegexToTrigramQuery(syn).Simplify()
	candidateBitmap := evalTrigramQuery(query, t.trigramToBitmap)
	t.candidateCache[pattern] = candidateBitmap
	return candidateBitmap
}

// evalTrigramQuery evaluates a TrigramQuery against the index bitmaps to produce a candidate *roaring.Bitmap.
// If the query does not constrain the set of matching structs (AllQuery), it returns nil (Universe).
func evalTrigramQuery(q TrigramQuery, trigramToBitmap map[string]*roaring.Bitmap) *roaring.Bitmap {
	if q == nil {
		return nil
	}

	switch node := q.(type) {
	case *AllQuery:
		return nil

	case *NoneQuery:
		return roaring.NewBitmap()

	case *TermQuery:
		bm, ok := trigramToBitmap[node.Term]
		if !ok {
			return roaring.NewBitmap()
		}
		return bm

	case *AndQuery:
		var bms []*roaring.Bitmap
		for _, child := range node.Children {
			bm := evalTrigramQuery(child, trigramToBitmap)
			if bm != nil {
				if bm.IsEmpty() {
					return bm
				}
				bms = append(bms, bm)
			}
		}
		if len(bms) == 0 {
			return nil
		}
		if len(bms) == 1 {
			return bms[0]
		}
		return roaring.FastAnd(bms...)

	case *OrQuery:
		var bms []*roaring.Bitmap
		for _, child := range node.Children {
			bm := evalTrigramQuery(child, trigramToBitmap)
			if bm == nil {
				// Universe OR anything is Universe
				return nil
			}
			bms = append(bms, bm)
		}
		if len(bms) == 0 {
			return nil
		}
		if len(bms) == 1 {
			return bms[0]
		}
		return roaring.FastOr(bms...)

	default:
		return nil
	}
}
