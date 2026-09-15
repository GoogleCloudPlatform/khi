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

package multicloudapiaudit_impl

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/model/id"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/multicloudapiaudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
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
	builder := khifilev6.NewTestBuilder(id.NewGenerator())

	testTime := time.Date(2025, time.January, 1, 1, 1, 1, 1, time.UTC)

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
		Name: "project-foo",
		Type: gcpcommon.TimelineTypeGCPProject,
	})
	wantClusterPath := builder.TimelineAccumulator.GetPath(wantProjPath, khifilev6.PathSegment{
		Name: "test-cluster",
		Type: multicloudapiaudit.TimelineTypeMultiCloudCluster,
	})
	wantNodepoolPath := builder.TimelineAccumulator.GetPath(wantClusterPath, khifilev6.PathSegment{
		Name: "test-nodepool",
		Type: multicloudapiaudit.TimelineTypeMultiCloudNodepool,
	})

	wantOp1ClusterPath := builder.TimelineAccumulator.GetPath(wantClusterPath, khifilev6.PathSegment{
		Name: "CreateCluster-op-1",
		Type: inspectioncore.TimelineTypeSubresource,
	})
	wantOp1AzureClusterPath := builder.TimelineAccumulator.GetPath(wantClusterPath, khifilev6.PathSegment{
		Name: "CreateCluster-op-1",
		Type: inspectioncore.TimelineTypeSubresource,
	})
	wantOp2NodepoolPath := builder.TimelineAccumulator.GetPath(wantNodepoolPath, khifilev6.PathSegment{
		Name: "CreateNodePool-op-2",
		Type: inspectioncore.TimelineTypeSubresource,
	})
	wantOp2AzureNodepoolPath := builder.TimelineAccumulator.GetPath(wantNodepoolPath, khifilev6.PathSegment{
		Name: "CreateNodePool-op-2",
		Type: inspectioncore.TimelineTypeSubresource,
	})
	wantOp2DeleteNodepoolPath := builder.TimelineAccumulator.GetPath(wantNodepoolPath, khifilev6.PathSegment{
		Name: "DeleteNodePool-op-2",
		Type: inspectioncore.TimelineTypeSubresource,
	})
	wantOp2UnknownNodepoolPath := builder.TimelineAccumulator.GetPath(wantNodepoolPath, khifilev6.PathSegment{
		Name: "UnknownLongRunningOperation-op-2",
		Type: inspectioncore.TimelineTypeSubresource,
	})

	testCases := []struct {
		desc          string
		inputResource multicloudapiaudit.MulticloudAPIAuditResourceFieldSet
		inputAudit    gcpcommon.GCPAuditLogFieldSet
		inputTracker  *gcpcommon.GCPOperationTracker
		assert        func(t *testing.T, cs *khifilev6.TimelineChangeSet)
	}{
		{
			desc: "cluster create started",
			inputResource: multicloudapiaudit.MulticloudAPIAuditResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "",
				ClusterType:  multicloudapiaudit.ClusterTypeAWS,
			},
			inputAudit: gcpcommon.GCPAuditLogFieldSet{
				OperationID:    "op-1",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "google.cloud.gkemulticloud.v1.AwsClusters.CreateAwsCluster",
				PrincipalEmail: "foobar@qux.test",
				Request: testReaderFromYAML(t, `cluster:
  initialNodeCount: 1
  name: test-cluster`),
				ProjectID: "project-foo",
			},
			inputTracker: gcpcommon.NewGCPOperationTracker(),
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantClusterPath, &khifilev6.StagingRevision{
						ChangedTime: testTime,
						ResourceBody: testReaderFromYAML(t, `initialNodeCount: 1
name: test-cluster`).Node,
						Principal: "foobar@qux.test",
						VerbType:  k8saudit.VerbCreate,
						StateType: k8saudit.RevisionStateK8sClusterProvisioning,
					}, nodeComparer).
					HasRevision(wantOp1ClusterPath, &khifilev6.StagingRevision{
						ChangedTime: testTime,
						ResourceBody: testReaderFromYAML(t, `cluster:
  initialNodeCount: 1
  name: test-cluster`).Node,
						Principal: "foobar@qux.test",
						VerbType:  gcpcommon.VerbOperationStart,
						StateType: gcpcommon.RevisionStateOperationStarted,
					}, nodeComparer)
			},
		},
		{
			desc: "cluster create finished",
			inputResource: multicloudapiaudit.MulticloudAPIAuditResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "",
				ClusterType:  multicloudapiaudit.ClusterTypeAzure,
			},
			inputAudit: gcpcommon.GCPAuditLogFieldSet{
				OperationID:    "op-1",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.cloud.gkemulticloud.v1.AzureClusters.CreateAzureCluster",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
				ProjectID:      "project-foo",
			},
			inputTracker: gcpcommon.NewGCPOperationTracker(),
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantClusterPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: nil,
						Principal:    "foobar@qux.test",
						VerbType:     k8saudit.VerbCreate,
						StateType:    k8saudit.RevisionStateK8sClusterExisting,
					}, nodeComparer).
					HasRevision(wantOp1AzureClusterPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: nil,
						Principal:    "foobar@qux.test",
						VerbType:     gcpcommon.VerbOperationFinish,
						StateType:    gcpcommon.RevisionStateOperationSucceed,
					}, nodeComparer)
			},
		},
		{
			desc: "nodepool create started",
			inputResource: multicloudapiaudit.MulticloudAPIAuditResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
				ClusterType:  multicloudapiaudit.ClusterTypeAWS,
			},
			inputAudit: gcpcommon.GCPAuditLogFieldSet{
				OperationID:    "op-2",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "google.cloud.gkemulticloud.v1.AwsClusters.CreateAwsNodePool",
				PrincipalEmail: "foobar@qux.test",
				Request: testReaderFromYAML(t, `nodePool:
  initialNodeCount: 1
  name: test-nodepool`),
				ProjectID: "project-foo",
			},
			inputTracker: gcpcommon.NewGCPOperationTracker(),
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantNodepoolPath, &khifilev6.StagingRevision{
						ChangedTime: testTime,
						ResourceBody: testReaderFromYAML(t, `initialNodeCount: 1
name: test-nodepool`).Node,
						Principal: "foobar@qux.test",
						VerbType:  k8saudit.VerbCreate,
						StateType: k8saudit.RevisionStateK8sNodepoolProvisioning,
					}, nodeComparer).
					HasRevision(wantOp2NodepoolPath, &khifilev6.StagingRevision{
						ChangedTime: testTime,
						ResourceBody: testReaderFromYAML(t, `nodePool:
  initialNodeCount: 1
  name: test-nodepool`).Node,
						Principal: "foobar@qux.test",
						VerbType:  gcpcommon.VerbOperationStart,
						StateType: gcpcommon.RevisionStateOperationStarted,
					}, nodeComparer)
			},
		},
		{
			desc: "nodepool create finished",
			inputResource: multicloudapiaudit.MulticloudAPIAuditResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
				ClusterType:  multicloudapiaudit.ClusterTypeAzure,
			},
			inputAudit: gcpcommon.GCPAuditLogFieldSet{
				OperationID:    "op-2",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.cloud.gkemulticloud.v1.AzureClusters.CreateAzureNodePool",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
				ProjectID:      "project-foo",
			},
			inputTracker: gcpcommon.NewGCPOperationTracker(),
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantNodepoolPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: nil,
						Principal:    "foobar@qux.test",
						VerbType:     k8saudit.VerbCreate,
						StateType:    k8saudit.RevisionStateK8sNodepoolProvisioningLogNotFound,
					}, nodeComparer).
					HasRevision(wantNodepoolPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: nil,
						Principal:    "foobar@qux.test",
						VerbType:     k8saudit.VerbCreate,
						StateType:    k8saudit.RevisionStateK8sNodepoolExisting,
					}, nodeComparer).
					HasRevision(wantOp2AzureNodepoolPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: nil,
						Principal:    "foobar@qux.test",
						VerbType:     gcpcommon.VerbOperationFinish,
						StateType:    gcpcommon.RevisionStateOperationSucceed,
					}, nodeComparer)
			},
		},
		{
			desc: "nodepool deletion finished",
			inputResource: multicloudapiaudit.MulticloudAPIAuditResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
				ClusterType:  multicloudapiaudit.ClusterTypeAWS,
			},
			inputAudit: gcpcommon.GCPAuditLogFieldSet{
				OperationID:    "op-2",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.cloud.gkemulticloud.v1.AwsClusters.DeleteAwsNodePool",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
				ProjectID:      "project-foo",
			},
			inputTracker: gcpcommon.NewGCPOperationTracker(),
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantNodepoolPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: nil,
						Principal:    "foobar@qux.test",
						VerbType:     k8saudit.VerbDelete,
						StateType:    k8saudit.RevisionStateK8sNodepoolDeletingLogNotFound,
					}, nodeComparer).
					HasRevision(wantNodepoolPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: nil,
						Principal:    "foobar@qux.test",
						VerbType:     k8saudit.VerbDelete,
						StateType:    k8saudit.RevisionStateK8sNodepoolDeleted,
					}, nodeComparer).
					HasRevision(wantOp2DeleteNodepoolPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: nil,
						Principal:    "foobar@qux.test",
						VerbType:     gcpcommon.VerbOperationFinish,
						StateType:    gcpcommon.RevisionStateOperationSucceed,
					}, nodeComparer)
			},
		},
		{
			desc: "immediate action",
			inputResource: multicloudapiaudit.MulticloudAPIAuditResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
				ClusterType:  multicloudapiaudit.ClusterTypeAzure,
			},
			inputAudit: gcpcommon.GCPAuditLogFieldSet{
				OperationID:    "op-2",
				OperationFirst: true,
				OperationLast:  true,
				MethodName:     "google.cloud.gkemulticloud.v1.AzureClusters.UpdateAzureCluster",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
				ProjectID:      "project-foo",
			},
			inputTracker: gcpcommon.NewGCPOperationTracker(),
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantNodepoolPath)
			},
		},
		{
			desc: "long running action for unknown cluster type",
			inputResource: multicloudapiaudit.MulticloudAPIAuditResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
				ClusterType:  multicloudapiaudit.ClusterTypeUnknown,
			},
			inputAudit: gcpcommon.GCPAuditLogFieldSet{
				OperationID:    "op-2",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "google.cloud.gkemulticloud.v1.FooClusters.UnknownLongRunningOperation",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
				ProjectID:      "project-foo",
			},
			inputTracker: gcpcommon.NewGCPOperationTracker(),
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantOp2UnknownNodepoolPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: nil,
						Principal:    "foobar@qux.test",
						VerbType:     gcpcommon.VerbOperationStart,
						StateType:    gcpcommon.RevisionStateOperationStarted,
					}, nodeComparer)
			},
		},
	}

	mapper := &multicloudAuditLogLogToTimelineMapperSetting{}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l := testlog.NewMockLog(testTime, tc.inputAudit, tc.inputResource)

			ctx := khictx.WithValue(t.Context(), inspectioncore.Builder, builder)
			cs, _, err := mapper.ProcessLogByGroup(ctx, l, tc.inputTracker)
			if err != nil {
				t.Fatalf("ProcessLogByGroup() returned unexpected error: %v", err)
			}

			tc.assert(t, cs)
		})
	}
}
