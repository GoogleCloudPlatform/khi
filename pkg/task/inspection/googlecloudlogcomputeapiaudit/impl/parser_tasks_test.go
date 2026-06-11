// Copyright 2024 Google LLC
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

package googlecloudlogcomputeapiaudit_impl

import (
	"context"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	core_contract "github.com/GoogleCloudPlatform/khi/pkg/task/core/contract"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogcomputeapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcomputeapiaudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"

	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
)

func TestLogIngester_ProcessLog(t *testing.T) {
	testTime := time.Date(2025, time.January, 1, 1, 1, 1, 1, time.UTC)
	testCases := []struct {
		name   string
		input  *log.Log
		assert func(t *testing.T, cs *khifilev6.LogChangeSet)
	}{
		{
			name: "ingest compute API audit log - start",
			input: log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{
					Timestamp: testTime,
				},
				&inspectioncore_contract.DefaultSeverityFieldSet{
					Severity: inspectioncore_contract.SeverityInfo,
				},
				&googlecloudcommon_contract.GCPAuditLogFieldSet{
					MethodName:     "compute.instances.insert",
					OperationFirst: true,
					OperationLast:  false,
				},
			),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary("compute.instances.insert Started").
					HasLogType(googlecloudlogcomputeapiaudit_contract.LogTypeComputeApi).
					HasTimestamp(testTime)
			},
		},
		{
			name: "ingest compute API audit log - finish",
			input: log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{
					Timestamp: testTime,
				},
				&inspectioncore_contract.DefaultSeverityFieldSet{
					Severity: inspectioncore_contract.SeverityError,
				},
				&googlecloudcommon_contract.GCPAuditLogFieldSet{
					MethodName:     "compute.instances.insert",
					OperationFirst: false,
					OperationLast:  true,
				},
			),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary("compute.instances.insert Finished").
					HasLogType(googlecloudlogcomputeapiaudit_contract.LogTypeComputeApi).
					HasTimestamp(testTime)
			},
		},
	}

	ingester := &gcpComputeAuditLogLogIngester{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cs, err := ingester.ProcessLog(t.Context(), tc.input)
			if err != nil {
				t.Fatalf("ProcessLog() returned unexpected error: %v", err)
			}
			tc.assert(t, cs)
		})
	}
}

