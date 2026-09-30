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
	"sync"
	"time"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/GoogleCloudPlatform/khi/pkg/server/streamingutil"
)

var (
	// ErrWorkbenchNotFound indicates that the requested inspection is not loaded, either because it was never opened or because it was evicted.
	ErrWorkbenchNotFound = errors.New("workbench not found or evicted")
	// ErrWorkbenchClosed indicates that the requested workbench has been closed.
	ErrWorkbenchClosed = errors.New("workbench has been closed")
	// ErrInspectionNotFound indicates that no inspection exists for the requested inspection ID.
	ErrInspectionNotFound = errors.New("inspection not found")
)

const (
	// browserSeenWindow is how long a browser heartbeat or request keeps a workbench in use.
	// It spans three frontend heartbeat intervals, so a hidden tab, which stops heartbeating, becomes an eviction candidate soon after.
	browserSeenWindow = 45 * time.Second
	// mcpUseWindow is how long an MCP request keeps a workbench in use.
	// MCP clients call tools intermittently, so the window is longer than the browser one to avoid reloading between calls.
	mcpUseWindow = 5 * time.Minute
)

// Accessor identifies the kind of client that accesses a workbench, so the manager can tell whether the workbench is still in use.
type Accessor int

const (
	// AccessorBrowser is the KHI frontend running in a browser.
	AccessorBrowser Accessor = iota
	// AccessorMCP is an MCP client calling KHI tools.
	AccessorMCP
)

// LoadProgress is a snapshot of the progress of loading an inspection into a workbench.
type LoadProgress struct {
	Stage      apiv1.OpenWorkbenchResponse_Stage
	Percentage float64
	Message    string
}

// LoadHandle tracks one shared load of an inspection into a workbench.
// Every caller that requests the same inspection while it loads receives the same handle.
type LoadHandle struct {
	done chan struct{}
	wb   *Workbench
	err  error

	mu          sync.Mutex
	progress    LoadProgress
	subscribers []chan LoadProgress
}

func newLoadHandle() *LoadHandle {
	return &LoadHandle{
		done: make(chan struct{}),
		progress: LoadProgress{
			Stage:   apiv1.OpenWorkbenchResponse_STAGE_INITIALIZING,
			Message: "Initializing workbench...",
		},
	}
}

// Progress returns the latest loading progress.
func (h *LoadHandle) Progress() LoadProgress {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.progress
}

// Wait blocks until loading finishes or ctx is done, forwarding progress updates to onProgress.
// Cancelling ctx stops only this wait; loading continues for the other callers.
func (h *LoadHandle) Wait(ctx context.Context, onProgress ProgressCallback) (*Workbench, error) {
	ch, unsubscribe := h.subscribe()
	defer unsubscribe()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-h.done:
			if h.err != nil {
				return nil, h.err
			}
			if err := onProgress(apiv1.OpenWorkbenchResponse_STAGE_READY, 100, "Workbench ready."); err != nil {
				return nil, err
			}
			return h.wb, nil
		case p := <-ch:
			if p.Stage != apiv1.OpenWorkbenchResponse_STAGE_READY {
				if err := onProgress(p.Stage, p.Percentage, p.Message); err != nil {
					return nil, err
				}
			}
		}
	}
}

// subscribe registers a channel that receives the current progress immediately and every later update.
// Updates are dropped when the subscriber falls behind, because only the latest progress matters.
func (h *LoadHandle) subscribe() (<-chan LoadProgress, func()) {
	ch := make(chan LoadProgress, 16)
	h.mu.Lock()
	ch <- h.progress
	h.subscribers = append(h.subscribers, ch)
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		for i, sub := range h.subscribers {
			if sub == ch {
				copy(h.subscribers[i:], h.subscribers[i+1:])
				h.subscribers[len(h.subscribers)-1] = nil
				h.subscribers = h.subscribers[:len(h.subscribers)-1]
				break
			}
		}
	}
}

// report records a progress update and broadcasts it to subscribers. It implements ProgressCallback.
func (h *LoadHandle) report(stage apiv1.OpenWorkbenchResponse_Stage, progressPercentage float64, message string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.progress = LoadProgress{Stage: stage, Percentage: progressPercentage, Message: message}
	for _, ch := range h.subscribers {
		select {
		case ch <- h.progress:
		default:
		}
	}
	return nil
}

// finish stores the load result and wakes every waiter.
func (h *LoadHandle) finish(wb *Workbench, err error) {
	if err == nil {
		h.mu.Lock()
		h.progress = LoadProgress{Stage: apiv1.OpenWorkbenchResponse_STAGE_READY, Percentage: 100, Message: "Workbench ready."}
		h.mu.Unlock()
	}
	h.wb = wb
	h.err = err
	close(h.done)
}

