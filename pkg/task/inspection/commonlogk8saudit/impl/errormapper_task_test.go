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

package commonlogk8saudit_impl

import (
	"context"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
)

func TestNonSuccessLogLogToTimelineMapperTaskSetting_ProcessLogByGroup(t *testing.T) {
	// 1. Set up the mock Builder and construct comparison paths hierarchically.
	builder := khifilev6.NewTestBuilder(id.NewGenerator())
	cluster := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{Name: "k8s", Type: inspectioncore_contract.TimelineTypeK8sCluster})
	api := builder.TimelineAccumulator.GetPath(cluster, khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore_contract.TimelineTypeAPIVersion})
	kind := builder.TimelineAccumulator.GetPath(api, khifilev6.PathSegment{Name: "pod", Type: inspectioncore_contract.TimelineTypeKind})
	ns := builder.TimelineAccumulator.GetPath(kind, khifilev6.PathSegment{Name: "default", Type: inspectioncore_contract.TimelineTypeNamespace})

	parentPath := builder.TimelineAccumulator.GetPath(ns, khifilev6.PathSegment{Name: "test-pod", Type: inspectioncore_contract.TimelineTypeResource})
	otherSubresourcePath := builder.TimelineAccumulator.GetPath(parentPath, khifilev6.PathSegment{Name: "proxy", Type: inspectioncore_contract.TimelineTypeSubresource})

	testCases := []struct {
		name            string
		subresourceName string
		wantPath        *khifilev6.TimelinePath
	}{
		{
			name:            "standard pod mapping",
			subresourceName: "",
			wantPath:        parentPath,
		},
		{
			name:            "status subresource mapped to parent",
			subresourceName: "status",
			wantPath:        parentPath,
		},
		{
			name:            "non-status subresource proxy NOT mapped to parent",
			subresourceName: "proxy",
			wantPath:        otherSubresourcePath,
		},
	}

	mapperSetting := &nonSuccessLogLogToTimelineMapperTaskSetting{
		subresourceMapToWriteToParent: map[string]struct{}{
			"status":   {},
			"finalize": {},
			"approve":  {},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logObj := testlog.NewMockLog(
				time.Now(),
				commonlogk8saudit_contract.K8sAuditLogFieldSet{
					APIVersion:      "core/v1",
					PluralKind:      "pods",
					Namespace:       "default",
					ResourceName:    "test-pod",
					SubresourceName: tc.subresourceName,
					ClusterName:     "k8s",
				},
			)
			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

			cs, _, err := mapperSetting.ProcessLogByGroup(ctx, logObj, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLogByGroup() failed: %v", err)
			}

			testchangeset.AssertTimeline(t, cs).
				HasEvent(tc.wantPath)
		})
	}
}

func TestNonSuccessLogLogToTimelineMapperTask(t *testing.T) {
	testCases := []struct {
		name     string
		logYamls []string
	}{
		{
			name: "maps non-success log to timeline using extractor dependency",
			logYamls: []string{
				`id: log-1`,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
			idGen := khictx.MustGetValue(ctx, inspectioncore_contract.IDGenerator)
			logs := make([]*log.Log, 0, len(tc.logYamls))
			for _, yml := range tc.logYamls {
				l, err := log.NewLogFromYAMLString(idGen, yml)
				if err != nil {
					t.Fatalf("failed to parse yaml: %v", err)
				}
				logs = append(logs, l)
			}

			builder := khictx.MustGetValue(ctx, inspectioncore_contract.Builder)
			for _, l := range logs {
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

			mockExtractor := commonlogk8saudit_contract.K8sAuditLogExtractor(func(reader *structured.NodeReader) (*commonlogk8saudit_contract.K8sAuditLogFieldSet, error) {
				return &commonlogk8saudit_contract.K8sAuditLogFieldSet{
					APIVersion:   "core/v1",
					PluralKind:   "pods",
					Namespace:    "default",
					ResourceName: "pod-1",
					ClusterName:  "k8s",
				}, nil
			})

			logGroupMap := inspectiontaskbase.LogGroupMap{
				"group1": &inspectiontaskbase.LogGroup{
					Logs: logs,
				},
			}

			_, _, err := inspectiontest.RunInspectionTask(
				ctx,
				NonSuccessLogLogToTimelineMapperTask,
				inspectioncore_contract.TaskModeRun,
				map[string]any{},
				tasktest.NewTaskDependencyValuePair(commonlogk8saudit_contract.NonSuccessLogGrouperTaskID.Ref(), logGroupMap),
				tasktest.NewTaskDependencyValuePair(commonlogk8saudit_contract.K8sAuditLogIngesterTaskID.Ref(), struct{}{}),
				tasktest.NewTaskDependencyValuePair(commonlogk8saudit_contract.K8sAuditLogExtractorRef, mockExtractor),
			)
			if err != nil {
				t.Fatalf("RunInspectionTask returned an unexpected error: %v", err)
			}

			cluster := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{Name: "k8s", Type: inspectioncore_contract.TimelineTypeK8sCluster})
			api := builder.TimelineAccumulator.GetPath(cluster, khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore_contract.TimelineTypeAPIVersion})
			kind := builder.TimelineAccumulator.GetPath(api, khifilev6.PathSegment{Name: "pod", Type: inspectioncore_contract.TimelineTypeKind})
			ns := builder.TimelineAccumulator.GetPath(kind, khifilev6.PathSegment{Name: "default", Type: inspectioncore_contract.TimelineTypeNamespace})
			wantPath := builder.TimelineAccumulator.GetPath(ns, khifilev6.PathSegment{Name: "pod-1", Type: inspectioncore_contract.TimelineTypeResource})

			if !builder.TimelineAccumulator.HasEvent(wantPath) {
				t.Errorf("expected timeline %v to have events, but none found", wantPath)
			}
		})
	}
}
