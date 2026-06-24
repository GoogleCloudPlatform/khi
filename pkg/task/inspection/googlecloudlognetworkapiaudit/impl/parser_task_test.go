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

package googlecloudlognetworkapiaudit_impl

import (
	"context"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourceinfo/resourcelease"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	core_contract "github.com/GoogleCloudPlatform/khi/pkg/task/core/contract"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlognetworkapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlognetworkapiaudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/google/go-cmp/cmp"
)

func testReaderFromYAML(t *testing.T, yaml string) *structured.NodeReader {
	t.Helper()
	node, err := structured.FromYAML(yaml)
	if err != nil {
		t.Fatalf("failed to parse yaml: %v", err)
	}
	return structured.NewNodeReader(node)
}

var nodeTransformer = cmp.Transformer("NodeToString", func(n structured.Node) string {
	if n == nil {
		return ""
	}
	serializer := &structured.JSONNodeSerializer{}
	bytes, err := serializer.Serialize(n)
	if err != nil {
		return ""
	}
	return string(bytes)
})

func TestNetworkAPILogIngester_ProcessLog(t *testing.T) {
	testTime := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	testCases := []struct {
		name   string
		input  *log.Log
		assert func(t *testing.T, cs *khifilev6.LogChangeSet)
	}{
		{
			name: "successful audit log ingestion starting",
			input: log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{
					Timestamp: testTime,
				},
				&inspectioncore_contract.DefaultSeverityFieldSet{
					Severity: inspectioncore_contract.SeverityInfo,
				},
				&googlecloudcommon_contract.GCPAuditLogFieldSet{
					MethodName:     "v1.Compute.NetworkEndpointGroups.attachNetworkEndpoints",
					OperationID:    "op-1",
					OperationFirst: true,
					OperationLast:  false,
				},
			),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary("Start: v1.Compute.NetworkEndpointGroups.attachNetworkEndpoints").
					HasTimestamp(testTime).
					HasLogType(googlecloudlognetworkapiaudit_contract.LogTypeNetworkAPI)
			},
		},
		{
			name: "successful audit log ingestion ending",
			input: log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{
					Timestamp: testTime,
				},
				&inspectioncore_contract.DefaultSeverityFieldSet{
					Severity: inspectioncore_contract.SeverityInfo,
				},
				&googlecloudcommon_contract.GCPAuditLogFieldSet{
					MethodName:     "v1.Compute.NetworkEndpointGroups.attachNetworkEndpoints",
					OperationID:    "op-1",
					OperationFirst: false,
					OperationLast:  true,
				},
			),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasSummary("Succeeded: v1.Compute.NetworkEndpointGroups.attachNetworkEndpoints").
					HasTimestamp(testTime).
					HasLogType(googlecloudlognetworkapiaudit_contract.LogTypeNetworkAPI)
			},
		},
	}

	ingester := googlecloudcommon_contract.NewGCPOperationLogIngester(googlecloudlognetworkapiaudit_contract.FieldSetReaderTaskID.Ref(), googlecloudlognetworkapiaudit_contract.LogTypeNetworkAPI)
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