// isLoaded reports whether loading finished successfully.
func (h *LoadHandle) isLoaded() bool {
	select {
	case <-h.done:
		return h.err == nil
	default:
		return false
	}
}

// entry is the manager's bookkeeping for one inspection.
type entry struct {
	load *LoadHandle
	// lastAccess is the last time a request used the workbench. It orders eviction candidates.
	lastAccess time.Time
	// browserSeenAt is the last time a browser opened the workbench, sent a heartbeat or made a request.
	browserSeenAt time.Time
	// mcpUsedAt is the last time an MCP request used the workbench.
	mcpUsedAt time.Time
}

// inUse reports whether a browser is still showing the workbench or MCP used it recently.
func (e *entry) inUse(now time.Time) bool {
	return now.Sub(e.browserSeenAt) < browserSeenWindow || now.Sub(e.mcpUsedAt) < mcpUseWindow
}

// WorkbenchManager keeps one shared Workbench per inspection ID for the browser and MCP.
// Workbenches stay loaded without any time limit and are evicted only when loading another inspection would exceed the retention limit.
type WorkbenchManager struct {
	mu                   sync.Mutex
	entries              map[string]*entry
	inspectionServer     *coreinspection.InspectionTaskServer
	indexManager         *InspectionIndexManager
	maxLoadedInspections int
	now                  func() time.Time
	openJobs             *streamingutil.AsyncJobManager[*apiv1.OpenWorkbenchSyncResponse, string]
}

// NewWorkbenchManager creates a WorkbenchManager that keeps at most maxLoadedInspections inspections loaded, except while loads are still in progress.
func NewWorkbenchManager(inspectionServer *coreinspection.InspectionTaskServer, indexManager *InspectionIndexManager, maxLoadedInspections int) *WorkbenchManager {
	if indexManager == nil {
		panic("indexManager is required")
	}
	return &WorkbenchManager{
		entries:              make(map[string]*entry),
		inspectionServer:     inspectionServer,
		indexManager:         indexManager,
		maxLoadedInspections: maxLoadedInspections,
		now:                  time.Now,
		openJobs:             streamingutil.NewAsyncJobManager[*apiv1.OpenWorkbenchSyncResponse, string](15*time.Second, 1*time.Minute),
	}
}

// OpenJobManager returns the AsyncJobManager tracking asynchronous workbench open tasks.
func (m *WorkbenchManager) OpenJobManager() *streamingutil.AsyncJobManager[*apiv1.OpenWorkbenchSyncResponse, string] {
	return m.openJobs
}

// Load returns the handle of the workbench for the inspection without blocking.
// It starts loading when the inspection is neither loaded nor loading, evicting other workbenches first if the retention limit is reached.
// Loading runs detached from any caller, so it continues after the caller returns.
func (m *WorkbenchManager) Load(inspectionID string, accessor Accessor) *LoadHandle {
	m.mu.Lock()
	now := m.now()
	if e, ok := m.entries[inspectionID]; ok {
		e.touch(accessor, now)
		m.mu.Unlock()
		return e.load
	}
	evicted := m.evictForNewLoadLocked(now)
	e := &entry{load: newLoadHandle()}
	e.touch(accessor, now)
	m.entries[inspectionID] = e
	m.mu.Unlock()

	for _, wb := range evicted {
		wb.Close()
	}
	go m.runLoad(inspectionID, e)
	return e.load
}

// Open returns the workbench for the inspection, loading it if needed, and blocks until loading finishes or ctx is done.
func (m *WorkbenchManager) Open(ctx context.Context, inspectionID string, accessor Accessor, onProgress ProgressCallback) (*Workbench, error) {
	return m.Load(inspectionID, accessor).Wait(ctx, onProgress)
}

// Get returns the loaded workbench for the inspection and records the access.
// It returns ErrWorkbenchNotFound when the inspection is not loaded yet or was evicted.
func (m *WorkbenchManager) Get(inspectionID string, accessor Accessor) (*Workbench, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.entries[inspectionID]
	if !ok || !e.load.isLoaded() {
		return nil, ErrWorkbenchNotFound
	}
	e.touch(accessor, m.now())
	return e.load.wb, nil
}

// Heartbeat records that a browser is still showing the inspection and reports whether its workbench is loaded.
// It does not update the last access time, so a tab left open does not keep its workbench ahead of recently used ones.
func (m *WorkbenchManager) Heartbeat(inspectionID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.entries[inspectionID]
	if !ok || !e.load.isLoaded() {
		return false
	}
	e.browserSeenAt = m.now()
	return true
}

