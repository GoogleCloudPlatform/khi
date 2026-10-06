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

package inspectiontaskbase

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
)

var (
	pathMockError = structured.CompileFieldPath("error")
	pathMockSkip  = structured.CompileFieldPath("skip")
)

var mockLogToTimelineMapperPrevTaskID = taskid.NewDefaultImplementationID[LogGroupMap]("mock-timeline-mapper-prev")
var mockLogSerializerPrevTaskID = taskid.NewDefaultImplementationID[struct{}]("mock-timeline-mapper-prev-log-serializer")

type inputPathMapperGroupData struct {
	ProcessedLogs int
}

// inputPathMapper is a TimelineMapper that reads the timeline path through an input handle.
type inputPathMapper struct {
	passCount int
	path      coretask.Input[*khifilev6.TimelinePath]
}

func (m *inputPathMapper) PassCount() int {
	return m.passCount
}

func (m *inputPathMapper) PreProcessLogByGroup(ctx context.Context, passIndex int, l *log.Log, prevGroupData inputPathMapperGroupData) (inputPathMapperGroupData, error) {
	return inputPathMapperGroupData{
		ProcessedLogs: prevGroupData.ProcessedLogs + 1,
	}, nil
}

func (m *inputPathMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData inputPathMapperGroupData) (*khifilev6.TimelineChangeSet, inputPathMapperGroupData, error) {
	if l.ReadBoolOrDefault(pathMockError, false) {
		return nil, prevGroupData, fmt.Errorf("test error")
	}
	nextGroupData := inputPathMapperGroupData{
		ProcessedLogs: prevGroupData.ProcessedLogs + 1,
	}
	if l.ReadBoolOrDefault(pathMockSkip, false) {
		return nil, nextGroupData, nil
	}

	cs := khifilev6.NewTimelineChangeSet(l)
	cs.AddEvent(m.path.Get(ctx))
	return cs, nextGroupData, nil
}

var _ TimelineMapper[inputPathMapperGroupData] = (*inputPathMapper)(nil)

func TestDefineLogToTimelineMapperTask(t *testing.T) {
	timelinePathTaskID := taskid.NewDefaultImplementationID[*khifilev6.TimelinePath]("mock-timeline-path")
	inputs := TimelineMapperInputs{
		LogIngester: mockLogSerializerPrevTaskID.Ref(),
		GroupedLogs: mockLogToTimelineMapperPrevTaskID.Ref(),
	}
	wantInputs := []string{
		"ordering mock-timeline-mapper-prev-log-serializer",
		"required mock-timeline-mapper-prev",
		"required mock-timeline-path",
	}

	testCases := []struct {
		desc          string
		taskMode      inspectioncore.InspectionTaskModeType
		logYAMLs      []string
		passCount     int
		cancelContext bool
		wantErrSubstr string
		wantItems     bool
	}{
		{
			desc:      "DryRun mode",
			taskMode:  inspectioncore.TaskModeDryRun,
			logYAMLs:  []string{`{"name": "pod-1"}`},
			passCount: 1,
			wantItems: false,
		},
		{
			desc:     "Normal execution with some skipped logs and 2 passes",
			taskMode: inspectioncore.TaskModeRun,
			logYAMLs: []string{
				`{"name": "pod-1"}`,
				`{"name": "pod-2", "skip": true}`,
			},
			passCount: 2,
			wantItems: true,
		},
		{
			desc:     "Execution with only skipped logs",
			taskMode: inspectioncore.TaskModeRun,
			logYAMLs: []string{
				`{"name": "pod-1", "skip": true}`,
			},
			wantItems: false,
		},
		{
			desc:     "Execution with error in one log",
			taskMode: inspectioncore.TaskModeRun,
			logYAMLs: []string{
				`{"name": "pod-1"}`,
				`{"name": "pod-2", "error": true}`,
			},
			wantErrSubstr: "test error",
		},
		{
			desc:          "Execution with context cancelled",
			taskMode:      inspectioncore.TaskModeRun,
			logYAMLs:      []string{`{"name": "pod-1"}`},
			cancelContext: true,
			wantErrSubstr: context.Canceled.Error(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			task := DefineLogToTimelineMapperTask(taskid.NewDefaultImplementationID[struct{}]("mock-timeline-mapper"), inputs, func(b *coretask.Binder) TimelineMapper[inputPathMapperGroupData] {
				return &inputPathMapper{
					passCount: tc.passCount,
					path:      coretask.Use(b, timelinePathTaskID.Ref()),
				}
			})
			if diff := cmp.Diff(wantInputs, describeInputs(task.Inputs())); diff != "" {
				t.Errorf("Inputs() mismatch (-want +got):\n%s", diff)
			}

			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			idGen := khictx.MustGetValue(ctx, inspectioncore.IDGenerator)
			builder := khictx.MustGetValue(ctx, inspectioncore.Builder)

			var logs []*log.Log
			for _, logYAML := range tc.logYAMLs {
				l := mustNewLogFromYAML(t, ctx, logYAML)
				logs = append(logs, l)
				// The mapper flushes timeline changes for logs whose metadata is already ingested.
				severityID := uint32(1)
				logTypeID := uint32(2)
				if err := builder.LogAccumulator.AddLog(&khifilev6.StagingLog{
					Log:       l,
					Summary:   "test",
					Timestamp: time.Now(),
					Severity:  &pb.Severity{Id: &severityID},
					LogType:   &pb.LogType{Id: &logTypeID},
				}); err != nil {
					t.Fatalf("failed to add log to the log accumulator: %v", err)
				}
			}
			groupedLogs := LogGroupMap{"group1": {Group: "group1", Logs: logs}}

			pathPool := khifilev6.NewTimelinePathPool(idGen, khifilev6.NewTestInternPool(idGen))
			timelineTypeID := uint32(3)
			path := pathPool.Get(nil, khifilev6.PathSegment{Name: "test-path", Type: &pb.TimelineType{Id: &timelineTypeID}})

			if tc.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			_, _, err := inspectiontest.Run(t, ctx, task, tc.taskMode, map[string]any{},
				tasktest.Given(mockLogToTimelineMapperPrevTaskID.Ref(), groupedLogs),
				tasktest.Given(timelinePathTaskID.Ref(), path),
			)
			if tc.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Fatalf("Run() error = %v, want error containing %q", err, tc.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Run() returned an unexpected error: %v", err)
			}

			if got := builder.TimelineAccumulator.GetBuilder(path).HasItems(); got != tc.wantItems {
				t.Errorf("HasItems() = %v, want %v", got, tc.wantItems)
			}
		})
	}
}
