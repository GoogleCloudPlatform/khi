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
	"sort"
	"sync"
	"testing"
	"time"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logger"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	apiv1 "github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
)

func createTestInspectionServer(t *testing.T) (*coreinspection.InspectionTaskServer, string) {
	logger.InitGlobalKHILogger()
	ioConfig, err := inspectioncore.NewIOConfigForTest()
	if err != nil {
		t.Fatalf("failed to create test IOConfig: %v", err)
	}
	tempDir := t.TempDir()
	ioConfig.DataDestination = tempDir
	ioConfig.TemporaryFolder = tempDir
	server, err := coreinspection.NewServer(ioConfig)
	if err != nil {
		t.Fatalf("failed to create inspection server: %v", err)
	}

	inspectionType := coreinspection.InspectionType{
		Id:   "test-type",
		Name: "Test Type",
	}
	if err := server.AddInspectionType(inspectionType); err != nil {
		t.Fatalf("failed to add inspection type: %v", err)
	}

	dummyTaskID := taskid.NewDefaultImplementationID[any]("dummy-task")
	dummyTask := coretask.NewTask(
		dummyTaskID,
		nil,
		func(ctx context.Context) (any, error) {
			return "success", nil
		},
		coretask.WithLabelValue(inspectioncore.LabelKeyInspectionDefaultFeatureFlag, true),
		coretask.WithLabelValue(inspectioncore.LabelKeyInspectionFeatureFlag, true),
	)
	if err := server.AddTask(dummyTask); err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	return server, runTestInspection(t, server)
}

// runTestInspection creates and runs one more inspection on the server and returns its ID.
func runTestInspection(t *testing.T, server *coreinspection.InspectionTaskServer) string {
	t.Helper()
	inspectionID, err := server.CreateInspection("test-type")
	if err != nil {
		t.Fatalf("failed to create inspection: %v", err)
	}
	runner := server.GetInspection(inspectionID)
	if err := runner.Run(context.Background(), &inspectioncore.InspectionRequest{Values: map[string]any{}}); err != nil {
		t.Fatalf("failed to run inspection: %v", err)
	}
	<-runner.Wait()
	return inspectionID
}

// newTestManager creates a manager whose background indexing is awaited before the test's temporary directories are removed.
func newTestManager(t *testing.T, inspectionServer *coreinspection.InspectionTaskServer, maxWorkbenches int) *WorkbenchManager {
	t.Helper()
	indexMgr := NewInspectionIndexManager(inspectionServer, t.TempDir())
	mgr := NewWorkbenchManager(inspectionServer, indexMgr, maxWorkbenches)
	t.Cleanup(func() {
		mgr.mu.Lock()
		var loaded []*Workbench
		for _, e := range mgr.entries {
			if e.load.isLoaded() {
				loaded = append(loaded, e.load.wb)
			}
		}
		mgr.mu.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for _, wb := range loaded {
			if state, _, _, _ := wb.IndexStatus(); state == IndexStateBuilding {
				_ = wb.AwaitIndex(ctx)
			}
			wb.Close()
		}
		indexMgr.Wait()
	})
	return mgr
}

// fakeClock is a manually advanced clock injected into the manager so tests can simulate long idle periods without sleeping.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock(mgr *WorkbenchManager) *fakeClock {
	c := &fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	mgr.now = c.Now
	return c
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func noopProgress(stage apiv1.OpenWorkbenchResponse_Stage, pct float64, msg string) error {
	return nil
}

