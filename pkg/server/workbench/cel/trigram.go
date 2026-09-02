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
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"regexp/syntax"
	"slices"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/common/worker"
	khifilev6model "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/RoaringBitmap/roaring/v2"
)

const (
	// TrigramBinaryMagicBytes defines the 7-byte magic prefix for serialized TrigramIndex binary files.
	TrigramBinaryMagicBytes = "KHITRIX"
	// TrigramBinaryVersion defines the version byte of the TrigramIndex binary format.
	TrigramBinaryVersion = byte(0x03)
)

var (
	// ErrInvalidTrigramHeader indicates that the binary stream does not begin with valid magic bytes and version.
	ErrInvalidTrigramHeader = errors.New("invalid trigram index binary header")
)

// TrigramProgressCallback receives streaming progress updates during Trigram index building.
type TrigramProgressCallback = worker.ProgressCallback

// TrigramKey represents a 3-rune trigram packed into a single uint64.
// Each Unicode code point (rune) is at most 21 bits (0x000000 to 0x10FFFF).
// Bit layout: [ r0 (21 bits) | r1 (21 bits) | r2 (21 bits) ] using 63 of 64 bits.
type TrigramKey uint64

// MakeTrigramKey packs three lowercase runes into a TrigramKey.
func MakeTrigramKey(r0, r1, r2 rune) TrigramKey {
	return TrigramKey((uint64(r0&0x1fffff) << 42) | (uint64(r1&0x1fffff) << 21) | uint64(r2&0x1fffff))
}

// Unpack returns the three constituent runes of a TrigramKey.
func (k TrigramKey) Unpack() (r0, r1, r2 rune) {
	return rune((k >> 42) & 0x1fffff), rune((k >> 21) & 0x1fffff), rune(k & 0x1fffff)
}

// String returns the 3-rune string representation of the TrigramKey.
func (k TrigramKey) String() string {
	r0, r1, r2 := k.Unpack()
	return string([]rune{r0, r1, r2})
}

// TrigramKeyFromString converts a 3-rune string into a TrigramKey.
// If the string does not contain at least 3 runes, it returns 0, false.
func TrigramKeyFromString(s string) (TrigramKey, bool) {
	runes := []rune(s)
	if len(runes) < 3 {
		return 0, false
	}
	return MakeTrigramKey(runes[0], runes[1], runes[2]), true
}

// LogTrigramItem represents a minimal log record for Trigram indexing.
type LogTrigramItem struct {
	ID              uint32
	SummaryStringID uint32
	BodyStructID    uint32
}

// TrigramIndex provides fast regular expression and substring candidate search over log IDs using Roaring Bitmaps.
type TrigramIndex struct {
	// trigramToBitmap maps each packed 3-rune TrigramKey to a Roaring Bitmap of LogIDs containing it.
	trigramToBitmap map[TrigramKey]*roaring.Bitmap
	// mu protects candidateCache.
	mu sync.RWMutex
	// candidateCache stores cached candidate query results (*roaring.Bitmap or nil) for concurrent evaluators.
	candidateCache map[string]*roaring.Bitmap
}

// NewTrigramIndex creates an empty TrigramIndex.
func NewTrigramIndex() *TrigramIndex {
	return &TrigramIndex{
		trigramToBitmap: make(map[TrigramKey]*roaring.Bitmap),
		candidateCache:  make(map[string]*roaring.Bitmap),
	}
}

var (
	_ io.WriterTo   = (*TrigramIndex)(nil)
	_ io.ReaderFrom = (*TrigramIndex)(nil)
)

// countingWriter wraps an io.Writer and tracks the total number of bytes written.
type countingWriter struct {
	w            io.Writer
	bytesWritten int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.bytesWritten += int64(n)
	return n, err
}

// countingReader wraps an io.Reader and tracks the total number of bytes read.
type countingReader struct {
	r         io.Reader
	bytesRead int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.bytesRead += int64(n)
	return n, err
}

