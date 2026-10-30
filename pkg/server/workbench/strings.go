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

package workbench

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6model "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
)

const (
	defaultStringSearchLimit = 200
	maxStringSearchLimit     = 1000
)

// ErrInvalidRegexQuery indicates that a string pool search query failed RE2 regular expression compilation.
var ErrInvalidRegexQuery = errors.New("invalid regular expression query")

var quotedQueryLiteralPattern = regexp.MustCompile(`"([^"\\]{2,})"|'([^'\\]{2,})'`)

// StringPoolIndex maintains a deduplicated, priority-ordered list of unique strings from a Workbench dataset
// alongside an asynchronously built TrigramIndex with linear-scan fallback.
type StringPoolIndex struct {
	values []string

	mu                 sync.RWMutex
	trigramIndex       *cel.TrigramIndex
	state              IndexState
	progressPercentage float64
	message            string
	err                error
	subscribers        []chan IndexProgressEvent
	closed             bool
}

// NewStringPoolIndex constructs a StringPoolIndex over the provided priority-ordered unique strings.
func NewStringPoolIndex(values []string) *StringPoolIndex {
	return &StringPoolIndex{
		values:  values,
		state:   IndexStateNotStarted,
		message: "String pool index not started.",
	}
}

// StartAsyncTrigramBuild launches background trigram index construction if it has not yet started.
func (s *StringPoolIndex) StartAsyncTrigramBuild(parentCtx context.Context) {
	s.mu.Lock()
	if s.closed || s.state != IndexStateNotStarted {
		s.mu.Unlock()
		return
	}
	s.state = IndexStateBuilding
	s.progressPercentage = 0.0
	s.message = "Starting string pool index construction..."
	s.notifySubscribersLocked(IndexProgressEvent{
		State:              IndexStateBuilding,
		ProgressPercentage: 0.0,
		Message:            s.message,
	})
	s.mu.Unlock()

	go func() {
		idx := cel.NewTrigramIndex()
		err := idx.BuildFromStrings(parentCtx, s.values, func(progress float64, msg string) error {
			select {
			case <-parentCtx.Done():
				return parentCtx.Err()
			default:
			}
			pct := progress * 100.0
			if pct > 99.0 && progress < 1.0 {
				pct = 99.0
			}
			s.mu.Lock()
			if s.closed {
				s.mu.Unlock()
				return context.Canceled
			}
			s.progressPercentage = pct
			s.message = msg
			s.notifySubscribersLocked(IndexProgressEvent{
				State:              IndexStateBuilding,
				ProgressPercentage: pct,
				Message:            msg,
			})
			s.mu.Unlock()
			return nil
		})

		s.mu.Lock()
		defer s.mu.Unlock()
		if s.closed {
			return
		}
		if err != nil {
			s.state = IndexStateFailed
			s.err = err
			s.message = fmt.Sprintf("String pool index build failed: %v", err)
			s.notifySubscribersLocked(IndexProgressEvent{
				State:              IndexStateFailed,
				ProgressPercentage: s.progressPercentage,
				Message:            s.message,
				Err:                err,
			})
			for _, ch := range s.subscribers {
				close(ch)
			}
			s.subscribers = nil
			return
		}

		s.trigramIndex = idx
		s.state = IndexStateReady
		s.progressPercentage = 100.0
		s.message = "String pool index ready."
		s.notifySubscribersLocked(IndexProgressEvent{
			State:              IndexStateReady,
			ProgressPercentage: 100.0,
			Message:            s.message,
		})
		for _, ch := range s.subscribers {
			close(ch)
		}
		s.subscribers = nil
	}()
}

// Status returns a snapshot of the current string pool trigram index construction state.
func (s *StringPoolIndex) Status() (IndexState, float64, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state, s.progressPercentage, s.message, s.err
}

