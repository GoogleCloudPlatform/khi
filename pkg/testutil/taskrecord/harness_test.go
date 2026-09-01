// Copyright 2025 Google LLC
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

package taskrecord

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logger"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
)

var pathHarnessTestTextPayload = structured.CompileFieldPath("textPayload")

var (
	upstreamTaskID   = taskid.NewDefaultImplementationID[[]*log.Log]("test/upstream")
	downstreamTaskID = taskid.NewDefaultImplementationID[[]string]("test/downstream")
)

type harnessTestContext struct {
	server          *coreinspection.InspectionTaskServer
	upstreamExecs   *atomic.Int32
	downstreamExecs *atomic.Int32
}

func setupTestServer(t *testing.T) *harnessTestContext {
	t.Helper()
	logger.InitGlobalKHILogger()
	ioConfig, err := inspectioncore_contract.NewIOConfigForTest()
	if err != nil {
		t.Fatalf("failed to create ioConfig: %v", err)
	}

	server, err := coreinspection.NewServer(ioConfig)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	if err := server.AddInspectionType(coreinspection.InspectionType{
		Id:   "test-type",
		Name: "Test Type",
	}); err != nil {
		t.Fatalf("failed to add inspection type: %v", err)
	}

	upstreamExecs := &atomic.Int32{}
	downstreamExecs := &atomic.Int32{}

	upstreamTask := coretask.NewTask[[]*log.Log](
		upstreamTaskID,
		[]taskid.UntypedTaskReference{},
		func(ctx context.Context) ([]*log.Log, error) {
			upstreamExecs.Add(1)
			l, err := log.NewLogFromYAMLString("textPayload: hello from upstream\nseverity: INFO")
			if err != nil {
				return nil, err
			}
			return []*log.Log{l}, nil
		},
	)

	downstreamTask := coretask.NewTask[[]string](
		downstreamTaskID,
		[]taskid.UntypedTaskReference{upstreamTaskID.Ref()},
		func(ctx context.Context) ([]string, error) {
			downstreamExecs.Add(1)
			logs := coretask.GetTaskResult(ctx, upstreamTaskID.Ref())
			var msgs []string
			for _, l := range logs {
				msgs = append(msgs, l.ReadStringOrDefault(pathHarnessTestTextPayload, ""))
			}
			return msgs, nil
		},
		coretask.NewSubsequentTaskRefsTaskLabel(inspectioncore_contract.SerializerTaskID.Ref()),
		inspectioncore_contract.FeatureTaskLabel("Downstream Task", "Downstream Task", 0, true),
	)

	if err := server.AddTask(upstreamTask); err != nil {
		t.Fatalf("failed to add upstream task: %v", err)
	}
	if err := server.AddTask(downstreamTask); err != nil {
		t.Fatalf("failed to add downstream task: %v", err)
	}

	return &harnessTestContext{
		server:          server,
		upstreamExecs:   upstreamExecs,
		downstreamExecs: downstreamExecs,
	}
}

func TestJobTestHarness_RecordAndReplay(t *testing.T) {
	tc := setupTestServer(t)

	cfg := &JobTestConfig{
		InspectionType: "test-type",
		RecordedTasks: []taskid.UntypedTaskReference{
			upstreamTaskID.Ref(),
		},
		TargetTask: downstreamTaskID.Ref(),
	}

	harness := NewJobTestHarness(t, tc.server, cfg)
	defer os.RemoveAll(harness.fixtureDir)

	t.Run("record mode captures fixture", func(t *testing.T) {
		res, err := harness.Record(t.Context())
		if err != nil {
			t.Fatalf("harness.Record() failed: %v", err)
		}
		if res == nil {
			t.Fatal("expected non-nil JobTestResult from Record()")
		}

		fixtureFile := filepath.Join(harness.fixtureDir, sanitizeTaskReferenceForFileName(upstreamTaskID.Ref().ReferenceIDString())+".json")
		if _, err := os.Stat(fixtureFile); os.IsNotExist(err) {
			t.Fatalf("expected fixture file %s to exist", fixtureFile)
		}

		if got := tc.upstreamExecs.Load(); got != 1 {
			t.Errorf("upstream executions mismatch: got %d, want 1", got)
		}
	})

	t.Run("replay mode injects stub and executes target task", func(t *testing.T) {
		// Reset counters
		tc.upstreamExecs.Store(0)
		tc.downstreamExecs.Store(0)

		res, err := harness.Replay(t.Context())
		if err != nil {
			t.Fatalf("harness.Replay() failed: %v", err)
		}
		if res == nil {
			t.Fatal("expected non-nil JobTestResult from Replay()")
		}

		// Original upstream task should NOT have executed (stub task was used)
		if got := tc.upstreamExecs.Load(); got != 0 {
			t.Errorf("expected upstream task to be skipped in Replay, got %d executions", got)
		}

		// Target task should have executed once
		if got := tc.downstreamExecs.Load(); got != 1 {
			t.Errorf("expected downstream task to execute 1 time, got %d", got)
		}

		// Verify TargetTask result
		gotResult, ok := GetTaskResult(res, downstreamTaskID.Ref())
		if !ok {
			t.Fatal("expected downstream task result in JobTestResult")
		}

		wantResult := []string{"hello from upstream"}
		if diff := cmp.Diff(wantResult, gotResult); diff != "" {
			t.Errorf("downstream task result mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestJobTestHarness_Profiling(t *testing.T) {
	tc := setupTestServer(t)

	cfg := &JobTestConfig{
		InspectionType: "test-type",
		RecordedTasks: []taskid.UntypedTaskReference{
			upstreamTaskID.Ref(),
		},
		TargetTask: downstreamTaskID.Ref(),
	}

	defer os.RemoveAll("pprof")

	t.Run("profiling generates cpu and mem profiles", func(subT *testing.T) {
		harness := NewJobTestHarness(subT, tc.server, cfg)
		defer os.RemoveAll(harness.fixtureDir)

		// Step 1: Record fixture
		if _, err := harness.Record(subT.Context()); err != nil {
			subT.Fatalf("harness.Record() failed: %v", err)
		}

		// Step 2: Enable CPU & Mem profile paths explicitly for testing profiling flow
		harness.cpuProfilePath = filepath.Join("pprof", sanitizeTestName(subT.Name()), "cpu.pprof")
		harness.memProfilePath = filepath.Join("pprof", sanitizeTestName(subT.Name()), "mem.pprof")

		if _, err := harness.Replay(subT.Context()); err != nil {
			subT.Fatalf("harness.Replay() failed: %v", err)
		}
	})

	cpuPath := filepath.Join("pprof", sanitizeTestName(t.Name()+"/profiling_generates_cpu_and_mem_profiles"), "cpu.pprof")
	memPath := filepath.Join("pprof", sanitizeTestName(t.Name()+"/profiling_generates_cpu_and_mem_profiles"), "mem.pprof")

	if _, err := os.Stat(cpuPath); os.IsNotExist(err) {
		t.Errorf("expected CPU profile %s to exist", cpuPath)
	}

	if _, err := os.Stat(memPath); os.IsNotExist(err) {
		t.Errorf("expected Mem profile %s to exist", memPath)
	}
}
