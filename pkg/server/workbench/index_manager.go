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
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6model "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench/cel"
	"golang.org/x/sync/singleflight"
	"google.golang.org/protobuf/proto"
)

// InspectionIndexManager coordinates persistent disk caching, memory caching, and background generation of TrigramIndex instances.
type InspectionIndexManager struct {
	mu               sync.RWMutex
	memoryCache      map[string]*cel.TrigramIndex
	latestEvent      map[string]IndexProgressEvent
	subscribers      map[string][]chan IndexProgressEvent
	dataDir          string
	inspectionServer *coreinspection.InspectionTaskServer
	buildGroup       singleflight.Group
	loadGroup        singleflight.Group
	wg               sync.WaitGroup
}

// NewInspectionIndexManager creates an InspectionIndexManager.
func NewInspectionIndexManager(inspectionServer *coreinspection.InspectionTaskServer, dataDir string) *InspectionIndexManager {
	return &InspectionIndexManager{
		memoryCache:      make(map[string]*cel.TrigramIndex),
		latestEvent:      make(map[string]IndexProgressEvent),
		subscribers:      make(map[string][]chan IndexProgressEvent),
		dataDir:          dataDir,
		inspectionServer: inspectionServer,
	}
}

// GetTrigramIndex returns the cached TrigramIndex if available in memory or fast-loaded from disk.
func (m *InspectionIndexManager) GetTrigramIndex(inspectionID string) (*cel.TrigramIndex, bool) {
	m.mu.RLock()
	if idx, ok := m.memoryCache[inspectionID]; ok && idx != nil {
		m.mu.RUnlock()
		return idx, true
	}
	m.mu.RUnlock()

	// Use singleflight to deduplicate concurrent disk loads of the same trigram index.
	val, err, _ := m.loadGroup.Do(inspectionID, func() (any, error) {
		m.mu.RLock()
		if idx, ok := m.memoryCache[inspectionID]; ok && idx != nil {
			m.mu.RUnlock()
			return idx, nil
		}
		m.mu.RUnlock()

		idx, err := m.loadTrigramIndexFromDisk(inspectionID)
		if err != nil {
			return nil, err
		}
		m.mu.Lock()
		m.memoryCache[inspectionID] = idx
		m.latestEvent[inspectionID] = IndexProgressEvent{
			InspectionID:       inspectionID,
			State:              IndexStateReady,
			ProgressPercentage: 100,
			Message:            "Text search index ready.",
		}
		m.mu.Unlock()
		return idx, nil
	})

	if err == nil && val != nil {
		return val.(*cel.TrigramIndex), true
	}
	return nil, false
}

// StartAsyncIndexing initiates asynchronous background Trigram index construction if not already built or building.
func (m *InspectionIndexManager) StartAsyncIndexing(ctx context.Context, inspectionID string) {
	if _, ok := m.GetTrigramIndex(inspectionID); ok {
		m.broadcast(IndexProgressEvent{
			InspectionID:       inspectionID,
			State:              IndexStateReady,
			ProgressPercentage: 100,
			Message:            "Text search index ready.",
		})
		return
	}

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		_, _, _ = m.buildGroup.Do(inspectionID, func() (any, error) {
			// Check again under singleflight
			if _, ok := m.GetTrigramIndex(inspectionID); ok {
				m.broadcast(IndexProgressEvent{
					InspectionID:       inspectionID,
					State:              IndexStateReady,
					ProgressPercentage: 100,
					Message:            "Text search index ready.",
				})
				return nil, nil
			}

			m.broadcast(IndexProgressEvent{
				InspectionID:       inspectionID,
				State:              IndexStateBuilding,
				ProgressPercentage: 0,
				Message:            "Starting text search index...",
			})

			reader, err := m.openInspectionReader(inspectionID)
			if err != nil {
				errMsg := fmt.Sprintf("failed to open inspection dataset: %v", err)
				m.broadcast(IndexProgressEvent{
					InspectionID: inspectionID,
					State:        IndexStateFailed,
					Err:          err,
					Message:      errMsg,
				})
				return nil, err
			}
			defer reader.Close()

			onProgress := func(progress float64, msg string) error {
				m.broadcast(IndexProgressEvent{
					InspectionID:       inspectionID,
					State:              IndexStateBuilding,
					ProgressPercentage: progress * 100,
					Message:            msg,
				})
				return nil
			}

			idx, err := m.buildTrigramIndexFromReader(reader, onProgress)
			if err != nil {
				errMsg := fmt.Sprintf("failed to build text search index: %v", err)
				m.broadcast(IndexProgressEvent{
					InspectionID: inspectionID,
					State:        IndexStateFailed,
					Err:          err,
					Message:      errMsg,
				})
				return nil, err
			}

			if err := m.saveTrigramIndexToDisk(inspectionID, idx); err != nil {
				slog.Warn(fmt.Sprintf("failed to save trigram index to disk: %v", err))
			}

			m.mu.Lock()
			m.memoryCache[inspectionID] = idx
			m.mu.Unlock()

			m.broadcast(IndexProgressEvent{
				InspectionID:       inspectionID,
				State:              IndexStateReady,
				ProgressPercentage: 100,
				Message:            "Text search index ready.",
			})
			return idx, nil
		})
	}()
}