func TestWorkbenchManager_Open(t *testing.T) {
	inspectionServer, validInspectionID := createTestInspectionServer(t)

	testCases := []struct {
		name         string
		inspectionID string
		wantErrIs    error
	}{
		{
			name:         "opens a workbench identified by the inspection ID",
			inspectionID: validInspectionID,
		},
		{
			name:         "fails when the inspection does not exist",
			inspectionID: "invalid-inspection",
			wantErrIs:    ErrInspectionNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mgr := newTestManager(t, inspectionServer, 3)

			var stages []apiv1.OpenWorkbenchResponse_Stage
			wb, err := mgr.Open(context.Background(), tc.inspectionID, AccessorBrowser, func(stage apiv1.OpenWorkbenchResponse_Stage, pct float64, msg string) error {
				stages = append(stages, stage)
				return nil
			})
			if tc.wantErrIs != nil {
				if !errors.Is(err, tc.wantErrIs) {
					t.Fatalf("Open() error = %v, want %v", err, tc.wantErrIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("Open() unexpected error: %v", err)
			}
			if wb.ID() != tc.inspectionID {
				t.Errorf("wb.ID() = %q, want %q", wb.ID(), tc.inspectionID)
			}
			if len(stages) == 0 {
				t.Fatalf("Open() reported no progress")
			}
			if got := stages[len(stages)-1]; got != apiv1.OpenWorkbenchResponse_STAGE_READY {
				t.Errorf("final progress stage = %v, want STAGE_READY", got)
			}
		})
	}
}

func TestWorkbenchManager_SharesWorkbenchAcrossBrowsersAndMCP(t *testing.T) {
	inspectionServer, inspectionID := createTestInspectionServer(t)
	mgr := newTestManager(t, inspectionServer, 3)

	accessors := []Accessor{AccessorBrowser, AccessorBrowser, AccessorMCP}
	handles := make([]*LoadHandle, len(accessors))
	workbenches := make([]*Workbench, len(accessors))
	errs := make([]error, len(accessors))
	var wg sync.WaitGroup
	for i, accessor := range accessors {
		wg.Add(1)
		go func() {
			defer wg.Done()
			handles[i] = mgr.Load(inspectionID, accessor)
			workbenches[i], errs[i] = handles[i].Wait(context.Background(), noopProgress)
		}()
	}
	wg.Wait()

	for i := range accessors {
		if errs[i] != nil {
			t.Fatalf("caller %d returned error: %v", i, errs[i])
		}
		if handles[i] != handles[0] {
			t.Errorf("caller %d received load handle %p, want the shared handle %p", i, handles[i], handles[0])
		}
		if workbenches[i] != workbenches[0] {
			t.Errorf("caller %d received workbench %p, want the shared workbench %p", i, workbenches[i], workbenches[0])
		}
	}
	got, err := mgr.Get(inspectionID, AccessorMCP)
	if err != nil {
		t.Fatalf("Get() unexpected error: %v", err)
	}
	if got != workbenches[0] {
		t.Errorf("Get() = %p, want the shared workbench %p", got, workbenches[0])
	}
}

func TestWorkbenchManager_KeepsWorkbenchAfterLongInactivityBelowLimit(t *testing.T) {
	inspectionServer, inspectionID := createTestInspectionServer(t)
	secondInspectionID := runTestInspection(t, inspectionServer)
	mgr := newTestManager(t, inspectionServer, 2)
	clock := newFakeClock(mgr)

	wb, err := mgr.Open(context.Background(), inspectionID, AccessorBrowser, noopProgress)
	if err != nil {
		t.Fatalf("Open() unexpected error: %v", err)
	}
	clock.Advance(30 * 24 * time.Hour)
	if _, err := mgr.Open(context.Background(), secondInspectionID, AccessorMCP, noopProgress); err != nil {
		t.Fatalf("Open(second) unexpected error: %v", err)
	}

	got, err := mgr.Get(inspectionID, AccessorBrowser)
	if err != nil {
		t.Fatalf("Get() after long inactivity unexpected error: %v", err)
	}
	if got != wb {
		t.Errorf("Get() = %p, want %p", got, wb)
	}
	if wb.IsClosed() {
		t.Errorf("workbench was closed after long inactivity below the retention limit")
	}
}

func TestWorkbenchManager_EvictsOnNewLoadOverLimit(t *testing.T) {
	const (
		recent  = time.Second
		longAgo = 24 * time.Hour
	)
	type seed struct {
		id      string
		loading bool
		// Each duration is how long before the new load the corresponding access happened.
		lastAccessAgo  time.Duration
		browserSeenAgo time.Duration
		mcpUsedAgo     time.Duration
	}
	testCases := []struct {
		name           string
		maxWorkbenches int
		seeds          []seed
		wantRemaining  []string
	}{
		{
			name:           "evicts the least recently accessed workbench that is not in use",
			maxWorkbenches: 2,
			seeds: []seed{
				{id: "older", lastAccessAgo: 10 * time.Minute, browserSeenAgo: longAgo, mcpUsedAgo: longAgo},
				{id: "newer", lastAccessAgo: 5 * time.Minute, browserSeenAgo: longAgo, mcpUsedAgo: longAgo},
			},
			wantRemaining: []string{"newer"},
		},
		{
			name:           "keeps a workbench shown in a browser",
			maxWorkbenches: 2,
			seeds: []seed{
				{id: "shown", lastAccessAgo: 10 * time.Minute, browserSeenAgo: 10 * time.Second, mcpUsedAgo: longAgo},
				{id: "idle", lastAccessAgo: 5 * time.Minute, browserSeenAgo: longAgo, mcpUsedAgo: longAgo},
			},
			wantRemaining: []string{"shown"},
		},
		{
			name:           "evicts a workbench whose browser stopped sending heartbeats",
			maxWorkbenches: 2,
			seeds: []seed{
				{id: "hidden-tab", lastAccessAgo: 10 * time.Minute, browserSeenAgo: time.Minute, mcpUsedAgo: longAgo},
				{id: "idle", lastAccessAgo: 5 * time.Minute, browserSeenAgo: longAgo, mcpUsedAgo: longAgo},
			},
			wantRemaining: []string{"idle"},
		},
		{
			name:           "keeps a workbench recently used by MCP",
			maxWorkbenches: 2,
			seeds: []seed{
				{id: "mcp", lastAccessAgo: 10 * time.Minute, browserSeenAgo: longAgo, mcpUsedAgo: time.Minute},
				{id: "idle", lastAccessAgo: 5 * time.Minute, browserSeenAgo: longAgo, mcpUsedAgo: longAgo},
			},
			wantRemaining: []string{"mcp"},
		},
		{
			name:           "falls back to the least recently accessed workbench when all are in use",
			maxWorkbenches: 2,
			seeds: []seed{
				{id: "older", lastAccessAgo: 10 * time.Minute, browserSeenAgo: recent, mcpUsedAgo: longAgo},
				{id: "newer", lastAccessAgo: 5 * time.Minute, browserSeenAgo: longAgo, mcpUsedAgo: recent},
			},
			wantRemaining: []string{"newer"},
		},
		{
			name:           "never evicts a workbench that is still loading",
			maxWorkbenches: 2,
			seeds: []seed{
				{id: "loading", loading: true, lastAccessAgo: 10 * time.Minute, browserSeenAgo: longAgo, mcpUsedAgo: longAgo},
				{id: "loaded", lastAccessAgo: 5 * time.Minute, browserSeenAgo: recent, mcpUsedAgo: recent},
			},
			wantRemaining: []string{"loading"},
		},
		{
			name:           "exceeds the limit when only loading workbenches remain",
			maxWorkbenches: 1,
			seeds: []seed{
				{id: "loading", loading: true, lastAccessAgo: 10 * time.Minute, browserSeenAgo: longAgo, mcpUsedAgo: longAgo},
			},
			wantRemaining: []string{"loading"},
		},
		{
			name:           "does not evict below the limit",
			maxWorkbenches: 3,
			seeds: []seed{
				{id: "a", lastAccessAgo: 10 * time.Minute, browserSeenAgo: longAgo, mcpUsedAgo: longAgo},
				{id: "b", lastAccessAgo: 5 * time.Minute, browserSeenAgo: longAgo, mcpUsedAgo: longAgo},
			},
			wantRemaining: []string{"a", "b"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inspectionServer, _ := createTestInspectionServer(t)
			mgr := newTestManager(t, inspectionServer, tc.maxWorkbenches)
			clock := newFakeClock(mgr)
			now := clock.Now()

			workbenches := map[string]*Workbench{}
			for _, s := range tc.seeds {
				h := newLoadHandle()
				if !s.loading {
					wb := NewWorkbench(s.id, s.id)
					workbenches[s.id] = wb
					h.finish(wb, nil)
				}
				mgr.entries[s.id] = &entry{
					load:          h,
					lastAccess:    now.Add(-s.lastAccessAgo),
					browserSeenAt: now.Add(-s.browserSeenAgo),
					mcpUsedAt:     now.Add(-s.mcpUsedAgo),
				}
			}

			// The new inspection does not exist, so its own load fails and is discarded; only the eviction matters here.
			mgr.Load("new-inspection", AccessorMCP)

			mgr.mu.Lock()
			var remaining []string
			for id := range mgr.entries {
				if id != "new-inspection" {
					remaining = append(remaining, id)
				}
			}
			mgr.mu.Unlock()
			sort.Strings(remaining)
			if diff := cmp.Diff(tc.wantRemaining, remaining); diff != "" {
				t.Errorf("remaining workbenches mismatch (-want +got):\n%s", diff)
			}
			for id, wb := range workbenches {
				wantClosed := true
				for _, kept := range tc.wantRemaining {
					if kept == id {
						wantClosed = false
					}
				}
				if wb.IsClosed() != wantClosed {
					t.Errorf("workbench %q IsClosed() = %v, want %v", id, wb.IsClosed(), wantClosed)
				}
			}
		})
	}
}

func TestWorkbenchManager_LoadContinuesAfterCallerCancel(t *testing.T) {
	inspectionServer, inspectionID := createTestInspectionServer(t)
	mgr := newTestManager(t, inspectionServer, 3)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := mgr.Open(ctx, inspectionID, AccessorMCP, noopProgress); err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("Open() with a cancelled context error = %v, want nil or context.Canceled", err)
	}

	h := mgr.Load(inspectionID, AccessorBrowser)
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer waitCancel()
	wb, err := h.Wait(waitCtx, noopProgress)
	if err != nil {
		t.Fatalf("Wait() error = %v, want nil", err)
	}
	if wb.IsClosed() {
		t.Errorf("loaded workbench is closed")
	}
}