func TestNetworkAPITimelineMapper_ProcessLogByGroup(t *testing.T) {
	testTime := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)
	builder := khifilev6.NewBuilder()

	// Define expected timeline paths.
	wantNEGPath := googlecloudlognetworkapiaudit_contract.MustNEGTimeline(khictx.WithValue(context.Background(), inspectioncore_contract.Builder, builder), "cluster", "test-ns", "test-neg")

	testCases := []struct {
		name          string
		inputLog      *log.Log
		prevGroupData *perNEGHistoryModificationStatus
		wantGroupData *perNEGHistoryModificationStatus
		setupContext  func(ctx context.Context) context.Context
		assert        func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet)
	}{
		{
			name: "operation started revision is correctly created",
			inputLog: log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{
					Timestamp: testTime,
				},
				&googlecloudcommon_contract.GCPAuditLogFieldSet{
					MethodName:     "v1.Compute.NetworkEndpointGroups.attachNetworkEndpoints",
					ResourceName:   "projects/test-project/zones/us-central1-a/networkEndpointGroups/test-neg",
					OperationID:    "op-1",
					OperationFirst: true,
					OperationLast:  false,
					PrincipalEmail: "test-user@google.com",
					Request:        testReaderFromYAML(t, "networkEndpoints:\n- instance: test-node\n  ipAddress: 10.0.0.1\n  port: \"80\""),
				},
			),
			prevGroupData: nil,
			wantGroupData: &perNEGHistoryModificationStatus{
				LastNegAttachRequest: &negAttachOrDetachRequest{
					NetworkEndpoints: []*negAttachOrDetachRequestEndpoint{
						{
							Instance:  "test-node",
							IpAddress: "10.0.0.1",
							Port:      "80",
						},
					},
				},
			},
			setupContext: func(ctx context.Context) context.Context {
				negs := googlecloudk8scommon_contract.NEGNameToResourceIdentityMap{
					"test-neg": {
						Namespace: "test-ns",
						Name:      "test-neg",
					},
				}
				return tasktest.WithTaskResult(ctx, googlecloudk8scommon_contract.NEGNamesInventoryTaskID.Ref(), negs)
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				wantOpPath := googlecloudlognetworkapiaudit_contract.MustNEGOperationTimeline(ctx, wantNEGPath, "v1.Compute.NetworkEndpointGroups.attachNetworkEndpoints", "op-1")
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantOpPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						Principal:    "test-user@google.com",
						VerbType:     googlecloudcommon_contract.VerbOperationStart,
						StateType:    googlecloudcommon_contract.RevisionStateOperationStarted,
						ResourceBody: testReaderFromYAML(t, "networkEndpoints:\n- instance: test-node\n  ipAddress: 10.0.0.1\n  port: \"80\"").Node,
					}, nodeTransformer)
			},
		},
		{
			name: "operation finished revision is correctly created",
			inputLog: log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{
					Timestamp: testTime,
				},
				&googlecloudcommon_contract.GCPAuditLogFieldSet{
					MethodName:     "v1.Compute.NetworkEndpointGroups.attachNetworkEndpoints",
					ResourceName:   "projects/test-project/zones/us-central1-a/networkEndpointGroups/test-neg",
					OperationID:    "op-1",
					OperationFirst: false,
					OperationLast:  true,
					PrincipalEmail: "test-user@google.com",
				},
			),
			prevGroupData: &perNEGHistoryModificationStatus{
				OperationTracker: googlecloudcommon_contract.NewGCPOperationTracker(),
				LastNegAttachRequest: &negAttachOrDetachRequest{
					NetworkEndpoints: []*negAttachOrDetachRequestEndpoint{
						{
							Instance:  "test-node",
							IpAddress: "10.0.0.1",
							Port:      "80",
						},
					},
				},
			},
			wantGroupData: &perNEGHistoryModificationStatus{
				LastNegAttachRequest: nil,
			},
			setupContext: func(ctx context.Context) context.Context {
				negs := googlecloudk8scommon_contract.NEGNameToResourceIdentityMap{
					"test-neg": {
						Namespace: "test-ns",
						Name:      "test-neg",
					},
				}
				ipLeases := resourcelease.NewResourceLeaseHistory[*commonlogk8saudit_contract.ResourceIdentity]()
				ipLeases.TouchResourceLease("10.0.0.1", testTime, &commonlogk8saudit_contract.ResourceIdentity{
					Kind:      "pod",
					Namespace: "test-ns",
					Name:      "test-pod",
				})
				negToBS := googlecloudk8scommon_contract.NEGToBackendServiceMap{
					"test-neg": "test-bs",
				}
				ctx = tasktest.WithTaskResult(ctx, googlecloudk8scommon_contract.NEGNamesInventoryTaskID.Ref(), negs)
				ctx = tasktest.WithTaskResult(ctx, commonlogk8saudit_contract.IPLeaseHistoryInventoryTaskID.Ref(), ipLeases)
				ctx = tasktest.WithTaskResult(ctx, googlecloudk8scommon_contract.NEGToBackendServiceInventoryTaskID.Ref(), negToBS)
				return ctx
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				wantOpPath := googlecloudlognetworkapiaudit_contract.MustNEGOperationTimeline(ctx, wantNEGPath, "v1.Compute.NetworkEndpointGroups.attachNetworkEndpoints", "op-1")
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantOpPath, &khifilev6.StagingRevision{
						ChangedTime: testTime,
						Principal:   "test-user@google.com",
						VerbType:    googlecloudcommon_contract.VerbOperationFinish,
						StateType:   googlecloudcommon_contract.RevisionStateOperationSucceed,
					}, nodeTransformer)

				clusterPath := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, "cluster")
				apiPath := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterPath, "core/v1")
				kindPath := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiPath, "pod")
				nsPath := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindPath, "test-ns")
				podPath := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, nsPath, "test-pod")
				wantPodNEGPath := googlecloudlognetworkapiaudit_contract.MustNEGUnderResourceTimeline(ctx, podPath, "test-neg")

				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantPodNEGPath, &khifilev6.StagingRevision{
						ChangedTime: testTime,
						Principal:   "test-user@google.com",
						VerbType:    commonlogk8saudit_contract.VerbReady,
						StateType:   commonlogk8saudit_contract.RevisionStateConditionTrue,
					}, nodeTransformer)

				bsPath := googlecloudlognetworkapiaudit_contract.MustGCPResourceTimeline(ctx, "test-project", "backendServices", "test-bs")
				wantBSNEGPath := googlecloudlognetworkapiaudit_contract.MustNEGUnderResourceTimeline(ctx, bsPath, "test-pod")

				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantBSNEGPath, &khifilev6.StagingRevision{
						ChangedTime: testTime,
						Principal:   "test-user@google.com",
						VerbType:    commonlogk8saudit_contract.VerbReady,
						StateType:   commonlogk8saudit_contract.RevisionStateConditionTrue,
					}, nodeTransformer)
			},
		},
	}

	mapper := &networkAPITimelineMapper{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := khictx.WithValue(t.Context(), core_contract.TaskImplementationIDContextKey, googlecloudlognetworkapiaudit_contract.LogToTimelineMapperTaskID.(taskid.UntypedTaskImplementationID))
			ctx = khictx.WithValue(ctx, inspectioncore_contract.Builder, builder)

			// Provide default empty inventories.
			clusterIdentity := googlecloudk8scommon_contract.GoogleCloudClusterIdentity{
				ClusterName: "cluster",
				ProjectID:   "test-project",
			}
			negs := googlecloudk8scommon_contract.NEGNameToResourceIdentityMap{}
			ipLeases := resourcelease.NewResourceLeaseHistory[*commonlogk8saudit_contract.ResourceIdentity]()
			negToBS := googlecloudk8scommon_contract.NEGToBackendServiceMap{}

			ctx = tasktest.WithTaskResult(ctx, googlecloudk8scommon_contract.ClusterIdentityTaskID.Ref(), clusterIdentity)
			ctx = tasktest.WithTaskResult(ctx, googlecloudk8scommon_contract.NEGNamesInventoryTaskID.Ref(), negs)
			ctx = tasktest.WithTaskResult(ctx, commonlogk8saudit_contract.IPLeaseHistoryInventoryTaskID.Ref(), ipLeases)
			ctx = tasktest.WithTaskResult(ctx, googlecloudk8scommon_contract.NEGToBackendServiceInventoryTaskID.Ref(), negToBS)
			if tc.setupContext != nil {
				ctx = tc.setupContext(ctx)
			}

			cs, gotGroupData, err := mapper.ProcessLogByGroup(ctx, tc.inputLog, tc.prevGroupData)
			if err != nil {
				t.Fatalf("ProcessLogByGroup() returned unexpected error: %v", err)
			}

			if tc.assert != nil {
				tc.assert(t, ctx, cs)
			}

			// Wait, assert the returned group data.
			if tc.wantGroupData != nil {
				if tc.wantGroupData.LastNegAttachRequest == nil {
					if gotGroupData.LastNegAttachRequest != nil {
						t.Errorf("want LastNegAttachRequest to be nil, but got %v", gotGroupData.LastNegAttachRequest)
					}
				} else {
					if gotGroupData.LastNegAttachRequest == nil {
						t.Errorf("want LastNegAttachRequest to be %v, but got nil", tc.wantGroupData.LastNegAttachRequest)
					} else {
						// check network endpoints
						if len(gotGroupData.LastNegAttachRequest.NetworkEndpoints) != len(tc.wantGroupData.LastNegAttachRequest.NetworkEndpoints) {
							t.Fatalf("network endpoints length mismatch: want %d, got %d", len(tc.wantGroupData.LastNegAttachRequest.NetworkEndpoints), len(gotGroupData.LastNegAttachRequest.NetworkEndpoints))
						}
						for i, wantEndpoint := range tc.wantGroupData.LastNegAttachRequest.NetworkEndpoints {
							gotEndpoint := gotGroupData.LastNegAttachRequest.NetworkEndpoints[i]
							if gotEndpoint.Instance != wantEndpoint.Instance || gotEndpoint.IpAddress != wantEndpoint.IpAddress || gotEndpoint.Port != wantEndpoint.Port {
								t.Errorf("endpoint mismatch: want %+v, got %+v", wantEndpoint, gotEndpoint)
							}
						}
					}
				}
			}
		})
	}
}
