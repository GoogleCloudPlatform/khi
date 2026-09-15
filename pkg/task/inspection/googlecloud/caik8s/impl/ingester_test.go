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
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/id"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/caik8s"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// newLogFromMap builds a log from data. The caller shares one generator across the logs of a test
// so that every log gets a distinct log ID; a fresh generator restarts the counter at zero.
func newLogFromMap(t *testing.T, generator *id.Generator, timestamp time.Time, data map[string]any) *log.Log {
	t.Helper()
	node, err := structured.FromGoValue(data, &structured.AlphabeticalGoMapKeyOrderProvider{})
	if err != nil {
		t.Fatalf("failed to build node from map: %v", err)
	}
	return log.NewLogWithTimestamp(generator, structured.NewNodeReader(node), timestamp)
}

func TestCAIClusterResourceLogIngester_ProcessLog(t *testing.T) {
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
		name      string
		inputLog  *log.Log
		assertLog func(t *testing.T, cs *khifilev6.LogChangeSet)
	}{
		{
			name:     "successfully populates metadata for log with resource identity",
			inputLog: resourceDataLog,
			assertLog: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasTimestamp(testTime).
					HasSeverity(inspectioncore.SeverityInfo).
					HasLogType(caik8s.LogTypeCAIResourceSnapshot).
					HasSummary("CAI resource snapshot: pod/pod-1")
			},
		},
		{
			name:     "successfully populates metadata for log without resource identity",
			inputLog: noResourceDataLog,
			assertLog: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasTimestamp(testTime).
					HasSeverity(inspectioncore.SeverityInfo).
					HasLogType(caik8s.LogTypeCAIResourceSnapshot).
					HasSummary("CAI resource snapshot: /")
			},
		},
	}

	ingester := &caiClusterResourceLogIngester{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cs, err := ingester.ProcessLog(t.Context(), tc.inputLog)
			if err != nil {
				t.Fatalf("ProcessLog() returned error: %v", err)
			}
			if cs == nil {
				t.Fatal("ProcessLog() = nil, want non-nil changeset")
			}
			tc.assertLog(t, cs)
		})
	}
}

func TestRawLogTask(t *testing.T) {
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

	snapshotWithResourceData := &caik8s.ClusterResourceSnapshot{
		StartTime: testTime,
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

	snapshotWithoutResourceData := &caik8s.ClusterResourceSnapshot{
		StartTime: testTime,
		TemporalAsset: &assetpb.TemporalAsset{
			Asset: &assetpb.Asset{
				Name:      "//container.googleapis.com/projects/test/locations/us-central1/clusters/c1/k8s/namespaces/default/pods/pod-asset",
				AssetType: "k8s.io/Pod",
			},
		},
	}

	windowStartTime := testTime.Add(-1 * time.Hour)
	windowEndTime := testTime.Add(1 * time.Hour)

	deletedSnapshotWithWindow := &caik8s.ClusterResourceSnapshot{
		StartTime: testTime,
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

	snapshotWithoutTypeMeta := &caik8s.ClusterResourceSnapshot{
		StartTime: testTime,
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

	snapshotWithEncodedMetadata := &caik8s.ClusterResourceSnapshot{
		StartTime: testTime,
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
		snapshots              []*caik8s.ClusterResourceSnapshot
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
			snapshots: []*caik8s.ClusterResourceSnapshot{snapshotWithResourceData},
			wantCount: 0,
		},
		{
			name:                "converts snapshots to raw logs in run mode",
			taskMode:            inspectioncore.TaskModeRun,
			snapshots:           []*caik8s.ClusterResourceSnapshot{snapshotWithResourceData},
			wantCount:           1,
			wantIdentity:        podSampleIdentity,
			wantHasResourceBody: true,
		},
		{
			name:                   "restores missing apiVersion and kind into resource body",
			taskMode:               inspectioncore.TaskModeRun,
			snapshots:              []*caik8s.ClusterResourceSnapshot{snapshotWithoutTypeMeta},
			wantCount:              1,
			wantIdentity:           podRestoredIdentity,
			wantHasResourceBody:    true,
			wantManifestAPIVersion: "v1",
			wantManifestKind:       "Pod",
		},
		{
			name:                "decodes managedFields base64 fieldsV1 and strips empty default fields in metadata",
			taskMode:            inspectioncore.TaskModeRun,
			snapshots:           []*caik8s.ClusterResourceSnapshot{snapshotWithEncodedMetadata},
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
			snapshots:           []*caik8s.ClusterResourceSnapshot{snapshotWithoutResourceData},
			wantCount:           1,
			wantIdentity:        podAssetIdentity,
			wantHasResourceBody: false,
		},
		{
			name:                "serializes the validity window and tombstone flag readable by the log extractors",
			taskMode:            inspectioncore.TaskModeRun,
			snapshots:           []*caik8s.ClusterResourceSnapshot{deletedSnapshotWithWindow},
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
			got, _, err := inspectiontest.RunInspectionTask(ctx, RawLogTask, tc.taskMode, map[string]any{},
				tasktest.NewTaskDependencyValuePair(caik8s.ClusterResourceFetcherTaskID.Ref(), tc.snapshots),
			)
			if err != nil {
				t.Fatalf("RawLogTask failed: %v", err)
			}
			if len(got) != tc.wantCount {
				t.Errorf("len(got) = %d, want %d", len(got), tc.wantCount)
			}
			if tc.wantCount > 0 {
				if !got[0].Timestamp.Equal(testTime) {
					t.Errorf("got[0].Timestamp = %v, want %v", got[0].Timestamp, testTime)
				}
				identity := extractResourceIdentityFromLog(got[0].NodeReader)
				if diff := cmp.Diff(tc.wantIdentity, identity); diff != "" {
					t.Errorf("extractResourceIdentityFromLog() mismatch (-want +got):\n%s", diff)
				}
				gotWindowStartTime, gotWindowEndTime, gotDeleted := extractTimeWindow(got[0].NodeReader)
				if !gotWindowStartTime.Equal(tc.wantWindowStartTime) {
					t.Errorf("extractTimeWindow() startTime = %v, want %v", gotWindowStartTime, tc.wantWindowStartTime)
				}
				if !gotWindowEndTime.Equal(tc.wantWindowEndTime) {
					t.Errorf("extractTimeWindow() endTime = %v, want %v", gotWindowEndTime, tc.wantWindowEndTime)
				}
				if gotDeleted != tc.wantDeleted {
					t.Errorf("extractTimeWindow() isDeleted = %v, want %v", gotDeleted, tc.wantDeleted)
				}
				gotHasResourceBody := extractResourceBody(got[0].NodeReader) != nil
				if gotHasResourceBody != tc.wantHasResourceBody {
					t.Errorf("extractResourceBody() != nil = %v, want %v", gotHasResourceBody, tc.wantHasResourceBody)
				}
				if tc.wantManifestAPIVersion != "" {
					bodyNode := extractResourceBody(got[0].NodeReader)
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
					bodyNode := extractResourceBody(got[0].NodeReader)
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
					bodyNode := extractResourceBody(got[0].NodeReader)
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

func TestLogGrouperTask(t *testing.T) {
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
			groups, _, err := inspectiontest.RunInspectionTask(ctx, LogGrouperTask, inspectioncore.TaskModeRun, map[string]any{},
				tasktest.NewTaskDependencyValuePair(caik8s.RawLogTaskID.Ref(), tc.inputLogs),
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
