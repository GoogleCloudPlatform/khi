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
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
)

var mockLogToTimelineMapperPrevTaskID = taskid.NewDefaultImplementationID[LogGroupMap]("mock-timeline-mapper-v2-prev")
var mockLogSerializerV2PrevTaskID = taskid.NewDefaultImplementationID[[]*log.Log]("mock-timeline-mapper-v2-prev-log-serializer")

type mockLogToTimelineMapperGroupData struct {
	ProcessedLogs int
}

type mockLogToTimelineMapper struct {
	passCount int
	path      *khifilev6.TimelinePath
}

func (m *mockLogToTimelineMapper) GroupedLogTask() taskid.TaskReference[LogGroupMap] {
	return mockLogToTimelineMapperPrevTaskID.Ref()
}

func (m *mockLogToTimelineMapper) LogIngesterTask() taskid.TaskReference[[]*log.Log] {
	return mockLogSerializerV2PrevTaskID.Ref()
}

func (m *mockLogToTimelineMapper) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

func (m *mockLogToTimelineMapper) PassCount() int {
	return m.passCount
}

func (m *mockLogToTimelineMapper) PreProcessLogByGroup(ctx context.Context, passIndex int, l *log.Log, prevGroupData mockLogToTimelineMapperGroupData) (mockLogToTimelineMapperGroupData, error) {
	return mockLogToTimelineMapperGroupData{
		ProcessedLogs: prevGroupData.ProcessedLogs + 1,
	}, nil
}

func (m *mockLogToTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, prevGroupData mockLogToTimelineMapperGroupData) (*khifilev6.TimelineChangeSet, mockLogToTimelineMapperGroupData, error) {
	shouldErr := l.ReadBoolOrDefault("error", false)
	if shouldErr {
		return nil, prevGroupData, fmt.Errorf("test error")
	}
	shouldSkip := l.ReadBoolOrDefault("skip", false)
	if shouldSkip {
		return nil, mockLogToTimelineMapperGroupData{
			ProcessedLogs: prevGroupData.ProcessedLogs + 1,
		}, nil
	}

	cs := khifilev6.NewTimelineChangeSet(l)
	cs.AddEvent(m.path)

	return cs, mockLogToTimelineMapperGroupData{
		ProcessedLogs: prevGroupData.ProcessedLogs + 1,
	}, nil
}

var _ LogToTimelineMapper[mockLogToTimelineMapperGroupData] = (*mockLogToTimelineMapper)(nil)

