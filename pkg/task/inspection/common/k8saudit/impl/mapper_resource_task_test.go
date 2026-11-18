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

package k8saudit_impl

import (
	"context"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
)

func TestResourceRevisionLogToTimelineMapperTask(t *testing.T) {
	ctx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
	builder := khictx.MustGetValue(ctx, inspectioncore.Builder)

	testTime := time.Date(2023, 10, 26, 10, 0, 0, 0, time.UTC)
	logObj := testlog.NewMockLog(
		testTime,
		k8saudit.K8sAuditLogFieldSet{
			Principal:    "admin",
			APIVersion:   "core/v1",
			PluralKind:   "pods",
			ResourceName: "pod-1",
			Namespace:    "default",
			ClusterName:  "k8s",
			Verb:         k8saudit.VerbCreate,
		},
	)

	severityID := uint32(1)
	logTypeID := uint32(2)
	_ = builder.LogAccumulator.AddLog(&khifilev6.StagingLog{
		Log:       logObj,
		Summary:   "create pod-1",
		Timestamp: testTime,
		Severity:  &pb.Severity{Id: &severityID},
		LogType:   &pb.LogType{Id: &logTypeID},
	})

	bodyNode, err := structured.FromYAML("metadata:\n  uid: \"uid-1\"\n")
	if err != nil {
		t.Fatalf("failed to parse YAML: %v", err)
	}
	resIdentity := &k8saudit.ResourceIdentity{
		APIVersion: "core/v1",
		Kind:       "pod",
		Namespace:  "default",
		Name:       "pod-1",
	}
	groupedLogs := k8saudit.ResourceManifestLogGroupMap{
		resIdentity.String(): &k8saudit.ResourceManifestLogGroup{
			Resource: resIdentity,
			Logs: []*k8saudit.ResourceManifestLog{
				{
					Log:                logObj,
					ResourceBodyReader: structured.NewNodeReader(bodyNode),
				},
			},
		},
	}

	_, _, err = inspectiontest.Run(
		t,
		ctx,
		resourceRevisionLogToTimelineMapperTask,
		inspectioncore.TaskModeRun,
		map[string]any{},
		tasktest.Given(k8saudit.ResourceLifetimeTrackerTaskID.Ref(), groupedLogs),
		tasktest.Given(k8saudit.InitialResourceStateProviderRef, newTestInitialResourceStateProvider(t, "")),
	)
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	cluster := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{Name: "k8s", Type: inspectioncore.TimelineTypeK8sCluster})
	api := builder.TimelineAccumulator.GetPath(cluster, khifilev6.PathSegment{Name: "core/v1", Type: inspectioncore.TimelineTypeAPIVersion})
	kind := builder.TimelineAccumulator.GetPath(api, khifilev6.PathSegment{Name: "pod", Type: inspectioncore.TimelineTypeKind})
	ns := builder.TimelineAccumulator.GetPath(kind, khifilev6.PathSegment{Name: "default", Type: inspectioncore.TimelineTypeNamespace})
	wantPath := builder.TimelineAccumulator.GetPath(ns, khifilev6.PathSegment{Name: "pod-1", Type: inspectioncore.TimelineTypeResource})

	protoItems := builder.TimelineAccumulator.GetBuilder(wantPath).ToProto()
	if protoItems == nil || len(protoItems.GetRevisions()) != 1 {
		t.Fatalf("got revisions = %v, want 1 revision on %v", protoItems.GetRevisions(), wantPath)
	}
}