// SubscribeProgress registers a channel receiving string pool index progress updates, emitting the current snapshot immediately.
func (s *StringPoolIndex) SubscribeProgress(ctx context.Context) (<-chan IndexProgressEvent, func()) {
	ch := make(chan IndexProgressEvent, 16)

	s.mu.Lock()
	initial := IndexProgressEvent{
		State:              s.state,
		ProgressPercentage: s.progressPercentage,
		Message:            s.message,
		Err:                s.err,
	}
	ch <- initial
	active := !s.closed && s.state != IndexStateReady && s.state != IndexStateFailed
	if active {
		s.subscribers = append(s.subscribers, ch)
	} else {
		close(ch)
	}
	s.mu.Unlock()

	var once sync.Once
	stopCh := make(chan struct{})
	unsubscribe := func() {
		once.Do(func() {
			close(stopCh)
			if !active {
				return
			}
			s.mu.Lock()
			defer s.mu.Unlock()
			for i, sub := range s.subscribers {
				if sub == ch {
					s.subscribers = append(s.subscribers[:i], s.subscribers[i+1:]...)
					break
				}
			}
		})
	}

	if active {
		go func() {
			select {
			case <-ctx.Done():
				unsubscribe()
			case <-stopCh:
			}
		}()
	}

	return ch, unsubscribe
}

// AwaitReady blocks until the background trigram index reaches Ready or Failed, or ctx is canceled.
func (s *StringPoolIndex) AwaitReady(ctx context.Context) error {
	ch, unsubscribe := s.SubscribeProgress(ctx)
	defer unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-ch:
			if !ok {
				state, _, _, err := s.Status()
				if state == IndexStateReady {
					return nil
				}
				if state == IndexStateFailed {
					return err
				}
				return fmt.Errorf("string pool index subscription closed before completion")
			}
			if ev.State == IndexStateReady {
				return nil
			}
			if ev.State == IndexStateFailed {
				return ev.Err
			}
		}
	}
}

// Close releases the trigram index and closes active progress subscribers.
func (s *StringPoolIndex) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	s.trigramIndex = nil
	for _, ch := range s.subscribers {
		close(ch)
	}
	s.subscribers = nil
}

func (s *StringPoolIndex) notifySubscribersLocked(event IndexProgressEvent) {
	for _, ch := range s.subscribers {
		select {
		case ch <- event:
		default:
		}
	}
}

// Search filters indexed strings by a case-insensitive RE2 regular expression with pagination.
// It uses the Roaring Bitmap trigram index once ready to prune candidates, and falls back to linear scan otherwise.
func (s *StringPoolIndex) Search(query string, offset int, limit int) ([]string, int, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = defaultStringSearchLimit
	} else if limit > maxStringSearchLimit {
		limit = maxStringSearchLimit
	}

	if query == "" {
		total := len(s.values)
		if offset >= total {
			return []string{}, total, nil
		}
		end := offset + limit
		if end > total {
			end = total
		}
		out := make([]string, end-offset)
		copy(out, s.values[offset:end])
		return out, total, nil
	}

	re, err := cel.CompileCaseInsensitiveRegex(query)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %w", ErrInvalidRegexQuery, err)
	}

	s.mu.RLock()
	tri := s.trigramIndex
	s.mu.RUnlock()

	if tri != nil {
		if bm := tri.FindCandidateLogs(query); bm != nil {
			page := make([]string, 0, min(limit, int(bm.GetCardinality())))
			total := 0
			it := bm.Iterator()
			for it.HasNext() {
				idx := int(it.Next())
				if idx >= len(s.values) {
					continue
				}
				if re.MatchString(s.values[idx]) {
					if total >= offset && len(page) < limit {
						page = append(page, s.values[idx])
					}
					total++
				}
			}
			return page, total, nil
		}
	}

	page := make([]string, 0, min(limit, len(s.values)))
	total := 0
	for _, val := range s.values {
		if re.MatchString(val) {
			if total >= offset && len(page) < limit {
				page = append(page, val)
			}
			total++
		}
	}
	return page, total, nil
}