func TestWorkbenchManager_RetriesFailedLoad(t *testing.T) {
	inspectionServer, _ := createTestInspectionServer(t)
	mgr := newTestManager(t, inspectionServer, 3)

	first := mgr.Load("missing-inspection", AccessorMCP)
	if _, err := first.Wait(context.Background(), noopProgress); !errors.Is(err, ErrInspectionNotFound) {
		t.Fatalf("first Wait() error = %v, want ErrInspectionNotFound", err)
	}

	second := mgr.Load("missing-inspection", AccessorMCP)
	if second == first {
		t.Errorf("Load() after a failed load returned the failed handle, want a new load")
	}
	_, _ = second.Wait(context.Background(), noopProgress)
}

func TestWorkbenchManager_Heartbeat(t *testing.T) {
	testCases := []struct {
		name  string
		setup func(t *testing.T, mgr *WorkbenchManager, inspectionID string, otherInspectionID string) string
		want  bool
	}{
		{
			name: "loaded workbench is active",
			setup: func(t *testing.T, mgr *WorkbenchManager, inspectionID string, otherInspectionID string) string {
				if _, err := mgr.Open(context.Background(), inspectionID, AccessorBrowser, noopProgress); err != nil {
					t.Fatalf("Open() unexpected error: %v", err)
				}
				return inspectionID
			},
			want: true,
		},
		{
			name: "unknown workbench is inactive",
			setup: func(t *testing.T, mgr *WorkbenchManager, inspectionID string, otherInspectionID string) string {
				return "unknown-inspection"
			},
			want: false,
		},
		{
			name: "evicted workbench is inactive",
			setup: func(t *testing.T, mgr *WorkbenchManager, inspectionID string, otherInspectionID string) string {
				if _, err := mgr.Open(context.Background(), inspectionID, AccessorBrowser, noopProgress); err != nil {
					t.Fatalf("Open() unexpected error: %v", err)
				}
				if _, err := mgr.Open(context.Background(), otherInspectionID, AccessorBrowser, noopProgress); err != nil {
					t.Fatalf("Open(other) unexpected error: %v", err)
				}
				return inspectionID
			},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inspectionServer, inspectionID := createTestInspectionServer(t)
			otherInspectionID := runTestInspection(t, inspectionServer)
			mgr := newTestManager(t, inspectionServer, 1)

			target := tc.setup(t, mgr, inspectionID, otherInspectionID)
			if got := mgr.Heartbeat(target); got != tc.want {
				t.Errorf("Heartbeat(%q) = %v, want %v", target, got, tc.want)
			}
		})
	}
}

