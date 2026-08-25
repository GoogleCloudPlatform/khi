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
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6model "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/google/go-cmp/cmp"
)

func createTestKhiFile(t *testing.T, dir string, inspectionID string) {
	t.Helper()
	b := khifilev6model.NewBuilder()
	node := structured.NewStandardMap(
		[]string{"name", "message"},
		[]structured.Node{
			structured.NewStandardScalarNode("nginx-pod"),
			structured.NewStandardScalarNode("hello nginx"),
		},
	)
	severityID := uint32(1)
	logTypeID := uint32(2)
	if err := b.LogAccumulator.AddLog(&khifilev6model.StagingLog{
		Log:       log.NewLog(structured.NewNodeReader(node)),
		Summary:   "nginx log",
		Timestamp: time.Date(2026, 4, 29, 8, 0, 0, 0, time.UTC),
		Severity:  &khifilev6.Severity{Id: &severityID},
		LogType:   &khifilev6.LogType{Id: &logTypeID},
	}); err != nil {
		t.Fatalf("failed to add log: %v", err)
	}

	filePath := filepath.Join(dir, inspectionID+".khi")
	f, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer f.Close()

	if err := b.Build(f, nil); err != nil {
		t.Fatalf("failed to build KHI file: %v", err)
	}
}

func TestInspectionIndexManager(t *testing.T) {
	tempDir := t.TempDir()
	const inspectionID = "test-insp-001"
	createTestKhiFile(t, tempDir, inspectionID)

	mgr := NewInspectionIndexManager(nil, tempDir)

	testCases := []struct {
		name       string
		testAction func(t *testing.T)
	}{
		{
			name: "async indexing builds index and saves to disk",
			testAction: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				ch, unsubscribe := mgr.SubscribeIndexProgress(ctx, inspectionID)
				defer unsubscribe()

				mgr.StartAsyncIndexing(ctx, inspectionID)

				var lastEvent IndexProgressEvent
				for ev := range ch {
					lastEvent = ev
					if ev.State == IndexStateReady || ev.State == IndexStateFailed {
						break
					}
				}

				if lastEvent.State != IndexStateReady {
					t.Fatalf("expected IndexStateReady, got %v with error: %v", lastEvent.State, lastEvent.Err)
				}

				// Verify disk file was created
				diskPath := filepath.Join(tempDir, inspectionID+".trigram")
				if _, err := os.Stat(diskPath); err != nil {
					t.Fatalf("expected trigram disk file to exist at %s: %v", diskPath, err)
				}

				// Verify query candidate search
				idx, ok := mgr.GetTrigramIndex(inspectionID)
				if !ok || idx == nil {
					t.Fatalf("expected GetTrigramIndex to return ready index")
				}

				bm := idx.FindCandidateLogs("nginx")
				if bm == nil || bm.IsEmpty() {
					t.Errorf("expected candidate logs for 'nginx' to be non-empty, got %v", bm)
				}
				bmCoredns := idx.FindCandidateLogs("coredns")
				if bmCoredns != nil && !bmCoredns.IsEmpty() {
					t.Errorf("expected candidate logs for 'coredns' to be empty, got %v", bmCoredns)
				}
			},
		},
		{
			name: "second manager loads existing index directly from disk cache",
			testAction: func(t *testing.T) {
				newMgr := NewInspectionIndexManager(nil, tempDir)

				// Should fast-load from disk without starting async indexing
				idx, ok := newMgr.GetTrigramIndex(inspectionID)
				if !ok || idx == nil {
					t.Fatalf("expected GetTrigramIndex to load index from disk")
				}

				bm := idx.FindCandidateLogs("nginx")
				if bm == nil || bm.IsEmpty() {
					t.Errorf("expected candidate logs for 'nginx' to be non-empty, got %v", bm)
				}
			},
		},
		{
			name: "nonexistent inspection reports error",
			testAction: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()

				const nonExistentID = "nonexistent-insp"
				ch, unsubscribe := mgr.SubscribeIndexProgress(ctx, nonExistentID)
				defer unsubscribe()

				mgr.StartAsyncIndexing(ctx, nonExistentID)

				var lastEvent IndexProgressEvent
				for ev := range ch {
					lastEvent = ev
					if ev.State == IndexStateFailed {
						break
					}
				}

				if lastEvent.State != IndexStateFailed {
					t.Errorf("expected IndexStateFailed for nonexistent inspection, got %v", lastEvent.State)
				}
			},
		},
		{
			name: "delete index removes memory cache and disk file",
			testAction: func(t *testing.T) {
				mgr.DeleteIndex(inspectionID)

				diskPath := filepath.Join(tempDir, inspectionID+".trigram")
				if _, err := os.Stat(diskPath); !os.IsNotExist(err) {
					t.Errorf("expected trigram disk file to be removed, stat err: %v", err)
				}

				if _, ok := mgr.GetTrigramIndex(inspectionID); ok {
					t.Errorf("expected index to be evicted from memory and disk")
				}
			},
		},
		{
			name: "concurrent async indexing requests deduplicate via singleflight",
			testAction: func(t *testing.T) {
				const concurrentInspID = "test-concurrent-insp"
				createTestKhiFile(t, tempDir, concurrentInspID)

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				var wg sync.WaitGroup
				subscribersCount := 5
				events := make([]IndexProgressEvent, subscribersCount)

				for i := 0; i < subscribersCount; i++ {
					wg.Add(1)
					go func(idx int) {
						defer wg.Done()
						ch, unsub := mgr.SubscribeIndexProgress(ctx, concurrentInspID)
						defer unsub()

						mgr.StartAsyncIndexing(ctx, concurrentInspID)

						for ev := range ch {
							events[idx] = ev
							if ev.State == IndexStateReady || ev.State == IndexStateFailed {
								break
							}
						}
					}(i)
				}

				wg.Wait()

				for i, ev := range events {
					if ev.State != IndexStateReady {
						t.Errorf("subscriber %d expected IndexStateReady, got %v", i, ev.State)
					}
				}
			},
		},
		{
			name: "stale trigram index is discarded if .khi file is modified",
			testAction: func(t *testing.T) {
				const staleInspID = "test-stale-insp"
				createTestKhiFile(t, tempDir, staleInspID)

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				ch, unsubscribe := mgr.SubscribeIndexProgress(ctx, staleInspID)
				defer unsubscribe()

				mgr.StartAsyncIndexing(ctx, staleInspID)
				for ev := range ch {
					if ev.State == IndexStateReady || ev.State == IndexStateFailed {
						break
					}
				}

				diskPath := filepath.Join(tempDir, staleInspID+".trigram")
				if _, err := os.Stat(diskPath); err != nil {
					t.Fatalf("expected trigram disk file to exist: %v", err)
				}

				// Touch the .khi file so its ModTime is in the future compared to .trigram
				khiPath := filepath.Join(tempDir, staleInspID+".khi")
				newTime := time.Now().Add(10 * time.Second)
				if err := os.Chtimes(khiPath, newTime, newTime); err != nil {
					t.Fatalf("failed to update khi file mtime: %v", err)
				}

				// Evict from memory cache
				mgr.mu.Lock()
				delete(mgr.memoryCache, staleInspID)
				mgr.mu.Unlock()

				// Next GetTrigramIndex should detect stale file, delete it, and return false
				idx, ok := mgr.GetTrigramIndex(staleInspID)
				if ok || idx != nil {
					t.Errorf("expected GetTrigramIndex to reject stale index, but got ok=true")
				}
				if _, err := os.Stat(diskPath); !os.IsNotExist(err) {
					t.Errorf("expected stale trigram file to be removed from disk, stat err: %v", err)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.testAction)
	}
}

func TestInspectionIndexManager_SubscribeInitialState(t *testing.T) {
	tempDir := t.TempDir()
	mgr := NewInspectionIndexManager(nil, tempDir)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ch, unsubscribe := mgr.SubscribeIndexProgress(ctx, "unstarted-id")
	defer unsubscribe()

	select {
	case ev := <-ch:
		if diff := cmp.Diff(IndexStateNotStarted, ev.State); diff != "" {
			t.Errorf("initial state mismatch (-want +got):\n%s", diff)
		}
	case <-ctx.Done():
		t.Fatalf("timed out waiting for initial event")
	}
}