// MarkBrowserClosed records that a browser stopped showing the inspection, which makes its workbench an eviction candidate.
// The workbench stays loaded because other browsers or MCP may share it. Unknown inspection IDs are ignored.
func (m *WorkbenchManager) MarkBrowserClosed(inspectionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.entries[inspectionID]; ok {
		e.browserSeenAt = time.Time{}
	}
}

// touch records an access by the given accessor at now.
func (e *entry) touch(accessor Accessor, now time.Time) {
	e.lastAccess = now
	switch accessor {
	case AccessorBrowser:
		e.browserSeenAt = now
	case AccessorMCP:
		e.mcpUsedAt = now
	}
}

// evictForNewLoadLocked removes loaded workbenches until one more load fits in the retention limit, and returns them for closing.
// Workbenches still loading are never evicted, so the limit can be exceeded temporarily when many loads run at once.
func (m *WorkbenchManager) evictForNewLoadLocked(now time.Time) []*Workbench {
	var evicted []*Workbench
	for len(m.entries) >= m.maxLoadedInspections {
		victimID, inUse := m.pickEvictionVictimLocked(now)
		if victimID == "" {
			break
		}
		victim := m.entries[victimID]
		delete(m.entries, victimID)
		evicted = append(evicted, victim.load.wb)
		slog.Info("evicted a workbench to load another inspection",
			"inspectionID", victimID,
			"reason", evictionReason(inUse),
			"lastAccess", victim.lastAccess,
			"maxLoadedInspections", m.maxLoadedInspections)
	}
	return evicted
}

// pickEvictionVictimLocked returns the loaded entry with the oldest last access, preferring entries that are not in use.
// It falls back to an entry in use because refusing the load would stop users from opening another inspection.
// It returns an empty ID when no entry is loaded.
func (m *WorkbenchManager) pickEvictionVictimLocked(now time.Time) (victimID string, inUse bool) {
	var idleID, busyID string
	var idleAccess, busyAccess time.Time
	for id, e := range m.entries {
		if !e.load.isLoaded() {
			continue
		}
		if e.inUse(now) {
			if busyID == "" || e.lastAccess.Before(busyAccess) {
				busyID, busyAccess = id, e.lastAccess
			}
			continue
		}
		if idleID == "" || e.lastAccess.Before(idleAccess) {
			idleID, idleAccess = id, e.lastAccess
		}
	}
	if idleID != "" {
		return idleID, false
	}
	return busyID, busyID != ""
}

func evictionReason(inUse bool) string {
	if inUse {
		return "retention limit reached and every loaded workbench is in use; evicted the least recently accessed one"
	}
	return "retention limit reached; evicted the least recently accessed workbench not in use"
}

// runLoad loads the inspection for the entry and publishes the result to its handle.
// A failed load removes the entry so that the next request starts a fresh load.
func (m *WorkbenchManager) runLoad(inspectionID string, e *entry) {
	wb, err := m.loadWorkbench(inspectionID, e.load.report)
	if err != nil {
		m.mu.Lock()
		if m.entries[inspectionID] == e {
			delete(m.entries, inspectionID)
		}
		m.mu.Unlock()
	}
	e.load.finish(wb, err)
}

// loadWorkbench reads the inspection result into a new Workbench and starts building its search index.
func (m *WorkbenchManager) loadWorkbench(inspectionID string, onProgress ProgressCallback) (*Workbench, error) {
	reader, totalSize, err := m.loadInspectionData(inspectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load inspection data: %w", err)
	}
	defer reader.Close()

	if err := onProgress(apiv1.OpenWorkbenchResponse_STAGE_READING_FILE, 10, "Opening inspection dataset..."); err != nil {
		return nil, err
	}

	// The load is shared by every caller, so no single caller's context may cancel it.
	wb, err := NewFromReader(context.Background(), inspectionID, inspectionID, reader, totalSize, onProgress)
	if err != nil {
		return nil, err
	}
	wb.SetIndexManager(m.indexManager)
	wb.StartAsyncIndexing(context.Background())
	return wb, nil
}

// loadInspectionData loads the KHI result reader and byte size for the given inspection ID.
func (m *WorkbenchManager) loadInspectionData(inspectionID string) (io.ReadCloser, int64, error) {
	currentTask := m.inspectionServer.GetInspection(inspectionID)
	if currentTask == nil {
		return nil, 0, fmt.Errorf("%w: %s", ErrInspectionNotFound, inspectionID)
	}
	result, err := currentTask.Result()
	if err != nil {
		return nil, 0, err
	}
	size, err := result.ResultStore.GetInspectionResultSizeInBytes()
	if err != nil {
		return nil, 0, err
	}
	reader, err := result.ResultStore.GetRangeReader(0, int64(size))
	if err != nil {
		return nil, 0, err
	}
	return reader, int64(size), nil
}