// Wait waits for all in-flight asynchronous indexing tasks to complete.
func (m *InspectionIndexManager) Wait() {
	m.wg.Wait()
}

// IndexStatus returns the current index status snapshot for the given inspection ID.
func (m *InspectionIndexManager) IndexStatus(inspectionID string) (IndexState, float64, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if idx, ok := m.memoryCache[inspectionID]; ok && idx != nil {
		return IndexStateReady, 100.0, "Text search index ready.", nil
	}
	if ev, exists := m.latestEvent[inspectionID]; exists {
		return ev.State, ev.ProgressPercentage, ev.Message, ev.Err
	}
	return IndexStateNotStarted, 0.0, "Index not started.", nil
}

// SubscribeIndexProgress returns a channel streaming IndexProgressEvents and a cancel function.
func (m *InspectionIndexManager) SubscribeIndexProgress(ctx context.Context, inspectionID string) (<-chan IndexProgressEvent, func()) {
	ch := make(chan IndexProgressEvent, 16)

	m.mu.Lock()
	// Determine initial event to send
	var initialEvent IndexProgressEvent
	if idx, ok := m.memoryCache[inspectionID]; ok && idx != nil {
		initialEvent = IndexProgressEvent{
			InspectionID:       inspectionID,
			State:              IndexStateReady,
			ProgressPercentage: 100,
			Message:            "Text search index ready.",
		}
	} else if ev, exists := m.latestEvent[inspectionID]; exists {
		initialEvent = ev
	} else {
		initialEvent = IndexProgressEvent{
			InspectionID:       inspectionID,
			State:              IndexStateNotStarted,
			ProgressPercentage: 0,
			Message:            "Index not started.",
		}
	}

	m.subscribers[inspectionID] = append(m.subscribers[inspectionID], ch)
	m.mu.Unlock()

	// Non-blocking initial emit
	select {
	case ch <- initialEvent:
	default:
	}

	stopCh := make(chan struct{})
	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			close(stopCh)
			m.mu.Lock()
			defer m.mu.Unlock()
			subs := m.subscribers[inspectionID]
			for i, sub := range subs {
				if sub == ch {
					m.subscribers[inspectionID] = append(subs[:i], subs[i+1:]...)
					break
				}
			}
			close(ch)
		})
	}

	go func() {
		select {
		case <-ctx.Done():
			unsubscribe()
		case <-stopCh:
		}
	}()

	return ch, unsubscribe
}

// DeleteIndex evicts the TrigramIndex from memory cache and removes its disk file.
func (m *InspectionIndexManager) DeleteIndex(inspectionID string) {
	m.mu.Lock()
	delete(m.memoryCache, inspectionID)
	delete(m.latestEvent, inspectionID)
	m.mu.Unlock()

	if m.dataDir != "" {
		filePath := filepath.Join(m.dataDir, inspectionID+".trigram")
		_ = os.Remove(filePath)
	}
}

func (m *InspectionIndexManager) broadcast(event IndexProgressEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latestEvent[event.InspectionID] = event

	for _, ch := range m.subscribers[event.InspectionID] {
		select {
		case ch <- event:
		default:
		}
	}
}

func (m *InspectionIndexManager) openInspectionReader(inspectionID string) (io.ReadCloser, error) {
	if m.inspectionServer != nil {
		currentTask := m.inspectionServer.GetInspection(inspectionID)
		if currentTask != nil {
			result, err := currentTask.Result()
			if err == nil && result != nil && result.ResultStore != nil {
				size, err := result.ResultStore.GetInspectionResultSizeInBytes()
				if err == nil && size > 0 {
					return result.ResultStore.GetRangeReader(0, int64(size))
				}
				if r, err := result.ResultStore.GetReader(); err == nil {
					return r, nil
				}
			}
		}
	}
	if m.dataDir != "" {
		khiPath := filepath.Join(m.dataDir, inspectionID+".khi")
		if f, err := os.Open(khiPath); err == nil {
			return f, nil
		}
	}
	return nil, fmt.Errorf("failed to locate inspection dataset for ID %q", inspectionID)
}

