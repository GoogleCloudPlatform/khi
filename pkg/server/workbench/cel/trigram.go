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
	"strings"
	"sync"

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

// extractTrigramsFromYAML extracts unique lowercase 3-rune trigrams from the given YAML string.
func extractTrigramsFromYAML(yamlStr string) []string {
	lowerYAML := strings.ToLower(yamlStr)
	runes := []rune(lowerYAML)
	if len(runes) < 3 {
		return nil
	}
	trigramSet := make(map[string]struct{}, len(runes)-2)
	for i := 0; i <= len(runes)-3; i++ {
		trigramSet[string(runes[i:i+3])] = struct{}{}
	}
	trigrams := make([]string, 0, len(trigramSet))
	for tri := range trigramSet {
		trigrams = append(trigrams, tri)
	}
	return trigrams
}

// processYAMLEntryChunk processes a slice of yamlEntry, returning worker-local Roaring Bitmaps.
func processYAMLEntryChunk(chunk []yamlEntry, onProcessed func(int)) map[string]*roaring.Bitmap {
	localBitmaps := make(map[string]*roaring.Bitmap)
	for _, entry := range chunk {
		trigrams := extractTrigramsFromYAML(entry.yaml)
		for _, tri := range trigrams {
			bm, exists := localBitmaps[tri]
			if !exists {
				bm = roaring.NewBitmap()
				localBitmaps[tri] = bm
			}
			bm.Add(entry.id)
		}
		onProcessed(1)
	}
	return localBitmaps
}

// mergeTrigramChunk merges worker-local bitmaps for the given slice of trigrams using roaring.FastOr.
func mergeTrigramChunk(chunk []string, results []map[string]*roaring.Bitmap, onProcessed func(int)) map[string]*roaring.Bitmap {
	localMerged := make(map[string]*roaring.Bitmap, len(chunk))
	for _, tri := range chunk {
		var bms []*roaring.Bitmap
		for _, res := range results {
			if bm, ok := res[tri]; ok {
				bms = append(bms, bm)
			}
		}
		if len(bms) == 1 {
			localMerged[tri] = bms[0]
		} else if len(bms) > 1 {
			localMerged[tri] = roaring.FastOr(bms...)
		}
		onProcessed(1)
	}
	return localMerged
}

// BuildFromStructYAMLs indexes trigrams from pre-serialized struct YAML strings concurrently using Roaring Bitmaps.
func (t *TrigramIndex) BuildFromStructYAMLs(structYAMLs map[uint32]string, onProgress TrigramProgressCallback) error {
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
		context.Background(),
		entries,
		func(ctx context.Context, workerIdx int, chunk []yamlEntry, onProcessed func(int)) (map[string]*roaring.Bitmap, error) {
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
		context.Background(),
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
