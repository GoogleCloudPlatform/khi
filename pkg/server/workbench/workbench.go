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
	"sync"
	"time"

	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6model "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/server/streamingutil"
)

var (
	// ErrStructNotFound indicates that the requested struct ID was not found in the intern pool.
	ErrStructNotFound = errors.New("struct not found")
	// ErrIndexFailed indicates that search index construction encountered a terminal error.
	ErrIndexFailed = errors.New("search index construction failed")
)

// IndexState represents the lifecycle phase of the search index construction.
type IndexState int

const (
	// IndexStateNotStarted indicates that index construction has not started.
	IndexStateNotStarted IndexState = iota
	// IndexStateBuilding indicates that the search index is actively being generated.
	IndexStateBuilding
	// IndexStateReady indicates that the search index is fully built and queryable.
	IndexStateReady
	// IndexStateFailed indicates that search index construction failed.
	IndexStateFailed
)

// IndexProgressEvent encapsulates an index progress notification broadcast to subscribers.
type IndexProgressEvent struct {
	InspectionID       string
	State              IndexState
	ProgressPercentage float64
	Message            string
	Err                error
}

// Workbench represents an active in-memory analysis workspace for an inspection dataset.
type Workbench struct {
	id           string
	inspectionID string
	mu           sync.RWMutex
	closed       bool

	metadataChunks []*khifilev6.MetadataChunk
	internPool     *khifilev6model.InternPool
	styleChunk     *khifilev6.TimelineStyleChunk
	logChunks      []*khifilev6.LogChunk
	timelineChunks []*khifilev6.TimelineChunk
	searchIndex    *SearchIndex

	indexMu          sync.RWMutex
	indexState       IndexState
	indexPercentage  float64
	indexMessage     string
	indexErr         error
	indexSubscribers []chan IndexProgressEvent
	cancelIndex      context.CancelFunc
	indexManager     *InspectionIndexManager
	filterJobs       *streamingutil.AsyncJobManager[*apiv1.FilterProgress, *apiv1.FilterResult]
}

// NewWorkbench creates a new Workbench instance.
func NewWorkbench(id string, inspectionID string) *Workbench {
	return &Workbench{
		id:           id,
		inspectionID: inspectionID,
		internPool:   khifilev6model.NewInternPool(nil),
		filterJobs:   streamingutil.NewAsyncJobManager[*apiv1.FilterProgress, *apiv1.FilterResult](15*time.Second, 1*time.Minute),
	}
}

// FilterJobManager returns the AsyncJobManager tracking timeline filter jobs for this workbench.
func (w *Workbench) FilterJobManager() *streamingutil.AsyncJobManager[*apiv1.FilterProgress, *apiv1.FilterResult] {
	return w.filterJobs
}

// SetIndexManager sets the InspectionIndexManager used to retrieve or wait for background TrigramIndex instances.
func (w *Workbench) SetIndexManager(im *InspectionIndexManager) {
	w.indexManager = im
}

// ID returns the unique workbench identifier.
func (w *Workbench) ID() string {
	return w.id
}

// InspectionID returns the associated inspection identifier.
func (w *Workbench) InspectionID() string {
	return w.inspectionID
}

// IsClosed checks whether the workbench has been closed.
func (w *Workbench) IsClosed() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.closed
}

// ReadStructYAML decodes the interned struct matching the given structID and returns its YAML string representation.
// If the struct YAML is available in SearchIndex.StructYAMLs, it is returned directly from the pre-serialized index.
func (w *Workbench) ReadStructYAML(structID uint32) (string, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.closed {
		return "", ErrWorkbenchClosed
	}

	if w.searchIndex != nil && w.searchIndex.StructYAMLs != nil {
		if yamlStr, ok := w.searchIndex.StructYAMLs[structID]; ok {
			return yamlStr, nil
		}
	}

	if w.internPool == nil {
		return "", ErrStructNotFound
	}

	s := w.internPool.ResolveStructFromID(structID)
	if s == nil {
		return "", ErrStructNotFound
	}

	serializer := khifilev6model.NewDirectYAMLSerializer()
	yamlStr, err := serializer.SerializeStruct(s, w.internPool)
	if err != nil {
		return "", fmt.Errorf("failed to serialize struct to YAML: %w", err)
	}

	return yamlStr, nil
}

// IndexStatus returns the current index construction status snapshot.
func (w *Workbench) IndexStatus() (IndexState, float64, string, error) {
	w.indexMu.RLock()
	defer w.indexMu.RUnlock()
	return w.indexState, w.indexPercentage, w.indexMessage, w.indexErr
}

// SubscribeIndexProgress creates a subscription channel for index progress updates.
// The initial state event is immediately sent to the channel.
func (w *Workbench) SubscribeIndexProgress(ctx context.Context) (<-chan IndexProgressEvent, func()) {
	ch := make(chan IndexProgressEvent, 16)

	w.indexMu.Lock()
	currentState := w.indexState
	currentPct := w.indexPercentage
	currentMsg := w.indexMessage
	currentErr := w.indexErr

	ch <- IndexProgressEvent{
		State:              currentState,
		ProgressPercentage: currentPct,
		Message:            currentMsg,
		Err:                currentErr,
	}

	if currentState != IndexStateReady && currentState != IndexStateFailed {
		w.indexSubscribers = append(w.indexSubscribers, ch)
	}
	w.indexMu.Unlock()

	unsubscribe := func() {
		w.indexMu.Lock()
		defer w.indexMu.Unlock()
		for i, sub := range w.indexSubscribers {
			if sub == ch {
				w.indexSubscribers = append(w.indexSubscribers[:i], w.indexSubscribers[i+1:]...)
				break
			}
		}
	}

	return ch, unsubscribe
}