func TestLogToTimelineMapper_ProcessLogByGroup(t *testing.T) {
	builder := khifilev6.NewBuilder()

	// Setup context with task result mapping containing ClusterIdentity
	taskResults := typedmap.NewTypedMap()
	typedmap.Set(taskResults, typedmap.NewTypedKey[googlecloudk8scommon_contract.GoogleCloudClusterIdentity](googlecloudlogcomputeapiaudit_contract.ClusterIdentityTaskID.Ref().ReferenceIDString()), googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
		ClusterName: "test-cluster",
	})

	baseCtx := khictx.WithValue(t.Context(), core_contract.TaskResultMapContextKey, taskResults)
	ctx := khictx.WithValue(baseCtx, inspectioncore_contract.Builder, builder)

	// Independently build the expected paths segment-by-segment
	clusterTimeline := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, "test-cluster")
	apiTimeline := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterTimeline, "core/v1")
	kindTimeline := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiTimeline, "node")
	nsTimeline := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindTimeline, "cluster-scope")

	wantNodeAbcPath := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, nsTimeline, "abc")
	wantOp1Path := builder.TimelineAccumulator.GetPath(wantNodeAbcPath, khifilev6.PathSegment{
		Name: "insert-op-1",
		Type: googlecloudcommon_contract.TimelineTypeOperation,
	})

	wantNodeDefPath := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, nsTimeline, "def")

	wantNodeGhiPath := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, nsTimeline, "ghi")
	wantOp3Path := builder.TimelineAccumulator.GetPath(wantNodeGhiPath, khifilev6.PathSegment{
		Name: "delete-op-3",
		Type: googlecloudcommon_contract.TimelineTypeOperation,
	})

	testTime := time.Date(2025, time.January, 1, 1, 1, 1, 1, time.UTC)
	testCommonFieldSet := &log.CommonFieldSet{
		Timestamp: testTime,
	}

	testCases := []struct {
		name     string
		inputLog *log.Log
		assert   func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet)
	}{
		{
			name: "operation started",
			inputLog: log.NewLogWithFieldSetsForTest(testCommonFieldSet, &googlecloudcommon_contract.GCPAuditLogFieldSet{
				OperationID:    "op-1",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "compute.instances.insert",
				ResourceName:   "projects/123/resources/abc",
				PrincipalEmail: "foobar@qux.test",
			}),
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantOp1Path, &khifilev6.StagingRevision{
						ChangedTime: testTime,
						StateType:   googlecloudcommon_contract.RevisionStateOperationStarted,
						VerbType:    googlecloudcommon_contract.VerbOperationStart,
						Principal:   "foobar@qux.test",
					})
			},
		},
		{
			name: "operation finished",
			inputLog: log.NewLogWithFieldSetsForTest(testCommonFieldSet, &googlecloudcommon_contract.GCPAuditLogFieldSet{
				OperationID:    "op-1",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "compute.instances.insert",
				ResourceName:   "projects/123/resources/abc",
				PrincipalEmail: "foobar@qux.test",
			}),
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantOp1Path, &khifilev6.StagingRevision{
						ChangedTime: testTime,
						StateType:   googlecloudcommon_contract.RevisionStateOperationFinished,
						VerbType:    googlecloudcommon_contract.VerbOperationFinish,
						Principal:   "foobar@qux.test",
					})
			},
		},
		{
			name: "immediate operation",
			inputLog: log.NewLogWithFieldSetsForTest(testCommonFieldSet, &googlecloudcommon_contract.GCPAuditLogFieldSet{
				OperationID:    "op-2",
				OperationFirst: true,
				OperationLast:  true,
				MethodName:     "compute.instances.delete",
				ResourceName:   "projects/123/resources/def",
				PrincipalEmail: "foobar@qux.test",
			}),
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantNodeDefPath)
			},
		},
		{
			name: "deletion operation started",
			inputLog: log.NewLogWithFieldSetsForTest(testCommonFieldSet, &googlecloudcommon_contract.GCPAuditLogFieldSet{
				OperationID:    "op-3",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "compute.instances.delete",
				ResourceName:   "projects/123/resources/ghi",
				PrincipalEmail: "foobar@qux.test",
			}),
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantOp3Path, &khifilev6.StagingRevision{
						ChangedTime: testTime,
						StateType:   googlecloudcommon_contract.RevisionStateOperationStarted,
						VerbType:    googlecloudcommon_contract.VerbOperationStart,
						Principal:   "foobar@qux.test",
					})
			},
		},
		{
			name: "deletion operation finished",
			inputLog: log.NewLogWithFieldSetsForTest(testCommonFieldSet, &googlecloudcommon_contract.GCPAuditLogFieldSet{
				OperationID:    "op-3",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "compute.instances.delete",
				ResourceName:   "projects/123/resources/ghi",
				PrincipalEmail: "foobar@qux.test",
			}),
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantOp3Path, &khifilev6.StagingRevision{
						ChangedTime: testTime,
						StateType:   googlecloudcommon_contract.RevisionStateOperationFinished,
						VerbType:    googlecloudcommon_contract.VerbOperationFinish,
						Principal:   "foobar@qux.test",
					})
			},
		},
	}

	mapper := &gcpComputeAuditLogLogToTimelineMapperSetting{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testCtx := khictx.WithValue(ctx, inspectioncore_contract.Builder, builder)
			cs, _, err := mapper.ProcessLogByGroup(testCtx, tc.inputLog, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLogByGroup() returned unexpected error: %v", err)
			}

			tc.assert(t, testCtx, cs)
		})
	}
}

func TestGetInstanceNameFromResourceName(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  string
	}{
		{
			desc:  "standard resource name",
			input: "projects/123/zones/us-central1-a/instances/my-instance",
			want:  "my-instance",
		},
		{
			desc:  "empty resource name",
			input: "",
			want:  "",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			got := getInstanceNameFromResourceName(tc.input)
			if got != tc.want {
				t.Errorf("getInstanceNameFromResourceName(%q) got %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