func (m *InspectionIndexManager) buildTrigramIndexFromReader(reader io.Reader, onProgress cel.TrigramProgressCallback) (*cel.TrigramIndex, error) {
	khiReader, err := khifilev6model.NewReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create KHI file reader: %w", err)
	}

	pool := khifilev6model.NewReadonlyInternPool()
	var logs []cel.LogTrigramItem

	for {
		rawChunk, err := khiReader.NextRawChunk()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read next chunk: %w", err)
		}

		decompressed, err := rawChunk.Decompress()
		if err != nil {
			return nil, fmt.Errorf("failed to decompress chunk: %w", err)
		}
		if decompressed.Type == khifilev6model.ChunkTypeInternPool || decompressed.Type == khifilev6model.ChunkTypeServerInternPool {
			var poolChunk khifilev6.InterningPoolChunk
			if err := proto.Unmarshal(decompressed.Data, &poolChunk); err != nil {
				return nil, fmt.Errorf("failed to unmarshal intern pool chunk: %w", err)
			}
			pool.IngestChunk(&poolChunk)
		} else if decompressed.Type == khifilev6model.ChunkTypeLog {
			var logChunk khifilev6.LogChunk
			if err := proto.Unmarshal(decompressed.Data, &logChunk); err != nil {
				return nil, fmt.Errorf("failed to unmarshal log chunk: %w", err)
			}
			for _, l := range logChunk.Logs {
				if l.Id != nil {
					var sumID, bodyID uint32
					if l.SummaryStringId != nil {
						sumID = *l.SummaryStringId
					}
					if l.BodyStructId != nil {
						bodyID = *l.BodyStructId
					}
					logs = append(logs, cel.LogTrigramItem{
						ID:              *l.Id,
						SummaryStringID: sumID,
						BodyStructID:    bodyID,
					})
				}
			}
		}
	}

	idx := cel.NewTrigramIndex()
	if len(logs) > 0 {
		if err := idx.BuildFromLogPool(pool, logs, onProgress); err != nil {
			return nil, fmt.Errorf("failed to build trigram index: %w", err)
		}
		return idx, nil
	}

	structIDs := pool.AllStructIDs()
	if err := idx.BuildFromStructPool(pool, structIDs, onProgress); err != nil {
		return nil, fmt.Errorf("failed to build trigram index: %w", err)
	}
	return idx, nil
}

func (m *InspectionIndexManager) saveTrigramIndexToDisk(inspectionID string, idx *cel.TrigramIndex) error {
	if m.dataDir == "" {
		return nil
	}
	if err := os.MkdirAll(m.dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory %s: %w", m.dataDir, err)
	}
	finalPath := filepath.Join(m.dataDir, inspectionID+".trigram")
	tmpPath := filepath.Join(m.dataDir, inspectionID+".trigram.tmp")

	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temporary trigram file: %w", err)
	}
	defer func() {
		_ = f.Close()
		_ = os.Remove(tmpPath)
	}()

	if _, err := idx.WriteTo(f); err != nil {
		return fmt.Errorf("failed to write trigram index: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close temporary trigram file: %w", err)
	}

	if err := os.Rename(tmpPath, finalPath); err != nil {
		return fmt.Errorf("failed to rename trigram file: %w", err)
	}
	return nil
}

func (m *InspectionIndexManager) loadTrigramIndexFromDisk(inspectionID string) (*cel.TrigramIndex, error) {
	if m.dataDir == "" {
		return nil, os.ErrNotExist
	}
	filePath := filepath.Join(m.dataDir, inspectionID+".trigram")
	tStat, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	// Invalidate if the underlying .khi file was modified after the trigram index was generated
	khiPath := filepath.Join(m.dataDir, inspectionID+".khi")
	if kStat, err := os.Stat(khiPath); err == nil {
		if kStat.ModTime().After(tStat.ModTime()) {
			_ = os.Remove(filePath)
			return nil, fmt.Errorf("stale trigram index for %s: .khi modified %v is newer than .trigram %v", inspectionID, kStat.ModTime(), tStat.ModTime())
		}
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	idx := cel.NewTrigramIndex()
	if _, err := idx.ReadFrom(f); err != nil {
		// If header or data is corrupted, remove stale file
		_ = os.Remove(filePath)
		return nil, fmt.Errorf("failed to read trigram index from %s: %w", filePath, err)
	}
	return idx, nil
}

// InvalidateInspectionIndex evicts the cached TrigramIndex from memory and disk for the inspection.
func (m *InspectionIndexManager) InvalidateInspectionIndex(inspectionID string) {
	m.mu.Lock()
	delete(m.memoryCache, inspectionID)
	delete(m.latestEvent, inspectionID)
	m.mu.Unlock()

	if m.dataDir != "" {
		_ = os.Remove(filepath.Join(m.dataDir, inspectionID+".trigram"))
		_ = os.Remove(filepath.Join(m.dataDir, inspectionID+".trigram.tmp"))
	}
}
