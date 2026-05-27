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

package googlecloudlogmulticloudapiaudit_impl

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogmulticloudapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogmulticloudapiaudit/contract"
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

func TestLogToTimelineMapperTask(t *testing.T) {
	// 1. Initialize the Builder first.
	builder := khifilev6.NewBuilder()

	testTime := time.Date(2025, time.January, 1, 1, 1, 1, 1, time.UTC)
	testCommonFieldSet := &log.CommonFieldSet{
		Timestamp: testTime,
	}

	// Custom comparer for structured.Node interface.
	nodeComparer := cmp.Comparer(func(x, y structured.Node) bool {
		if x == nil && y == nil {
			return true
		}
		if x == nil || y == nil {
			return false
		}
		serializer := &structured.YAMLNodeSerializer{}
		xBytes, err1 := serializer.Serialize(x)
		yBytes, err2 := serializer.Serialize(y)
		if err1 != nil || err2 != nil {
			return false
		}
		return string(xBytes) == string(yBytes)
	})

	// Resolve comparative path instances independently using low-level accumulator.
	wantProjPath := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
		Name: "projects/123456",
		Type: googlecloudlogmulticloudapiaudit_contract.TimelineTypeProject,
	})
	wantClusterPath := builder.TimelineAccumulator.GetPath(wantProjPath, khifilev6.PathSegment{
		Name: "test-cluster",
		Type: googlecloudlogmulticloudapiaudit_contract.TimelineTypeMultiCloudCluster,
	})
	wantNodepoolPath := builder.TimelineAccumulator.GetPath(wantClusterPath, khifilev6.PathSegment{
		Name: "test-nodepool",
		Type: googlecloudlogmulticloudapiaudit_contract.TimelineTypeMultiCloudNodepool,
	})

	wantOp1ClusterPath := builder.TimelineAccumulator.GetPath(wantClusterPath, khifilev6.PathSegment{
		Name: "CreateCluster-op-1",
		Type: inspectioncore_contract.TimelineTypeSubresource,
	})
	wantOp1AzureClusterPath := builder.TimelineAccumulator.GetPath(wantClusterPath, khifilev6.PathSegment{
		Name: "CreateCluster-op-1",
		Type: inspectioncore_contract.TimelineTypeSubresource,
	})
	wantOp2NodepoolPath := builder.TimelineAccumulator.GetPath(wantNodepoolPath, khifilev6.PathSegment{
		Name: "CreateNodePool-op-2",
		Type: inspectioncore_contract.TimelineTypeSubresource,
	})
	wantOp2AzureNodepoolPath := builder.TimelineAccumulator.GetPath(wantNodepoolPath, khifilev6.PathSegment{
		Name: "CreateNodePool-op-2",
		Type: inspectioncore_contract.TimelineTypeSubresource,
	})
	wantOp2DeleteNodepoolPath := builder.TimelineAccumulator.GetPath(wantNodepoolPath, khifilev6.PathSegment{
		Name: "DeleteNodePool-op-2",
		Type: inspectioncore_contract.TimelineTypeSubresource,
	})
	wantOp2UnknownNodepoolPath := builder.TimelineAccumulator.GetPath(wantNodepoolPath, khifilev6.PathSegment{
		Name: "UnknownLongRunningOperation-op-2",
		Type: inspectioncore_contract.TimelineTypeSubresource,
	})

	testCases := []struct {
		desc          string
		inputResource googlecloudlogmulticloudapiaudit_contract.MulticloudAPIAuditResourceFieldSet
		inputAudit    googlecloudcommon_contract.GCPAuditLogFieldSet
		assert        func(t *testing.T, cs *khifilev6.TimelineChangeSet)
	}{
		{
			desc: "cluster create started",
			inputResource: googlecloudlogmulticloudapiaudit_contract.MulticloudAPIAuditResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "",
				ClusterType:  googlecloudlogmulticloudapiaudit_contract.ClusterTypeAWS,
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				OperationID:    "op-1",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "google.cloud.gkemulticloud.v1.AwsClusters.CreateAwsCluster",
				PrincipalEmail: "foobar@qux.test",
				Request: testReaderFromYAML(t, `cluster:
  initialNodeCount: 1
  name: test-cluster`),
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantClusterPath, &khifilev6.StagingRevision{
						ChangedTime: testTime,
						ResourceBody: testReaderFromYAML(t, `initialNodeCount: 1
name: test-cluster`).Node,
						Principal: "foobar@qux.test",
						VerbType:  commonlogk8saudit_contract.VerbCreate,
						StateType: googlecloudlogmulticloudapiaudit_contract.RevisionStateProvisioning,
					}, nodeComparer).
					HasRevision(wantOp1ClusterPath, &khifilev6.StagingRevision{
						ChangedTime: testTime,
						ResourceBody: testReaderFromYAML(t, `cluster:
  initialNodeCount: 1
  name: test-cluster`).Node,
						Principal: "foobar@qux.test",
						VerbType:  googlecloudcommon_contract.VerbOperationStart,
						StateType: googlecloudcommon_contract.RevisionStateOperationStarted,
					}, nodeComparer)
			},
		},
		{
			desc: "cluster create finished",
			inputResource: googlecloudlogmulticloudapiaudit_contract.MulticloudAPIAuditResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "",
				ClusterType:  googlecloudlogmulticloudapiaudit_contract.ClusterTypeAzure,
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				OperationID:    "op-1",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.cloud.gkemulticloud.v1.AzureClusters.CreateAzureCluster",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantClusterPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: nil,
						Principal:    "foobar@qux.test",
						VerbType:     commonlogk8saudit_contract.VerbCreate,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceExisting,
					}, nodeComparer).
					HasRevision(wantOp1AzureClusterPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: nil,
						Principal:    "foobar@qux.test",
						VerbType:     googlecloudcommon_contract.VerbOperationFinish,
						StateType:    googlecloudcommon_contract.RevisionStateOperationFinished,
					}, nodeComparer)
			},
		},
		{
			desc: "nodepool create started",
			inputResource: googlecloudlogmulticloudapiaudit_contract.MulticloudAPIAuditResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
				ClusterType:  googlecloudlogmulticloudapiaudit_contract.ClusterTypeAWS,
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				OperationID:    "op-2",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "google.cloud.gkemulticloud.v1.AwsClusters.CreateAwsNodePool",
				PrincipalEmail: "foobar@qux.test",
				Request: testReaderFromYAML(t, `nodePool:
  initialNodeCount: 1
  name: test-nodepool`),
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantNodepoolPath, &khifilev6.StagingRevision{
						ChangedTime: testTime,
						ResourceBody: testReaderFromYAML(t, `initialNodeCount: 1
name: test-nodepool`).Node,
						Principal: "foobar@qux.test",
						VerbType:  commonlogk8saudit_contract.VerbCreate,
						StateType: googlecloudlogmulticloudapiaudit_contract.RevisionStateProvisioning,
					}, nodeComparer).
					HasRevision(wantOp2NodepoolPath, &khifilev6.StagingRevision{
						ChangedTime: testTime,
						ResourceBody: testReaderFromYAML(t, `nodePool:
  initialNodeCount: 1
  name: test-nodepool`).Node,
						Principal: "foobar@qux.test",
						VerbType:  googlecloudcommon_contract.VerbOperationStart,
						StateType: googlecloudcommon_contract.RevisionStateOperationStarted,
					}, nodeComparer)
			},
		},
		{
			desc: "nodepool create finished",
			inputResource: googlecloudlogmulticloudapiaudit_contract.MulticloudAPIAuditResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
				ClusterType:  googlecloudlogmulticloudapiaudit_contract.ClusterTypeAzure,
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				OperationID:    "op-2",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.cloud.gkemulticloud.v1.AzureClusters.CreateAzureNodePool",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantNodepoolPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: nil,
						Principal:    "foobar@qux.test",
						VerbType:     commonlogk8saudit_contract.VerbCreate,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceExisting,
					}, nodeComparer).
					HasRevision(wantOp2AzureNodepoolPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: nil,
						Principal:    "foobar@qux.test",
						VerbType:     googlecloudcommon_contract.VerbOperationFinish,
						StateType:    googlecloudcommon_contract.RevisionStateOperationFinished,
					}, nodeComparer)
			},
		},
		{
			desc: "nodepool deletion finished",
			inputResource: googlecloudlogmulticloudapiaudit_contract.MulticloudAPIAuditResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
				ClusterType:  googlecloudlogmulticloudapiaudit_contract.ClusterTypeAWS,
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				OperationID:    "op-2",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.cloud.gkemulticloud.v1.AwsClusters.DeleteAwsNodePool",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantNodepoolPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: nil,
						Principal:    "foobar@qux.test",
						VerbType:     commonlogk8saudit_contract.VerbDelete,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceIsDeleted,
					}, nodeComparer).
					HasRevision(wantOp2DeleteNodepoolPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: nil,
						Principal:    "foobar@qux.test",
						VerbType:     googlecloudcommon_contract.VerbOperationFinish,
						StateType:    googlecloudcommon_contract.RevisionStateOperationFinished,
					}, nodeComparer)
			},
		},
		{
			desc: "immediate action",
			inputResource: googlecloudlogmulticloudapiaudit_contract.MulticloudAPIAuditResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
				ClusterType:  googlecloudlogmulticloudapiaudit_contract.ClusterTypeAzure,
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				OperationID:    "op-2",
				OperationFirst: true,
				OperationLast:  true,
				MethodName:     "google.cloud.gkemulticloud.v1.AzureClusters.UpdateAzureCluster",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantNodepoolPath)
			},
		},
		{
			desc: "long running action for unknown cluster type",
			inputResource: googlecloudlogmulticloudapiaudit_contract.MulticloudAPIAuditResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
				ClusterType:  googlecloudlogmulticloudapiaudit_contract.ClusterTypeUnknown,
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				OperationID:    "op-2",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "google.cloud.gkemulticloud.v1.FooClusters.UnknownLongRunningOperation",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantOp2UnknownNodepoolPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: nil,
						Principal:    "foobar@qux.test",
						VerbType:     googlecloudcommon_contract.VerbOperationStart,
						StateType:    googlecloudcommon_contract.RevisionStateOperationStarted,
					}, nodeComparer)
			},
		},
	}

	mapper := &multicloudAuditLogLogToTimelineMapperSetting{}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l := log.NewLogWithFieldSetsForTest(testCommonFieldSet, &tc.inputAudit, &tc.inputResource)
			l.NodeReader = testReaderFromYAML(t, "protoPayload:\n  resourceName: projects/123456/locations/asia-southeast1/awsClusters/test-cluster/awsNodePools/test-nodepool")

			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)
			cs, _, err := mapper.ProcessLogByGroup(ctx, l, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLogByGroup() returned unexpected error: %v", err)
			}

			tc.assert(t, cs)
		})
	}
}
