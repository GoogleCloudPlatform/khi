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

package caik8s_impl

import (
	"testing"
	"time"

	assetpb "cloud.google.com/go/asset/apiv1/assetpb"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/caik8s"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// newLogFromMap builds a log from data.
func newLogFromMap(t *testing.T, generator *id.Generator, timestamp time.Time, data map[string]any) *log.Log {
	t.Helper()
	node, err := structured.FromGoValue(data, &structured.AlphabeticalGoMapKeyOrderProvider{})
	if err != nil {
		t.Fatalf("failed to build node from map: %v", err)
	}
	return log.NewLogWithTimestamp(generator, structured.NewNodeReader(node), timestamp)
}

func TestFormatClusterResourceLogSummary(t *testing.T) {
	testCases := []struct {
		name     string
		identity *k8saudit.ResourceIdentity
		want     string
	}{
		{
			name: "formats summary for pod identity",
			identity: &k8saudit.ResourceIdentity{
				Kind: "pod",
				Name: "pod-1",
			},
			want: "CAI resource snapshot: pod/pod-1",
		},
		{
			name: "formats summary for deployment identity",
			identity: &k8saudit.ResourceIdentity{
				Kind: "deployment",
				Name: "frontend",
			},
			want: "CAI resource snapshot: deployment/frontend",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatClusterResourceLogSummary(tc.identity)
			if got != tc.want {
				t.Errorf("formatClusterResourceLogSummary() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestClusterResourceSuite_LogIngesterTask(t *testing.T) {
	testTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	generator := id.NewGenerator()

	resourceDataLog := newLogFromMap(t, generator, testTime, map[string]any{
		"asset": map[string]any{
			"resource": map[string]any{
				"data": map[string]any{
					"apiVersion": "v1",
					"kind":       "pod",
					"metadata": map[string]any{
						"name":      "pod-1",
						"namespace": "default",
					},
				},
			},
		},
	})

	noResourceDataLog := newLogFromMap(t, generator, testTime, map[string]any{})

	testCases := []struct {
		name string
		log  *log.Log
	}{
		{
			name: "ingests resource data log",
			log:  resourceDataLog,
		},
		{
			name: "ingests log without resource data",
			log:  noResourceDataLog,
		},
	}

	ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
	builder := khictx.MustGetValue(ctx, inspectioncore.Builder)

	_, _, err := inspectiontest.RunInspectionTask(ctx, ClusterResourceSuite.LogIngesterTask, inspectioncore.TaskModeRun, map[string]any{},
		tasktest.NewTaskDependencyValuePair(caik8s.ClusterResourceTaskIDs.RawLog.Ref(), []*log.Log{resourceDataLog, noResourceDataLog}),
	)
	if err != nil {
		t.Fatalf("LogIngesterTask failed: %v", err)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			confirmedID, ok := builder.LogAccumulator.ResolveLogID(tc.log.ID)
			if !ok || confirmedID == 0 {
				t.Errorf("expected log %d to be accumulated by LogIngesterTask with non-zero ID, got confirmedID=%d, ok=%v", tc.log.ID, confirmedID, ok)
			}
		})
	}
}

func TestClusterResourceSuite_RawLogTask(t *testing.T) {
	testTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)

	podManifestStruct, err := structpb.NewStruct(map[string]any{
		"apiVersion": "core/v1",
		"kind":       "pod",
		"metadata": map[string]any{
			"name":      "pod-sample",
			"namespace": "default",
		},
	})
	if err != nil {
		t.Fatalf("failed to create structpb: %v", err)
	}

	snapshotWithResourceData := &gcpcommon.CAIAssetSnapshot{
		TemporalAsset: &assetpb.TemporalAsset{
			Asset: &assetpb.Asset{
				Name:      "//container.googleapis.com/projects/test/locations/us-central1/clusters/c1/k8s/namespaces/default/pods/pod-sample",
				AssetType: "k8s.io/Pod",
				Resource: &assetpb.Resource{
					Data: podManifestStruct,
				},
			},
		},
	}

	snapshotWithoutResourceData := &gcpcommon.CAIAssetSnapshot{
		TemporalAsset: &assetpb.TemporalAsset{
			Asset: &assetpb.Asset{
				Name:      "//container.googleapis.com/projects/test/locations/us-central1/clusters/c1/k8s/namespaces/default/pods/pod-asset",
				AssetType: "k8s.io/Pod",
			},
		},
	}

	windowStartTime := testTime.Add(-1 * time.Hour)
	windowEndTime := testTime.Add(1 * time.Hour)

	deletedSnapshotWithWindow := &gcpcommon.CAIAssetSnapshot{
		TemporalAsset: &assetpb.TemporalAsset{
			Window: &assetpb.TimeWindow{
				StartTime: timestamppb.New(windowStartTime),
				EndTime:   timestamppb.New(windowEndTime),
			},
			Deleted: true,
			Asset: &assetpb.Asset{
				Name:      "//container.googleapis.com/projects/test/locations/us-central1/clusters/c1/k8s/namespaces/default/pods/pod-deleted",
				AssetType: "k8s.io/Pod",
			},
		},
	}
	podSampleIdentity := &k8saudit.ResourceIdentity{
		APIVersion: "core/v1",
		Kind:       "pod",
		Name:       "pod-sample",
		Namespace:  "default",
	}
	podAssetIdentity := &k8saudit.ResourceIdentity{
		APIVersion: "core/v1",
		Kind:       "pod",
		Name:       "pod-asset",
		Namespace:  "default",
	}
	podDeletedIdentity := &k8saudit.ResourceIdentity{
		APIVersion: "core/v1",
		Kind:       "pod",
		Name:       "pod-deleted",
		Namespace:  "default",
	}
	podWithoutTypeMetaStruct, err := structpb.NewStruct(map[string]any{
		"metadata": map[string]any{
			"name":      "pod-restored",
			"namespace": "default",
		},
	})
	if err != nil {
		t.Fatalf("failed to create structpb: %v", err)
	}

	snapshotWithoutTypeMeta := &gcpcommon.CAIAssetSnapshot{
		TemporalAsset: &assetpb.TemporalAsset{
			Asset: &assetpb.Asset{
				Name:      "//container.googleapis.com/projects/test/locations/us-central1/clusters/c1/k8s/namespaces/default/pods/pod-restored",
				AssetType: "k8s.io/Pod",
				Resource: &assetpb.Resource{
					Version: "v1",
					Data:    podWithoutTypeMetaStruct,
				},
			},
		},
	}

	podRestoredIdentity := &k8saudit.ResourceIdentity{
		APIVersion: "core/v1",
		Kind:       "pod",
		Name:       "pod-restored",
		Namespace:  "default",
	}

	podWithEncodedMetadataStruct, err := structpb.NewStruct(map[string]any{
		"metadata": map[string]any{
			"name":         "pod-metadata",
			"namespace":    "default",
			"clusterName":  "",
			"selfLink":     "",
			"generateName": "",
			"managedFields": []any{
				map[string]any{
					"manager":     "kubectl",
					"operation":   "Update",
					"subresource": "",
					"fieldsV1": map[string]any{
						"Raw": "eyJmOmRhdGEiOnt9fQ==", // {"f:data":{}}
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create structpb: %v", err)
	}

	snapshotWithEncodedMetadata := &gcpcommon.CAIAssetSnapshot{
		TemporalAsset: &assetpb.TemporalAsset{
			Asset: &assetpb.Asset{
				Name:      "//container.googleapis.com/projects/test/locations/us-central1/clusters/c1/k8s/namespaces/default/pods/pod-metadata",
				AssetType: "k8s.io/Pod",
				Resource: &assetpb.Resource{
					Version: "v1",
					Data:    podWithEncodedMetadataStruct,
				},
			},
		},
	}

	podMetadataIdentity := &k8saudit.ResourceIdentity{
		APIVersion: "core/v1",
		Kind:       "pod",
		Name:       "pod-metadata",
		Namespace:  "default",
	}

	testCases := []struct {
		name                   string
		taskMode               inspectioncore.InspectionTaskModeType
		snapshots              []*gcpcommon.CAIAssetSnapshot
		wantCount              int
		wantIdentity           *k8saudit.ResourceIdentity
		wantWindowStartTime    time.Time
		wantWindowEndTime      time.Time
		wantDeleted            bool
		wantHasResourceBody    bool
		wantManifestAPIVersion string
		wantManifestKind       string
		wantManifestMetadata   map[string]any
	}{
		{
			name:      "returns empty slice in dry run mode",
			taskMode:  inspectioncore.TaskModeDryRun,
			snapshots: []*gcpcommon.CAIAssetSnapshot{snapshotWithResourceData},
			wantCount: 0,
		},
		{
			name:                "converts snapshots to raw logs in run mode",
			taskMode:            inspectioncore.TaskModeRun,
			snapshots:           []*gcpcommon.CAIAssetSnapshot{snapshotWithResourceData},
			wantCount:           1,
			wantIdentity:        podSampleIdentity,
			wantHasResourceBody: true,
		},
		{
			name:                   "restores missing apiVersion and kind into resource body",
			taskMode:               inspectioncore.TaskModeRun,
			snapshots:              []*gcpcommon.CAIAssetSnapshot{snapshotWithoutTypeMeta},
			wantCount:              1,
			wantIdentity:           podRestoredIdentity,
			wantHasResourceBody:    true,
			wantManifestAPIVersion: "v1",
			wantManifestKind:       "Pod",
		},
		{
			name:                "decodes managedFields base64 fieldsV1 and strips empty default fields in metadata",
			taskMode:            inspectioncore.TaskModeRun,
			snapshots:           []*gcpcommon.CAIAssetSnapshot{snapshotWithEncodedMetadata},
			wantCount:           1,
			wantIdentity:        podMetadataIdentity,
			wantHasResourceBody: true,
			wantManifestMetadata: map[string]any{
				"name":      "pod-metadata",
				"namespace": "default",
				"managedFields": []any{
					map[string]any{
						"manager":   "kubectl",
						"operation": "Update",
						"fieldsV1": map[string]any{
							"f:data": map[string]any{},
						},
					},
				},
			},
		},
		{
			name:                "resolves identity from the asset name when the manifest is absent",
			taskMode:            inspectioncore.TaskModeRun,
			snapshots:           []*gcpcommon.CAIAssetSnapshot{snapshotWithoutResourceData},
			wantCount:           1,
			wantIdentity:        podAssetIdentity,
			wantHasResourceBody: false,
		},
		{
			name:                "serializes the validity window and tombstone flag readable by the log extractors",
			taskMode:            inspectioncore.TaskModeRun,
			snapshots:           []*gcpcommon.CAIAssetSnapshot{deletedSnapshotWithWindow},
			wantCount:           1,
			wantIdentity:        podDeletedIdentity,
			wantWindowStartTime: windowStartTime,
			wantWindowEndTime:   windowEndTime,
			wantDeleted:         true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			got, _, err := inspectiontest.RunInspectionTask(ctx, ClusterResourceSuite.RawLogTask, tc.taskMode, map[string]any{},
				tasktest.NewTaskDependencyValuePair(caik8s.ClusterResourceTaskIDs.Fetcher.Ref(), tc.snapshots),
			)
			if err != nil {
				t.Fatalf("RawLogTask failed: %v", err)
			}
			if len(got) != tc.wantCount {
				t.Errorf("len(got) = %d, want %d", len(got), tc.wantCount)
			}
			if tc.wantCount > 0 {
				if !got[0].Timestamp.Equal(tc.wantWindowStartTime) {
					t.Errorf("got[0].Timestamp = %v, want %v", got[0].Timestamp, tc.wantWindowStartTime)
				}
				identity, _ := extractK8sIdentity(got[0].NodeReader)
				if diff := cmp.Diff(tc.wantIdentity, identity); diff != "" {
					t.Errorf("extractK8sIdentity() mismatch (-want +got):\n%s", diff)
				}
				gotWindowStartTime, gotWindowEndTime, gotDeleted := gcpcommon.ExtractCAITimeWindow(got[0].NodeReader)
				if !gotWindowStartTime.Equal(tc.wantWindowStartTime) {
					t.Errorf("ExtractCAITimeWindow() startTime = %v, want %v", gotWindowStartTime, tc.wantWindowStartTime)
				}
				if !gotWindowEndTime.Equal(tc.wantWindowEndTime) {
					t.Errorf("ExtractCAITimeWindow() endTime = %v, want %v", gotWindowEndTime, tc.wantWindowEndTime)
				}
				if gotDeleted != tc.wantDeleted {
					t.Errorf("ExtractCAITimeWindow() isDeleted = %v, want %v", gotDeleted, tc.wantDeleted)
				}
				gotHasResourceBody := extractK8sResourceBody(got[0].NodeReader) != nil
				if gotHasResourceBody != tc.wantHasResourceBody {
					t.Errorf("extractK8sResourceBody() != nil = %v, want %v", gotHasResourceBody, tc.wantHasResourceBody)
				}
				if tc.wantManifestAPIVersion != "" {
					bodyNode := extractK8sResourceBody(got[0].NodeReader)
					if bodyNode == nil {
						t.Fatal("expected non-nil resource body")
					}
					bodyReader := structured.NewNodeReader(bodyNode)
					gotAPIVersion := bodyReader.ReadStringOrDefault(structured.CompileFieldPath("apiVersion"), "")
					if gotAPIVersion != tc.wantManifestAPIVersion {
						t.Errorf("manifest apiVersion = %v, want %v", gotAPIVersion, tc.wantManifestAPIVersion)
					}
				}
				if tc.wantManifestKind != "" {
					bodyNode := extractK8sResourceBody(got[0].NodeReader)
					if bodyNode == nil {
						t.Fatal("expected non-nil resource body")
					}
					bodyReader := structured.NewNodeReader(bodyNode)
					gotKind := bodyReader.ReadStringOrDefault(structured.CompileFieldPath("kind"), "")
					if gotKind != tc.wantManifestKind {
						t.Errorf("manifest kind = %v, want %v", gotKind, tc.wantManifestKind)
					}
				}
				if tc.wantManifestMetadata != nil {
					bodyNode := extractK8sResourceBody(got[0].NodeReader)
					if bodyNode == nil {
						t.Fatal("expected non-nil resource body")
					}
					bodyReader := structured.NewNodeReader(bodyNode)
					var gotMetadata map[string]any
					if err := structured.ReadReflect(bodyReader, structured.CompileFieldPath("metadata"), &gotMetadata); err != nil {
						t.Fatalf("failed to read manifest metadata: %v", err)
					}
					if diff := cmp.Diff(tc.wantManifestMetadata, gotMetadata); diff != "" {
						t.Errorf("manifest metadata mismatch (-want +got):\n%s", diff)
					}
				}
			}
		})
	}
}

func TestClusterResourceSuite_LogGrouperTask(t *testing.T) {
	testTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	generator := id.NewGenerator()

	resourceDataLog := newLogFromMap(t, generator, testTime, map[string]any{
		"asset": map[string]any{
			"resource": map[string]any{
				"data": map[string]any{
					"apiVersion": "core/v1",
					"kind":       "pod",
					"metadata": map[string]any{
						"name":      "pod-1",
						"namespace": "default",
					},
				},
			},
		},
	})
	resourceDataLogIdentity := &k8saudit.ResourceIdentity{
		APIVersion: "core/v1",
		Kind:       "pod",
		Name:       "pod-1",
		Namespace:  "default",
	}

	noResourceDataLog := newLogFromMap(t, generator, testTime, map[string]any{})

	testCases := []struct {
		name      string
		inputLogs []*log.Log
		wantKeys  []string
	}{
		{
			name:      "groups by resource identity string or unknown",
			inputLogs: []*log.Log{resourceDataLog, noResourceDataLog},
			wantKeys:  []string{resourceDataLogIdentity.String(), "unknown"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			groups, _, err := inspectiontest.RunInspectionTask(ctx, ClusterResourceSuite.LogGrouperTask, inspectioncore.TaskModeRun, map[string]any{},
				tasktest.NewTaskDependencyValuePair(caik8s.ClusterResourceTaskIDs.RawLog.Ref(), tc.inputLogs),
			)
			if err != nil {
				t.Fatalf("LogGrouperTask failed: %v", err)
			}
			if len(groups) != len(tc.wantKeys) {
				t.Errorf("len(groups) = %d, want %d", len(groups), len(tc.wantKeys))
			}
			for _, key := range tc.wantKeys {
				if _, ok := groups[key]; !ok {
					t.Errorf("group %q not found in result", key)
				}
			}
			logPointerComparer := cmp.Comparer(func(a, b *log.Log) bool {
				return a == b
			})
			if diff := cmp.Diff([]*log.Log{resourceDataLog}, groups[resourceDataLogIdentity.String()].Logs, logPointerComparer); diff != "" {
				t.Errorf("group %s mismatch (-want +got):\n%s", resourceDataLogIdentity.String(), diff)
			}
			if diff := cmp.Diff([]*log.Log{noResourceDataLog}, groups["unknown"].Logs, logPointerComparer); diff != "" {
				t.Errorf("group unknown mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestClusterResourceSuite_TimelineMapperTask(t *testing.T) {
	queryStartTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	assetWindowStartTime := queryStartTime.Add(-1 * time.Hour)
	generator := id.NewGenerator()

	activeLog := newLogFromMap(t, generator, assetWindowStartTime, map[string]any{
		"window": map[string]any{
			"startTime": assetWindowStartTime.Format(time.RFC3339Nano),
			"endTime":   queryStartTime.Add(1 * time.Hour).Format(time.RFC3339Nano),
		},
		"deleted": false,
		"asset": map[string]any{
			"resource": map[string]any{
				"data": map[string]any{
					"apiVersion": "core/v1",
					"kind":       "Pod",
					"metadata": map[string]any{
						"name":      "pod-1",
						"namespace": "default",
					},
				},
			},
		},
	})

	deletedLog := newLogFromMap(t, generator, assetWindowStartTime, map[string]any{
		"window": map[string]any{
			"startTime": assetWindowStartTime.Format(time.RFC3339Nano),
			"endTime":   queryStartTime.Add(1 * time.Hour).Format(time.RFC3339Nano),
		},
		"deleted": true,
		"asset": map[string]any{
			"resource": map[string]any{
				"data": map[string]any{
					"apiVersion": "core/v1",
					"kind":       "Pod",
					"metadata": map[string]any{
						"name":      "pod-deleted",
						"namespace": "default",
					},
				},
			},
		},
	})

	ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
	builder := khictx.MustGetValue(ctx, inspectioncore.Builder)

	severityID := uint32(1)
	logTypeID := uint32(2)
	for _, l := range []*log.Log{activeLog, deletedLog} {
		_ = builder.LogAccumulator.AddLog(&khifilev6.StagingLog{
			Log:       l,
			Summary:   "test",
			Timestamp: l.Timestamp,
			Severity:  &pb.Severity{Id: &severityID},
			LogType:   &pb.LogType{Id: &logTypeID},
		})
	}

	groupMap := inspectiontaskbase.LogGroupMap{
		"pod-1":       {Group: "pod-1", Logs: []*log.Log{activeLog}},
		"pod-deleted": {Group: "pod-deleted", Logs: []*log.Log{deletedLog}},
	}

	_, _, err := inspectiontest.RunInspectionTask(ctx, ClusterResourceSuite.TimelineMapperTask, inspectioncore.TaskModeRun, map[string]any{},
		tasktest.NewTaskDependencyValuePair(caik8s.ClusterResourceTaskIDs.LogGrouper.Ref(), groupMap),
		tasktest.NewTaskDependencyValuePair(caik8s.ClusterResourceTaskIDs.LogIngester.Ref(), struct{}{}),
		tasktest.NewTaskDependencyValuePair(k8scommon.ClusterIdentityTaskID.Ref(), k8scommon.GoogleCloudClusterIdentity{
			ProjectID:   "test-project",
			ClusterName: "test-cluster",
			Location:    "us-central1-a",
		}),
		tasktest.NewTaskDependencyValuePair(gcpcommon.InputStartTimeTaskID.Ref(), queryStartTime),
	)
	if err != nil {
		t.Fatalf("TimelineMapperTask failed: %v", err)
	}

	activePath := k8saudit.MustResourceTimeline(ctx, "test-cluster", &k8saudit.ResourceIdentity{
		APIVersion: "core/v1",
		Kind:       "pod",
		Name:       "pod-1",
		Namespace:  "default",
	})
	deletedPath := k8saudit.MustResourceTimeline(ctx, "test-cluster", &k8saudit.ResourceIdentity{
		APIVersion: "core/v1",
		Kind:       "pod",
		Name:       "pod-deleted",
		Namespace:  "default",
	})

	testCases := []struct {
		name          string
		path          *khifilev6.TimelinePath
		wantCount     int
		wantStateType *pb.RevisionState
		wantVerbType  *pb.Verb
	}{
		{
			name:          "stages revision for active resource",
			path:          activePath,
			wantCount:     1,
			wantStateType: caik8s.RevisionStateK8sResourceExistingFromCAI,
			wantVerbType:  k8saudit.VerbCreate,
		},
		{
			name:      "skips revision for deleted resource",
			path:      deletedPath,
			wantCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var revs []*pb.Revision
			if protoItems := builder.TimelineAccumulator.GetBuilder(tc.path).ToProto(); protoItems != nil {
				revs = protoItems.Revisions
			}
			if len(revs) != tc.wantCount {
				t.Fatalf("len(revs) = %d, want %d", len(revs), tc.wantCount)
			}
			if tc.wantCount > 0 {
				if revs[0].GetStateType() != tc.wantStateType.GetId() {
					t.Errorf("revs[0].GetStateType() = %d, want %d", revs[0].GetStateType(), tc.wantStateType.GetId())
				}
				if revs[0].GetVerbType() != tc.wantVerbType.GetId() {
					t.Errorf("revs[0].GetVerbType() = %d, want %d", revs[0].GetVerbType(), tc.wantVerbType.GetId())
				}
			}
		})
	}
}

func TestGKEResourceSuite_RawLogTask(t *testing.T) {
	testTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	clusterData, _ := structpb.NewStruct(map[string]any{
		"name": "test-cluster",
	})
	snapshot := &gcpcommon.CAIAssetSnapshot{
		TemporalAsset: &assetpb.TemporalAsset{
			Window: &assetpb.TimeWindow{
				StartTime: timestamppb.New(testTime),
			},
			Asset: &assetpb.Asset{
				Name:      "//container.googleapis.com/projects/test-project/locations/us-central1/clusters/test-cluster",
				AssetType: caik8s.GKEClusterAssetType,
				Resource: &assetpb.Resource{
					Data: clusterData,
				},
			},
		},
	}

	testCases := []struct {
		name      string
		taskMode  inspectioncore.InspectionTaskModeType
		snapshots []*gcpcommon.CAIAssetSnapshot
		wantCount int
	}{
		{
			name:      "returns empty on DryRun mode",
			taskMode:  inspectioncore.TaskModeDryRun,
			snapshots: []*gcpcommon.CAIAssetSnapshot{snapshot},
			wantCount: 0,
		},
		{
			name:      "converts snapshots to raw logs",
			taskMode:  inspectioncore.TaskModeRun,
			snapshots: []*gcpcommon.CAIAssetSnapshot{snapshot},
			wantCount: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			got, _, err := inspectiontest.RunInspectionTask(ctx, GKEResourceSuite.RawLogTask, tc.taskMode, map[string]any{},
				tasktest.NewTaskDependencyValuePair(caik8s.GKEResourceTaskIDs.Fetcher.Ref(), tc.snapshots),
			)
			if err != nil {
				t.Fatalf("RawLogTask error: %v", err)
			}
			if len(got) != tc.wantCount {
				t.Fatalf("len(got) = %d, want %d", len(got), tc.wantCount)
			}
			if tc.wantCount > 0 {
				if !got[0].Timestamp.Equal(testTime) {
					t.Errorf("got[0].Timestamp = %v, want %v", got[0].Timestamp, testTime)
				}
				name := gcpcommon.ExtractCAIAssetName(got[0].NodeReader)
				if name != snapshot.TemporalAsset.Asset.Name {
					t.Errorf("got asset name = %q, want %q", name, snapshot.TemporalAsset.Asset.Name)
				}
			}
		})
	}
}

func TestGKEResourceSuite_LogIngesterTask(t *testing.T) {
	testTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	generator := id.NewGenerator()

	clusterLog := newLogFromMap(t, generator, testTime, map[string]any{
		"asset": map[string]any{
			"name":      "//container.googleapis.com/projects/test-project/locations/us-central1/clusters/test-cluster",
			"assetType": caik8s.GKEClusterAssetType,
		},
	})

	nodePoolLog := newLogFromMap(t, generator, testTime, map[string]any{
		"asset": map[string]any{
			"name":      "//container.googleapis.com/projects/test-project/locations/us-central1/clusters/test-cluster/nodePools/default-pool",
			"assetType": caik8s.GKENodePoolAssetType,
		},
	})

	unknownLog := newLogFromMap(t, generator, testTime, map[string]any{
		"asset": map[string]any{
			"name": "//unknown",
		},
	})

	testCases := []struct {
		name string
		log  *log.Log
	}{
		{
			name: "ingests cluster log",
			log:  clusterLog,
		},
		{
			name: "ingests nodepool log",
			log:  nodePoolLog,
		},
		{
			name: "ingests unknown log",
			log:  unknownLog,
		},
	}

	ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
	builder := khictx.MustGetValue(ctx, inspectioncore.Builder)

	_, _, err := inspectiontest.RunInspectionTask(ctx, GKEResourceSuite.LogIngesterTask, inspectioncore.TaskModeRun, map[string]any{},
		tasktest.NewTaskDependencyValuePair(caik8s.GKEResourceTaskIDs.RawLog.Ref(), []*log.Log{clusterLog, nodePoolLog, unknownLog}),
	)
	if err != nil {
		t.Fatalf("LogIngesterTask failed: %v", err)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			confirmedID, ok := builder.LogAccumulator.ResolveLogID(tc.log.ID)
			if !ok || confirmedID == 0 {
				t.Errorf("expected log %d to be accumulated by LogIngesterTask with non-zero ID, got confirmedID=%d, ok=%v", tc.log.ID, confirmedID, ok)
			}
		})
	}
}

func TestGKEResourceSuite_LogGrouperTask(t *testing.T) {
	testTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	generator := id.NewGenerator()

	clusterLog := newLogFromMap(t, generator, testTime, map[string]any{
		"asset": map[string]any{
			"name": "//container.googleapis.com/projects/test-project/locations/us-central1/clusters/test-cluster",
		},
	})
	nodePoolLog := newLogFromMap(t, generator, testTime, map[string]any{
		"asset": map[string]any{
			"name": "//container.googleapis.com/projects/test-project/locations/us-central1/clusters/test-cluster/nodePools/default-pool",
		},
	})
	unknownLog := newLogFromMap(t, generator, testTime, map[string]any{})

	testCases := []struct {
		name     string
		rawLogs  []*log.Log
		wantKeys []string
	}{
		{
			name:     "groups cluster and nodepool logs correctly",
			rawLogs:  []*log.Log{clusterLog, nodePoolLog, unknownLog},
			wantKeys: []string{"cluster/test-cluster", "nodepool/test-cluster/default-pool", "unknown"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			got, _, err := inspectiontest.RunInspectionTask(ctx, GKEResourceSuite.LogGrouperTask, inspectioncore.TaskModeRun, map[string]any{},
				tasktest.NewTaskDependencyValuePair(caik8s.GKEResourceTaskIDs.RawLog.Ref(), tc.rawLogs),
			)
			if err != nil {
				t.Fatalf("LogGrouperTask error: %v", err)
			}
			for _, wantKey := range tc.wantKeys {
				if _, ok := got[wantKey]; !ok {
					t.Errorf("missing group key %q in got", wantKey)
				}
			}
			if len(got) != len(tc.wantKeys) {
				t.Errorf("len(got) = %d, want %d", len(got), len(tc.wantKeys))
			}
			logPointerComparer := cmp.Comparer(func(a, b *log.Log) bool {
				return a == b
			})
			if diff := cmp.Diff([]*log.Log{clusterLog}, got["cluster/test-cluster"].Logs, logPointerComparer); diff != "" {
				t.Errorf("group cluster/test-cluster logs mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff([]*log.Log{nodePoolLog}, got["nodepool/test-cluster/default-pool"].Logs, logPointerComparer); diff != "" {
				t.Errorf("group nodepool/test-cluster/default-pool logs mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff([]*log.Log{unknownLog}, got["unknown"].Logs, logPointerComparer); diff != "" {
				t.Errorf("group unknown logs mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGKEResourceSuite_TimelineMapperTask(t *testing.T) {
	queryStartTime := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	assetWindowStartTime := queryStartTime.Add(-1 * time.Hour)
	generator := id.NewGenerator()

	clusterLog := newLogFromMap(t, generator, assetWindowStartTime, map[string]any{
		"window": map[string]any{
			"startTime": assetWindowStartTime.Format(time.RFC3339Nano),
			"endTime":   queryStartTime.Add(1 * time.Hour).Format(time.RFC3339Nano),
		},
		"deleted": false,
		"asset": map[string]any{
			"name":      "//container.googleapis.com/projects/test-project/locations/us-central1-a/clusters/test-cluster",
			"assetType": caik8s.GKEClusterAssetType,
			"resource": map[string]any{
				"data": map[string]any{
					"name": "test-cluster",
				},
			},
		},
	})

	deletedLog := newLogFromMap(t, generator, assetWindowStartTime, map[string]any{
		"window": map[string]any{
			"startTime": assetWindowStartTime.Format(time.RFC3339Nano),
			"endTime":   queryStartTime.Add(1 * time.Hour).Format(time.RFC3339Nano),
		},
		"deleted": true,
		"asset": map[string]any{
			"name":      "//container.googleapis.com/projects/test-project/locations/us-central1-a/clusters/test-cluster/nodePools/deleted-pool",
			"assetType": caik8s.GKENodePoolAssetType,
		},
	})

	ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
	builder := khictx.MustGetValue(ctx, inspectioncore.Builder)

	severityID := uint32(1)
	logTypeID := uint32(2)
	for _, l := range []*log.Log{clusterLog, deletedLog} {
		_ = builder.LogAccumulator.AddLog(&khifilev6.StagingLog{
			Log:       l,
			Summary:   "test",
			Timestamp: l.Timestamp,
			Severity:  &pb.Severity{Id: &severityID},
			LogType:   &pb.LogType{Id: &logTypeID},
		})
	}

	groupMap := inspectiontaskbase.LogGroupMap{
		"cluster/test-cluster":               {Group: "cluster/test-cluster", Logs: []*log.Log{clusterLog}},
		"nodepool/test-cluster/deleted-pool": {Group: "nodepool/test-cluster/deleted-pool", Logs: []*log.Log{deletedLog}},
	}

	_, _, err := inspectiontest.RunInspectionTask(ctx, GKEResourceSuite.TimelineMapperTask, inspectioncore.TaskModeRun, map[string]any{},
		tasktest.NewTaskDependencyValuePair(caik8s.GKEResourceTaskIDs.LogGrouper.Ref(), groupMap),
		tasktest.NewTaskDependencyValuePair(caik8s.GKEResourceTaskIDs.LogIngester.Ref(), struct{}{}),
		tasktest.NewTaskDependencyValuePair(caik8s.GKEResourceTaskIDs.RawLog.Ref(), []*log.Log{clusterLog, deletedLog}),
		tasktest.NewTaskDependencyValuePair(k8scommon.ClusterIdentityTaskID.Ref(), k8scommon.GoogleCloudClusterIdentity{
			ProjectID:   "test-project",
			ClusterName: "test-cluster",
			Location:    "us-central1-a",
		}),
		tasktest.NewTaskDependencyValuePair(gcpcommon.InputStartTimeTaskID.Ref(), queryStartTime),
	)
	if err != nil {
		t.Fatalf("TimelineMapperTask failed: %v", err)
	}

	projectTimeline := gcpcommon.MustGCPProjectTimeline(ctx, "test-project")
	clusterTimeline := gcpcommon.MustGKEClusterTimeline(ctx, projectTimeline, "test-cluster")
	deletedNodePoolTimeline := gcpcommon.MustGKENodePoolTimeline(ctx, clusterTimeline, "deleted-pool")

	testCases := []struct {
		name          string
		path          *khifilev6.TimelinePath
		wantCount     int
		wantStateType *pb.RevisionState
		wantVerbType  *pb.Verb
	}{
		{
			name:          "stages revision for active cluster resource",
			path:          clusterTimeline,
			wantCount:     1,
			wantStateType: caik8s.RevisionStateGKEClusterSnapshotFromCAI,
			wantVerbType:  k8saudit.VerbCreate,
		},
		{
			name:      "skips revision for deleted nodepool resource",
			path:      deletedNodePoolTimeline,
			wantCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var revs []*pb.Revision
			if protoItems := builder.TimelineAccumulator.GetBuilder(tc.path).ToProto(); protoItems != nil {
				revs = protoItems.Revisions
			}
			if len(revs) != tc.wantCount {
				t.Fatalf("len(revs) = %d, want %d", len(revs), tc.wantCount)
			}
			if tc.wantCount > 0 {
				if revs[0].GetStateType() != tc.wantStateType.GetId() {
					t.Errorf("revs[0].GetStateType() = %d, want %d", revs[0].GetStateType(), tc.wantStateType.GetId())
				}
				if revs[0].GetVerbType() != tc.wantVerbType.GetId() {
					t.Errorf("revs[0].GetVerbType() = %d, want %d", revs[0].GetVerbType(), tc.wantVerbType.GetId())
				}
			}
		})
	}
}