func TestLogToTimelineMapperTask(t *testing.T) {
	type testLog struct {
		yaml         string
		shouldIngest bool
	}

	type testGroup struct {
		group string
		logs  []testLog
	}

	testCases := []struct {
		desc            string
		taskMode        inspectioncore_contract.InspectionTaskModeType
		prevLogGroupMap []testGroup
		passCount       int
		cancelContext   bool
		wantError       bool
		wantResult      func(path *khifilev6.TimelinePath) TimelineMapperResult
	}{
		{
			desc:     "DryRun mode",
			taskMode: inspectioncore_contract.TaskModeDryRun,
			prevLogGroupMap: []testGroup{
				{
					group: "group1",
					logs: []testLog{
						{
							yaml:         `{"apiVersion": "v1", "kind": "Pod", "namespace": "default", "name": "pod-1"}`,
							shouldIngest: false,
						},
					},
				},
			},
			passCount: 1,
			wantError: false,
			wantResult: func(path *khifilev6.TimelinePath) TimelineMapperResult {
				return NewTimelineMapperResult()
			},
		},
		{
			desc:     "Normal execution with some skipped logs and 2 passes",
			taskMode: inspectioncore_contract.TaskModeRun,
			prevLogGroupMap: []testGroup{
				{
					group: "group1",
					logs: []testLog{
						{
							yaml:         `{"apiVersion": "v1", "kind": "Pod", "namespace": "default", "name": "pod-1"}`,
							shouldIngest: true,
						},
						{
							yaml:         `{"apiVersion": "v1", "kind": "Pod", "namespace": "default", "name": "pod-2", "skip": true}`,
							shouldIngest: false,
						},
					},
				},
			},
			passCount: 2,
			wantError: false,
			wantResult: func(path *khifilev6.TimelinePath) TimelineMapperResult {
				res := NewTimelineMapperResult()
				res.Events[path] = 1
				return res
			},
		},
		{
			desc:     "Execution with error in one log",
			taskMode: inspectioncore_contract.TaskModeRun,
			prevLogGroupMap: []testGroup{
				{
					group: "group1",
					logs: []testLog{
						{
							yaml:         `{"apiVersion": "v1", "kind": "Pod", "namespace": "default", "name": "pod-1"}`,
							shouldIngest: false,
						},
						{
							yaml:         `{"apiVersion": "v1", "kind": "Pod", "namespace": "default", "name": "pod-2", "error": true}`,
							shouldIngest: false,
						},
					},
				},
			},
			passCount: 0,
			wantError: true,
			wantResult: func(path *khifilev6.TimelinePath) TimelineMapperResult {
				return NewTimelineMapperResult()
			},
		},
		{
			desc:     "Execution with context cancelled",
			taskMode: inspectioncore_contract.TaskModeRun,
			prevLogGroupMap: []testGroup{
				{
					group: "group1",
					logs: []testLog{
						{
							yaml:         `{"apiVersion": "v1", "kind": "Pod", "namespace": "default", "name": "pod-1"}`,
							shouldIngest: false,
						},
					},
				},
			},
			passCount:     0,
			cancelContext: true,
			wantError:     true,
			wantResult: func(path *khifilev6.TimelinePath) TimelineMapperResult {
				return NewTimelineMapperResult()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tid := taskid.NewDefaultImplementationID[TimelineMapperResult]("mock-timeline-mapper-v2")

			ctx := context.Background()
			ctx = inspectiontest.WithDefaultTestInspectionTaskContext(ctx)

			builder := khifilev6.NewBuilder()
			ctx = khictx.WithValue(ctx, inspectioncore_contract.Builder, builder)

			prevGroupMap := make(LogGroupMap)
			var shouldHaveItems bool

			for _, tg := range tc.prevLogGroupMap {
				var logs []*log.Log
				for _, tl := range tg.logs {
					l := mustNewLogFromYAML(t, tl.yaml)
					logs = append(logs, l)
					if tl.shouldIngest {
						shouldHaveItems = true
					}

					// Populate log accumulator with log metadata before mapping
					severityID := uint32(1)
					logTypeID := uint32(2)
					_ = builder.LogAccumulator.AddLog(&khifilev6.StagingLog{
						Log:       l,
						Summary:   "test",
						Timestamp: time.Now(),
						Severity:  &pb.Severity{Id: &severityID},
						LogType:   &pb.LogType{Id: &logTypeID},
					})
				}
				prevGroupMap[tg.group] = &LogGroup{
					Group: tg.group,
					Logs:  logs,
				}
			}

			idGen := &khifilev6.IDGenerator{}
			pool := khifilev6.NewInternPool(idGen)
			pathPool := khifilev6.NewTimelinePathPool(idGen, pool)
			timelineTypeID := uint32(3)
			timelineType := &pb.TimelineType{Id: &timelineTypeID}
			path := pathPool.Get(nil, khifilev6.PathSegment{Name: "test-path", Type: timelineType})

			mapper := &mockLogToTimelineMapper{
				passCount: tc.passCount,
				path:      path,
			}
			task := NewLogToTimelineMapperTask(tid, mapper)

			if tc.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}

			gotResult, _, err := inspectiontest.RunInspectionTask(ctx, task, tc.taskMode, map[string]any{}, tasktest.NewTaskDependencyValuePair(mockLogToTimelineMapperPrevTaskID.Ref(), prevGroupMap))
			if (err != nil) != tc.wantError {
				t.Fatalf("RunInspectionTask() error = %v, wantError %v", err, tc.wantError)
			}

			if tc.cancelContext {
				var expectedErr = context.Canceled
				if err == nil || err.Error() != expectedErr.Error() {
					t.Errorf("RunInspectionTask() error = %v, want %v", err, expectedErr)
				}
			}

			if diff := cmp.Diff(tc.wantResult(path), gotResult); diff != "" {
				t.Errorf("TimelineMapperResult mismatch (-want +got):\n%s", diff)
			}

			if !tc.wantError {
				// Verify timeline items inside the accumulator registry
				timelineBuilder := builder.TimelineAccumulator.GetBuilder(path)
				hasItems := timelineBuilder.HasItems()

				if shouldHaveItems {
					if !hasItems {
						t.Error("expected timeline builder to have items successfully flushed, but had none")
					}
				} else {
					if hasItems {
						t.Error("expected timeline builder to remain empty, but items were found")
					}
				}
			}
		})
	}
}