// AwaitIndex blocks until the search index construction reaches a terminal state (Ready or Failed), or the context is canceled.
func (w *Workbench) AwaitIndex(ctx context.Context) error {
	ch, unsubscribe := w.SubscribeIndexProgress(ctx)
	defer unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-ch:
			if !ok {
				state, _, _, err := w.IndexStatus()
				if state == IndexStateReady {
					return nil
				}
				if state == IndexStateFailed {
					return err
				}
				return fmt.Errorf("index progress subscription closed unexpectedly")
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

// notifyIndexSubscribersLocked broadcasts an index progress event to all active subscribers.
// Caller must hold w.indexMu (either read or write lock).
func (w *Workbench) notifyIndexSubscribersLocked(event IndexProgressEvent) {
	for _, ch := range w.indexSubscribers {
		select {
		case ch <- event:
		default:
		}
	}
}

// StartAsyncIndexing initiates background trigram search index construction if not already started.
func (w *Workbench) StartAsyncIndexing(parentCtx context.Context) {
	w.indexMu.Lock()
	if w.indexState != IndexStateNotStarted {
		w.indexMu.Unlock()
		return
	}
	w.indexState = IndexStateBuilding
	w.indexPercentage = 0.0
	w.indexMessage = "Starting search index construction..."
	ctx, cancel := context.WithCancel(parentCtx)
	w.cancelIndex = cancel
	w.notifyIndexSubscribersLocked(IndexProgressEvent{
		State:              IndexStateBuilding,
		ProgressPercentage: 0.0,
		Message:            "Starting search index construction...",
	})
	w.indexMu.Unlock()

	go func() {
		w.mu.RLock()
		targetIndex := w.searchIndex
		w.mu.RUnlock()

		if targetIndex == nil {
			w.indexMu.Lock()
			w.indexState = IndexStateFailed
			w.indexErr = fmt.Errorf("search index is not initialized")
			w.indexMessage = "Search index is not initialized"
			w.notifyIndexSubscribersLocked(IndexProgressEvent{
				State:              IndexStateFailed,
				ProgressPercentage: 0.0,
				Message:            w.indexMessage,
				Err:                w.indexErr,
			})
			for _, ch := range w.indexSubscribers {
				close(ch)
			}
			w.indexSubscribers = nil
			w.indexMu.Unlock()
			return
		}

		err := w.BuildAsyncIndexesWithProgress(ctx, targetIndex, func(stage apiv1.OpenWorkbenchResponse_Stage, progressPercentage float64, message string) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			w.indexMu.Lock()
			w.indexPercentage = progressPercentage
			w.indexMessage = message
			w.notifyIndexSubscribersLocked(IndexProgressEvent{
				State:              IndexStateBuilding,
				ProgressPercentage: progressPercentage,
				Message:            message,
			})
			w.indexMu.Unlock()
			return nil
		})

		if err != nil {
			w.indexMu.Lock()
			w.indexState = IndexStateFailed
			w.indexErr = err
			w.indexMessage = fmt.Sprintf("Index build failed: %v", err)
			w.notifyIndexSubscribersLocked(IndexProgressEvent{
				State:              IndexStateFailed,
				ProgressPercentage: w.indexPercentage,
				Message:            w.indexMessage,
				Err:                err,
			})
			for _, ch := range w.indexSubscribers {
				close(ch)
			}
			w.indexSubscribers = nil
			w.indexMu.Unlock()
			return
		}

		w.indexMu.Lock()
		w.indexState = IndexStateReady
		w.indexPercentage = 100.0
		w.indexMessage = "Search index ready."

		w.notifyIndexSubscribersLocked(IndexProgressEvent{
			State:              IndexStateReady,
			ProgressPercentage: 100.0,
			Message:            "Search index ready.",
		})
		for _, ch := range w.indexSubscribers {
			close(ch)
		}
		w.indexSubscribers = nil
		w.indexMu.Unlock()
	}()
}

// Close marks the workbench as closed and releases in-memory chunk references.
func (w *Workbench) Close() {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return
	}
	w.closed = true
	w.metadataChunks = nil
	w.internPool = nil
	w.styleChunk = nil
	w.logChunks = nil
	w.timelineChunks = nil
	w.searchIndex = nil
	w.mu.Unlock()

	w.indexMu.Lock()
	if w.cancelIndex != nil {
		w.cancelIndex()
		w.cancelIndex = nil
	}
	for _, ch := range w.indexSubscribers {
		close(ch)
	}
	w.indexSubscribers = nil
	w.indexMu.Unlock()

	if w.filterJobs != nil {
		w.filterJobs.Close()
	}
}