func TestWorkbenchManager_Get(t *testing.T) {
	inspectionServer, inspectionID := createTestInspectionServer(t)
	mgr := newTestManager(t, inspectionServer, 3)
	wb, err := mgr.Open(context.Background(), inspectionID, AccessorBrowser, noopProgress)
	if err != nil {
		t.Fatalf("Open() unexpected error: %v", err)
	}

	testCases := []struct {
		name         string
		inspectionID string
		want         *Workbench
		wantErrIs    error
	}{
		{
			name:         "returns the loaded workbench",
			inspectionID: inspectionID,
			want:         wb,
		},
		{
			name:         "returns ErrWorkbenchNotFound for an inspection that is not loaded",
			inspectionID: "unknown-inspection",
			wantErrIs:    ErrWorkbenchNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := mgr.Get(tc.inspectionID, AccessorBrowser)
			if !errors.Is(err, tc.wantErrIs) {
				t.Fatalf("Get(%q) error = %v, want %v", tc.inspectionID, err, tc.wantErrIs)
			}
			if got != tc.want {
				t.Errorf("Get(%q) = %p, want %p", tc.inspectionID, got, tc.want)
			}
		})
	}
}

func TestWorkbenchManager_MarkBrowserClosedMakesWorkbenchEvictable(t *testing.T) {
	inspectionServer, closedInspectionID := createTestInspectionServer(t)
	shownInspectionID := runTestInspection(t, inspectionServer)
	newInspectionID := runTestInspection(t, inspectionServer)
	mgr := newTestManager(t, inspectionServer, 2)
	clock := newFakeClock(mgr)

	shown, err := mgr.Open(context.Background(), shownInspectionID, AccessorBrowser, noopProgress)
	if err != nil {
		t.Fatalf("Open(shown) unexpected error: %v", err)
	}
	clock.Advance(time.Second)
	closed, err := mgr.Open(context.Background(), closedInspectionID, AccessorBrowser, noopProgress)
	if err != nil {
		t.Fatalf("Open(closed) unexpected error: %v", err)
	}

	mgr.MarkBrowserClosed(closedInspectionID)
	if closed.IsClosed() {
		t.Fatalf("MarkBrowserClosed() released the workbench, want it to stay loaded")
	}

	// The closed workbench was accessed more recently, but it is the only one not in use.
	if _, err := mgr.Open(context.Background(), newInspectionID, AccessorBrowser, noopProgress); err != nil {
		t.Fatalf("Open(new) unexpected error: %v", err)
	}
	if !closed.IsClosed() {
		t.Errorf("workbench closed by the browser was not evicted")
	}
	if shown.IsClosed() {
		t.Errorf("workbench shown in a browser was evicted")
	}
}

