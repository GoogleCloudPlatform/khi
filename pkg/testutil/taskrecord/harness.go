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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/GoogleCloudPlatform/khi/internal/testflags"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// JobTestConfig specifies parameters and target tasks for running an inspection Job test.
type JobTestConfig struct {
	// InspectionType is the type of inspection to create (e.g. "gke").
	InspectionType string

	// InspectionFeatures is the list of feature IDs to enable (e.g. "khi.google.com/feature/gke-audit-log" or "ALL").
	InspectionFeatures []string

	// InspectionValues is the map of input parameters passed to the inspection runner.
	InspectionValues map[string]any

	// RecordedTasks specifies the upstream tasks to record during Record mode and replace with stubs in Replay mode.
	RecordedTasks []taskid.UntypedTaskReference

	// TargetTask optionally specifies a single downstream task under test.
	// When specified, it executes strictly non-concurrently and early terminates once complete.
	TargetTask taskid.UntypedTaskReference
}

// JobTestResult contains the output and runner state from a completed Job test run.
type JobTestResult struct {
	// InspectionResult is the final inspection result store and metadata, if the inspection ran to completion.
	InspectionResult *coreinspection.InspectionRunResult

	// TaskRunner is the underlying task runner used during the run.
	TaskRunner coretask.TaskRunner

	taskResults map[string]any
}

// GetTaskResult retrieves the output of a completed task from the test result.
func GetTaskResult[T any](res *JobTestResult, taskRef taskid.TaskReference[T]) (T, bool) {
	if res == nil || res.taskResults == nil {
		var zero T
		return zero, false
	}
	val, ok := res.taskResults[taskRef.ReferenceIDString()]
	if !ok {
		var zero T
		return zero, false
	}
	typedVal, ok := val.(T)
	if !ok {
		var zero T
		return zero, false
	}
	return typedVal, true
}

// JobTestHarness executes inspection jobs in Record or Replay mode.
type JobTestHarness struct {
	t              testing.TB
	server         *coreinspection.InspectionTaskServer
	cfg            *JobTestConfig
	fixtureDir     string
	cpuProfilePath string
	memProfilePath string
	isRecordMode   bool
	isCPUProfile   bool
	isMemProfile   bool
	stubsInitOnce  sync.Once
	stubsInitErr   error

	cpuProfileOnce sync.Once
	memProfileOnce sync.Once
}

// sanitizeTestName replaces slashes and special characters in test names to create safe directory paths.
func sanitizeTestName(testName string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		" ", "_",
	)
	return replacer.Replace(testName)
}

// NewJobTestHarness creates a new JobTestHarness instance with paths derived from the test name.
func NewJobTestHarness(t testing.TB, server *coreinspection.InspectionTaskServer, cfg *JobTestConfig) *JobTestHarness {
	fixtureDir := filepath.Join("testdata", "fixtures", sanitizeTestName(t.Name()))

	isRecord := false
	if testflags.RecordTaskResults != nil && *testflags.RecordTaskResults {
		isRecord = true
	} else if os.Getenv("KHI_RECORD_TASK_RESULTS") == "1" {
		isRecord = true
	}

	isCPU := false
	if testflags.TaskCPUProfile != nil && *testflags.TaskCPUProfile {
		isCPU = true
	} else if os.Getenv("KHI_TASK_CPUPROFILE") == "1" {
		isCPU = true
	}

	isMem := false
	if testflags.TaskMemProfile != nil && *testflags.TaskMemProfile {
		isMem = true
	} else if os.Getenv("KHI_TASK_MEMPROFILE") == "1" {
		isMem = true
	}

	var cpuPath, memPath string
	if isCPU {
		cpuPath = filepath.Join("pprof", sanitizeTestName(t.Name()), "cpu.pprof")
	}
	if isMem {
		memPath = filepath.Join("pprof", sanitizeTestName(t.Name()), "mem.pprof")
	}

	return &JobTestHarness{
		t:              t,
		server:         server,
		cfg:            cfg,
		fixtureDir:     fixtureDir,
		cpuProfilePath: cpuPath,
		memProfilePath: memPath,
		isRecordMode:   isRecord,
		isCPUProfile:   isCPU,
		isMemProfile:   isMem,
	}
}

// IsRecordMode returns true if the current execution is configured to record fixtures.
func (h *JobTestHarness) IsRecordMode() bool {
	return h.isRecordMode
}

// Run executes the job in Record mode if the record flag is set, otherwise in Replay mode.
func (h *JobTestHarness) Run(ctx context.Context) (*JobTestResult, error) {
	if h.isRecordMode {
		return h.Record(ctx)
	}
	return h.Replay(ctx)
}