// collectWorkbenchStrings gathers all unique non-empty strings from a Workbench dataset ordered by masking priority.
func collectWorkbenchStrings(
	searchIndex *SearchIndex,
	internPool *khifilev6model.ReadonlyInternPool,
	metadataChunks []*khifilev6.MetadataChunk,
) []string {
	seen := make(map[string]struct{})
	var ordered []string

	addSegment := func(s string) {
		if s == "" || strings.TrimSpace(s) == "" {
			return
		}
		if _, exists := seen[s]; exists {
			return
		}
		seen[s] = struct{}{}
		ordered = append(ordered, s)
	}

	addValue := func(val string) {
		if val == "" {
			return
		}
		if !strings.Contains(val, "\x00") {
			addSegment(val)
			return
		}
		for _, seg := range strings.Split(val, "\x00") {
			addSegment(seg)
		}
	}

	// Tier 1a: Timeline names and Revision principals.
	if searchIndex != nil {
		for _, tl := range searchIndex.Timelines {
			if tl == nil {
				continue
			}
			addValue(tl.Name)
		}
		if internPool != nil {
			for _, tl := range searchIndex.Timelines {
				if tl == nil {
					continue
				}
				for _, rev := range tl.Revisions {
					if rev.PrincipalStringID != 0 {
						addValue(internPool.ResolveStringFromID(rev.PrincipalStringID))
					}
				}
			}
		}
	}

	// Tier 1b: Header metadata and quoted literals from inspection queries.
	for _, metaChunk := range metadataChunks {
		for _, item := range metaChunk.GetMetadata() {
			addValue(item.GetHeader().GetInspectionName())
			addValue(item.GetHeader().GetSuggestedFilename())
			for _, q := range item.GetQuery().GetQueries() {
				addValue(q.GetName())
				matches := quotedQueryLiteralPattern.FindAllStringSubmatch(q.GetQuery(), -1)
				for _, match := range matches {
					if match[1] != "" {
						addValue(match[1])
					} else if match[2] != "" {
						addValue(match[2])
					}
				}
			}
		}
	}

	// Tier 2 & 3: All remaining client strings, server strings, and full query expressions.
	if internPool != nil {
		for _, s := range internPool.ClientStrings() {
			addValue(s)
		}
		for _, s := range internPool.ServerStrings() {
			addValue(s)
		}
	}
	for _, metaChunk := range metadataChunks {
		for _, item := range metaChunk.GetMetadata() {
			for _, q := range item.GetQuery().GetQueries() {
				addValue(q.GetQuery())
			}
		}
	}

	return ordered
}

// getOrInitStringPoolIndex lazily initializes and starts background trigram indexing for the workbench string pool on first access.
func (w *Workbench) getOrInitStringPoolIndex() (*StringPoolIndex, error) {
	w.mu.RLock()
	if w.closed {
		w.mu.RUnlock()
		return nil, ErrWorkbenchClosed
	}
	searchIndex := w.searchIndex
	internPool := w.internPool
	metadataChunks := w.metadataChunks
	w.mu.RUnlock()

	w.stringPoolMu.Lock()
	defer w.stringPoolMu.Unlock()

	if w.IsClosed() {
		return nil, ErrWorkbenchClosed
	}
	if w.stringPoolIndex != nil {
		return w.stringPoolIndex, nil
	}

	values := collectWorkbenchStrings(searchIndex, internPool, metadataChunks)
	idx := NewStringPoolIndex(values)
	ctx, cancel := context.WithCancel(context.Background())
	w.stringPoolIndex = idx
	w.cancelStringPool = cancel
	idx.StartAsyncTrigramBuild(ctx)
	return idx, nil
}

// HasInitializedStringPoolIndex reports whether the on-demand string pool index has been initialized for this workbench.
func (w *Workbench) HasInitializedStringPoolIndex() bool {
	w.stringPoolMu.Lock()
	defer w.stringPoolMu.Unlock()
	return w.stringPoolIndex != nil
}

// ListStrings returns a paginated slice of unique strings matching query and the total match count.
func (w *Workbench) ListStrings(query string, offset int, limit int) ([]string, int, error) {
	idx, err := w.getOrInitStringPoolIndex()
	if err != nil {
		return nil, 0, err
	}
	return idx.Search(query, offset, limit)
}

// SubscribeStringPoolIndexProgress initializes the string pool index on demand and subscribes to its construction progress.
func (w *Workbench) SubscribeStringPoolIndexProgress(ctx context.Context) (<-chan IndexProgressEvent, func(), error) {
	idx, err := w.getOrInitStringPoolIndex()
	if err != nil {
		return nil, nil, err
	}
	ch, unsubscribe := idx.SubscribeProgress(ctx)
	return ch, unsubscribe, nil
}

// StringPoolIndexStatus initializes the string pool index on demand and returns a snapshot of its construction progress.
func (w *Workbench) StringPoolIndexStatus() (IndexState, float64, string, error) {
	idx, err := w.getOrInitStringPoolIndex()
	if err != nil {
		return IndexStateFailed, 0, err.Error(), err
	}
	return idx.Status()
}