func TestWorkbenchManager_WithInspectionIndexManager(t *testing.T) {
	testCases := []struct {
		name              string
		withPrebuiltIndex bool
	}{
		{
			name:              "loads prebuilt index when available in indexManager",
			withPrebuiltIndex: true,
		},
		{
			name:              "indexes asynchronously on demand when not prebuilt",
			withPrebuiltIndex: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inspectionServer, validInspectionID := createTestInspectionServer(t)
			indexMgr := NewInspectionIndexManager(inspectionServer, t.TempDir())
			t.Cleanup(indexMgr.Wait)

			if tc.withPrebuiltIndex {
				indexMgr.StartAsyncIndexing(context.Background(), validInspectionID)
				deadline := time.Now().Add(5 * time.Second)
				for time.Now().Before(deadline) {
					state, _, _, _ := indexMgr.IndexStatus(validInspectionID)
					if state == IndexStateReady {
						break
					}
					time.Sleep(10 * time.Millisecond)
				}
			}

			wbMgr := NewWorkbenchManager(inspectionServer, indexMgr, 3)
			wb, err := wbMgr.Open(context.Background(), validInspectionID, AccessorBrowser, noopProgress)
			if err != nil {
				t.Fatalf("Open() failed: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := wb.AwaitIndex(ctx); err != nil {
				t.Fatalf("AwaitIndex() error = %v", err)
			}
			if wb.searchIndex == nil || wb.searchIndex.TrigramIndex == nil {
				t.Errorf("expected searchIndex.TrigramIndex to be populated")
			}
		})
	}
}

func TestNewWorkbenchManager_PanicsOnNilIndexManager(t *testing.T) {
	inspectionServer, _ := createTestInspectionServer(t)
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("expected panic when indexManager is nil, got none")
		}
	}()
	NewWorkbenchManager(inspectionServer, nil, 3)
}