// Record runs the inspection job normally, intercepts the target task outputs, and saves them to fixtures.
func (h *JobTestHarness) Record(ctx context.Context) (*JobTestResult, error) {
	inspectionID, err := h.server.CreateInspection(h.cfg.InspectionType)
	if err != nil {
		return nil, fmt.Errorf("failed to create inspection: %w", err)
	}
	runner := h.server.GetInspection(inspectionID)
	if runner == nil {
		return nil, fmt.Errorf("failed to get inspection runner for %s", inspectionID)
	}

	if len(h.cfg.InspectionFeatures) > 0 {
		if err := runner.SetFeatureList(h.cfg.InspectionFeatures); err != nil {
			return nil, fmt.Errorf("failed to set feature list: %w", err)
		}
	}

	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var mu sync.Mutex
	capturedResults := make(map[string]any)

	recordInterceptor := newRecordInspectionInterceptor(
		h.cfg.RecordedTasks,
		func() {
			_ = runner.Cancel()
		},
		func(results map[string]any) {
			mu.Lock()
			for k, v := range results {
				capturedResults[k] = v
			}
			mu.Unlock()
		},
	)
	runner.AddInterceptors(recordInterceptor)

	req := &inspectioncore_contract.InspectionRequest{
		Values: h.cfg.InspectionValues,
	}

	if err := runner.Run(cancelCtx, req); err != nil {
		return nil, fmt.Errorf("failed to start runner: %w", err)
	}

	<-runner.Wait()

	mu.Lock()
	resultsToSave := make(map[string]any, len(capturedResults))
	for k, v := range capturedResults {
		resultsToSave[k] = v
	}
	mu.Unlock()

	if len(resultsToSave) < len(h.cfg.RecordedTasks) {
		return nil, fmt.Errorf("recorded %d of %d requested tasks", len(resultsToSave), len(h.cfg.RecordedTasks))
	}

	if err := saveRecordedTaskResults(h.fixtureDir, resultsToSave); err != nil {
		return nil, fmt.Errorf("failed to save recorded task results: %w", err)
	}

	return &JobTestResult{
		taskResults: resultsToSave,
	}, nil
}

// Replay injects recorded stub tasks and executes the inspection in isolated Replay mode.
func (h *JobTestHarness) Replay(ctx context.Context) (*JobTestResult, error) {
	// Register stub tasks for each recorded task once
	h.stubsInitOnce.Do(func() {
		for _, ref := range h.cfg.RecordedTasks {
			targetType, ok := ResolveTaskTypeFromTaskSet(h.server.RootTaskSet, ref)
			if !ok {
				targetType = reflect.TypeOf([]*log.Log{})
			}
			val, err := loadRecordedTaskResultForType(h.fixtureDir, ref, targetType)
			if err != nil {
				h.stubsInitErr = fmt.Errorf("failed to load fixture for %s: %w", ref.ReferenceIDString(), err)
				return
			}
			stubTask := newReplayStubTask(ref, val)
			if err := h.server.AddTask(stubTask); err != nil {
				h.stubsInitErr = fmt.Errorf("failed to add stub task for %s: %w", ref.ReferenceIDString(), err)
				return
			}
		}
	})
	if h.stubsInitErr != nil {
		return nil, h.stubsInitErr
	}

	inspectionID, err := h.server.CreateInspection(h.cfg.InspectionType)
	if err != nil {
		return nil, fmt.Errorf("failed to create inspection: %w", err)
	}
	runner := h.server.GetInspection(inspectionID)
	if runner == nil {
		return nil, fmt.Errorf("failed to get inspection runner for %s", inspectionID)
	}

	if len(h.cfg.InspectionFeatures) > 0 {
		if err := runner.SetFeatureList(h.cfg.InspectionFeatures); err != nil {
			return nil, fmt.Errorf("failed to set feature list: %w", err)
		}
	}

	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var mu sync.Mutex
	capturedResults := make(map[string]any)

	replayInterceptor := newReplayInspectionInterceptor(
		h,
		func() {
			_ = runner.Cancel()
		},
		func(result any) {
			if h.cfg.TargetTask != nil {
				mu.Lock()
				capturedResults[h.cfg.TargetTask.ReferenceIDString()] = result
				mu.Unlock()
			}
		},
	)
	runner.AddInterceptors(replayInterceptor)

	req := &inspectioncore_contract.InspectionRequest{
		Values: h.cfg.InspectionValues,
	}

	if err := runner.Run(cancelCtx, req); err != nil {
		return nil, fmt.Errorf("failed to start runner: %w", err)
	}

	<-runner.Wait()

	runResult, _ := runner.Result()

	// If early termination occurred for TargetTask, context canceled error is normal
	if h.cfg.TargetTask != nil {
		mu.Lock()
		defer mu.Unlock()
		return &JobTestResult{
			InspectionResult: runResult,
			taskResults:      capturedResults,
		}, nil
	}

	if runResult == nil && errors.Is(cancelCtx.Err(), context.Canceled) {
		return &JobTestResult{
			taskResults: capturedResults,
		}, nil
	}

	return &JobTestResult{
		InspectionResult: runResult,
		taskResults:      capturedResults,
	}, nil
}