// WriteTo serializes the TrigramIndex into the given io.Writer in compact binary format.
func (t *TrigramIndex) WriteTo(w io.Writer) (int64, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	cw := &countingWriter{w: w}

	header := make([]byte, 8)
	copy(header[:7], TrigramBinaryMagicBytes)
	header[7] = TrigramBinaryVersion
	if _, err := cw.Write(header); err != nil {
		return cw.bytesWritten, fmt.Errorf("failed to write header: %w", err)
	}

	keys := make([]TrigramKey, 0, len(t.trigramToBitmap))
	for k := range t.trigramToBitmap {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	var countBuf [4]byte
	binary.LittleEndian.PutUint32(countBuf[:], uint32(len(keys)))
	if _, err := cw.Write(countBuf[:]); err != nil {
		return cw.bytesWritten, fmt.Errorf("failed to write trigram count: %w", err)
	}

	var keyBuf [8]byte
	for _, key := range keys {
		bm := t.trigramToBitmap[key]
		binary.LittleEndian.PutUint64(keyBuf[:], uint64(key))
		if _, err := cw.Write(keyBuf[:]); err != nil {
			return cw.bytesWritten, fmt.Errorf("failed to write key bytes: %w", err)
		}
		if _, err := bm.WriteTo(cw); err != nil {
			return cw.bytesWritten, fmt.Errorf("failed to write bitmap for trigram %d: %w", key, err)
		}
	}

	return cw.bytesWritten, nil
}

// ReadFrom restores the TrigramIndex from an io.Reader containing binary serialized TrigramIndex data.
func (t *TrigramIndex) ReadFrom(r io.Reader) (int64, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	cr := &countingReader{r: r}

	var header [8]byte
	if _, err := io.ReadFull(cr, header[:]); err != nil {
		return cr.bytesRead, fmt.Errorf("failed to read header: %w", err)
	}
	if string(header[:7]) != TrigramBinaryMagicBytes || header[7] != TrigramBinaryVersion {
		return cr.bytesRead, ErrInvalidTrigramHeader
	}

	var countBuf [4]byte
	if _, err := io.ReadFull(cr, countBuf[:]); err != nil {
		return cr.bytesRead, fmt.Errorf("failed to read trigram count: %w", err)
	}
	numTrigrams := binary.LittleEndian.Uint32(countBuf[:])

	trigramMap := make(map[TrigramKey]*roaring.Bitmap, numTrigrams)
	var keyBuf [8]byte
	for i := uint32(0); i < numTrigrams; i++ {
		if _, err := io.ReadFull(cr, keyBuf[:]); err != nil {
			return cr.bytesRead, fmt.Errorf("failed to read key: %w", err)
		}
		key := TrigramKey(binary.LittleEndian.Uint64(keyBuf[:]))

		bm := roaring.NewBitmap()
		if _, err := bm.ReadFrom(cr); err != nil {
			return cr.bytesRead, fmt.Errorf("failed to read bitmap for trigram %d: %w", key, err)
		}
		trigramMap[key] = bm
	}

	t.trigramToBitmap = trigramMap
	t.candidateCache = make(map[string]*roaring.Bitmap)
	return cr.bytesRead, nil
}

// extractTrigramsToBitmap extracts lowercase 3-rune trigrams from str and directly adds id to localMap[triKey].
func extractTrigramsToBitmap(str string, id uint32, localMap map[TrigramKey]*roaring.Bitmap) {
	if len(str) < 3 {
		return
	}
	var r0, r1, r2 rune
	var count int

	for offset := 0; offset < len(str); {
		r, size := utf8.DecodeRuneInString(str[offset:])
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
		triKey := MakeTrigramKey(r0, r1, r2)
		bm, exists := localMap[triKey]
		if !exists {
			bm = roaring.NewBitmap()
			localMap[triKey] = bm
		}
		bm.Add(id)

		r0 = r1
		r1 = r2
	}
}

type workerTrigramData struct {
	localMap map[TrigramKey]*roaring.Bitmap
}

// BuildFromLogPool indexes trigrams from an InternPool and a slice of LogTrigramItems in streaming parallel chunks.
func (t *TrigramIndex) BuildFromLogPool(pool *khifilev6model.ReadonlyInternPool, logs []LogTrigramItem, onProgress TrigramProgressCallback) error {
	if pool == nil || len(logs) == 0 {
		return nil
	}

	// Step 1: Parallel extraction into worker-local Roaring Bitmaps (0% - 90%)
	workerResults, err := worker.ParallelChunkMap(
		context.Background(),
		logs,
		func(ctx context.Context, workerIdx int, chunk []LogTrigramItem, onProcessed func(int)) (*workerTrigramData, error) {
			data := &workerTrigramData{
				localMap: make(map[TrigramKey]*roaring.Bitmap),
			}
			serializer := khifilev6model.NewDirectYAMLSerializer()

			for _, l := range chunk {
				if l.ID == 0 {
					onProcessed(1)
					continue
				}

				if l.BodyStructID != 0 {
					if yamlStr, err := serializer.SerializeFlatStruct(l.BodyStructID, pool); err == nil {
						extractTrigramsToBitmap(yamlStr, l.ID, data.localMap)
					}
				}

				if l.SummaryStringID != 0 {
					sumStr := pool.ResolveStringFromID(l.SummaryStringID)
					if len(sumStr) >= 3 {
						extractTrigramsToBitmap(sumStr, l.ID, data.localMap)
					}
				}
				onProcessed(1)
			}
			return data, nil
		},
		onProgress,
		worker.ProgressOptions{
			MessageFmt:  "Indexing log bodies and summaries (%d/%d)...",
			MinProgress: 0.0,
			MaxProgress: 0.90,
		},
	)
	if err != nil {
		return err
	}

	// Collect unique trigram keys across all workers.
	keySet := make(map[TrigramKey]struct{})
	for _, wRes := range workerResults {
		if wRes != nil {
			for k := range wRes.localMap {
				keySet[k] = struct{}{}
			}
		}
	}
	allKeys := make([]TrigramKey, 0, len(keySet))
	for k := range keySet {
		allKeys = append(allKeys, k)
	}

	// Step 2: Key-partitioned parallel merge across workers (90% - 100%)
	shardResults, err := worker.ParallelChunkMap(
		context.Background(),
		allKeys,
		func(ctx context.Context, workerIdx int, chunk []TrigramKey, onProcessed func(int)) (map[TrigramKey]*roaring.Bitmap, error) {
			shardMap := make(map[TrigramKey]*roaring.Bitmap, len(chunk))
			for _, key := range chunk {
				var merged *roaring.Bitmap
				for _, wRes := range workerResults {
					if wRes == nil {
						continue
					}
					if bm, ok := wRes.localMap[key]; ok {
						if merged == nil {
							merged = bm
						} else {
							merged.Or(bm)
						}
					}
				}
				if merged != nil {
					merged.RunOptimize()
				}
				shardMap[key] = merged
				onProcessed(1)
			}
			return shardMap, nil
		},
		onProgress,
		worker.ProgressOptions{
			MessageFmt:  "Merging text search index (%d/%d)...",
			MinProgress: 0.50,
			MaxProgress: 1.0,
		},
	)
	if err != nil {
		return err
	}

	finalMap := make(map[TrigramKey]*roaring.Bitmap, len(allKeys))
	for _, shard := range shardResults {
		for k, bm := range shard {
			finalMap[k] = bm
		}
	}

	t.trigramToBitmap = finalMap
	t.candidateCache = make(map[string]*roaring.Bitmap)
	if onProgress != nil {
		_ = onProgress(1.0, "Text search index complete.")
	}
	return nil
}

// BuildFromStructPool indexes trigrams from a ReadonlyInternPool and a slice of StructIDs in streaming chunks.
// It maps each struct ID to itself as a log ID for backward compatibility with struct-level indexing tests.
func (t *TrigramIndex) BuildFromStructPool(pool *khifilev6model.ReadonlyInternPool, structIDs []uint32, onProgress TrigramProgressCallback) error {
	logs := make([]LogTrigramItem, 0, len(structIDs))
	for _, id := range structIDs {
		logs = append(logs, LogTrigramItem{
			ID:           id,
			BodyStructID: id,
		})
	}
	return t.BuildFromLogPool(pool, logs, onProgress)
}

// BuildFromStructYAMLs indexes trigrams from pre-serialized struct YAML strings concurrently using Roaring Bitmaps.
func (t *TrigramIndex) BuildFromStructYAMLs(ctx context.Context, structYAMLs map[uint32]string, onProgress TrigramProgressCallback) error {
	if len(structYAMLs) == 0 {
		return nil
	}

	type structEntry struct {
		id   uint32
		yaml string
	}
	entries := make([]structEntry, 0, len(structYAMLs))
	for id, yaml := range structYAMLs {
		if id == 0 {
			continue
		}
		entries = append(entries, structEntry{id: id, yaml: yaml})
	}

	if len(entries) == 0 {
		return nil
	}

	workerResults, err := worker.ParallelChunkMap(
		ctx,
		entries,
		func(ctx context.Context, workerIdx int, chunk []structEntry, onProcessed func(int)) (map[TrigramKey]*roaring.Bitmap, error) {
			localMap := make(map[TrigramKey]*roaring.Bitmap)
			for _, entry := range chunk {
				extractTrigramsToBitmap(entry.yaml, entry.id, localMap)
				onProcessed(1)
			}
			return localMap, nil
		},
		onProgress,
		worker.ProgressOptions{
			MessageFmt:  "Building text search index (%d/%d)...",
			MinProgress: 0.0,
			MaxProgress: 0.80,
		},
	)
	if err != nil {
		return err
	}

	trigramMap := make(map[TrigramKey]*roaring.Bitmap)
	for _, localMap := range workerResults {
		for tri, bm := range localMap {
			if existing, ok := trigramMap[tri]; ok {
				existing.Or(bm)
			} else {
				trigramMap[tri] = bm
			}
		}
	}
	for _, bm := range trigramMap {
		bm.RunOptimize()
	}

	t.trigramToBitmap = trigramMap
	t.candidateCache = make(map[string]*roaring.Bitmap)
	if onProgress != nil {
		_ = onProgress(1.0, "Text search index complete.")
	}
	return nil
}

// PathToTrigramQuery converts a field path key into a TrigramQuery constraining candidates
// to logs/structs that contain the path's constituent field segments formatted as YAML keys.
// If pathKey is empty or "*", or if none of the segments have at least 3 runes with colon, it returns &AllQuery{}.
func PathToTrigramQuery(pathKey string) TrigramQuery {
	if pathKey == "" || pathKey == "*" {
		return &AllQuery{}
	}

	segments := structured.ParseFieldPath(pathKey)
	var termQueries []TrigramQuery
	seenTerms := make(map[string]struct{})

	for _, seg := range segments {
		if seg == "" {
			continue
		}
		formattedKey := khifilev6model.FormatKeyName(seg) + ":"
		runes := []rune(strings.ToLower(formattedKey))
		if len(runes) < 3 {
			continue
		}
		for i := 0; i <= len(runes)-3; i++ {
			tri := string(runes[i : i+3])
			if _, seen := seenTerms[tri]; seen {
				continue
			}
			seenTerms[tri] = struct{}{}
			termQueries = append(termQueries, &TermQuery{Term: tri})
		}
	}

	if len(termQueries) == 0 {
		return &AllQuery{}
	}
	if len(termQueries) == 1 {
		return termQueries[0]
	}
	return &AndQuery{Children: termQueries}
}

// FindCandidateLogsWithField returns a Roaring Bitmap containing candidate LogIDs whose summary or body
// could match both the field pathKey and the regex pattern.
// If both the field path and pattern are unconstrained, it returns nil (Universe).
// If the combination cannot match any indexed log, it returns an empty Roaring Bitmap.
func (t *TrigramIndex) FindCandidateLogsWithField(pathKey string, pattern string) *roaring.Bitmap {
	if t == nil {
		return roaring.NewBitmap()
	}

	cacheKey := pathKey + "\x00" + pattern

	t.mu.RLock()
	if cached, ok := t.candidateCache[cacheKey]; ok {
		t.mu.RUnlock()
		return cached
	}
	t.mu.RUnlock()

	t.mu.Lock()
	defer t.mu.Unlock()
	if cached, ok := t.candidateCache[cacheKey]; ok {
		return cached
	}

	var patternQuery TrigramQuery = &AllQuery{}
	if pattern != "" && pattern != ".*" {
		syn, err := syntax.Parse(pattern, syntax.Perl)
		if err != nil {
			empty := roaring.NewBitmap()
			t.candidateCache[cacheKey] = empty
			return empty
		}
		patternQuery = RegexToTrigramQuery(syn.Simplify()).Simplify()
	}

	fieldQuery := PathToTrigramQuery(pathKey)

	combinedQuery := (&AndQuery{Children: []TrigramQuery{fieldQuery, patternQuery}}).Simplify()
	candidateBitmap := evalTrigramQuery(combinedQuery, t.trigramToBitmap)
	t.candidateCache[cacheKey] = candidateBitmap
	return candidateBitmap
}

// FindCandidateLogs returns a Roaring Bitmap containing candidate LogIDs whose summary or body could match the regex pattern.
// If the regex is unconstrained by trigrams (e.g. wildcards or <3 char literals), it returns nil, meaning all logs are candidates.
// If the pattern cannot match any indexed log, it returns an empty Roaring Bitmap.
func (t *TrigramIndex) FindCandidateLogs(pattern string) *roaring.Bitmap {
	return t.FindCandidateLogsWithField("*", pattern)
}

// evalTrigramQuery evaluates a TrigramQuery against the index bitmaps to produce a candidate *roaring.Bitmap.
// If the query does not constrain the set of matching structs (AllQuery), it returns nil (Universe).
func evalTrigramQuery(q TrigramQuery, trigramToBitmap map[TrigramKey]*roaring.Bitmap) *roaring.Bitmap {
	if q == nil {
		return nil
	}

	switch node := q.(type) {
	case *AllQuery:
		return nil

	case *NoneQuery:
		return roaring.NewBitmap()

	case *TermQuery:
		triKey, ok := TrigramKeyFromString(node.Term)
		if !ok {
			return roaring.NewBitmap()
		}
		bm, ok := trigramToBitmap[triKey]
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
