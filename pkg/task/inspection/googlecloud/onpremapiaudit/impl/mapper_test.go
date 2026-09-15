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

package onpremapiaudit_impl

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/model/id"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/onpremapiaudit"
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

func TestOnPremAPIAuditLogIngester_ProcessLog(t *testing.T) {
	testCases := []struct {
		name        string
		inputLog    *log.Log
		wantSummary string
	}{
		{
			name: "operation starting log",
			inputLog: testlog.NewMockLog(
				time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC),
				gcpcommon.GCPAuditLogFieldSet{
					MethodName:     "google.cloud.gkeonprem.v1.GkeOnPrem.CreateBaremetalAdminCluster",
					OperationFirst: true,
					OperationLast:  false,
				},
			),
			wantSummary: "Start: google.cloud.gkeonprem.v1.GkeOnPrem.CreateBaremetalAdminCluster",
		},
		{
			name: "operation ending log",
			inputLog: testlog.NewMockLog(
				time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC),
				gcpcommon.GCPAuditLogFieldSet{
					MethodName:     "google.cloud.gkeonprem.v1.GkeOnPrem.CreateBaremetalAdminCluster",
					OperationFirst: false,
					OperationLast:  true,
				},
			),
			wantSummary: "Succeeded: google.cloud.gkeonprem.v1.GkeOnPrem.CreateBaremetalAdminCluster",
		},
	}

	ingester := gcpcommon.NewGCPOperationLogIngester(onpremapiaudit.ListLogEntriesTaskID.Ref(), onpremapiaudit.LogTypeOnPremAPI)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cs, err := ingester.ProcessLog(t.Context(), tc.inputLog)
			if err != nil {
				t.Fatalf("ProcessLog() returned unexpected error: %v", err)
			}

			testchangeset.AssertLog(t, cs).
				HasSummary(tc.wantSummary).
				HasLogType(onpremapiaudit.LogTypeOnPremAPI)
		})
	}
}

func TestOnPremAPIAuditTimelineMapper_ProcessLogByGroup(t *testing.T) {
	testTime := time.Date(2025, time.January, 1, 1, 1, 1, 1, time.UTC)

	// 1. Initialize the Builder.
	builder := khifilev6.NewTestBuilder(id.NewGenerator())

	cmpNode := cmp.Comparer(func(x, y structured.Node) bool {
		if x == nil && y == nil {
			return true
		}
		if x == nil || y == nil {
			return false
		}
		serializer := &structured.YAMLNodeSerializer{}
		xBytes, errX := serializer.Serialize(x)
		yBytes, errY := serializer.Serialize(y)
		if errX != nil || errY != nil {
			return false
		}
		return string(xBytes) == string(yBytes)
	})

	// 2. Construct expected timeline paths independently.
	wantProjPath := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
		Name: "test-project",
		Type: gcpcommon.TimelineTypeGCPProject,
	})
	wantClusterPath := builder.TimelineAccumulator.GetPath(wantProjPath, khifilev6.PathSegment{
		Name: "test-cluster",
		Type: onpremapiaudit.TimelineTypeOnPremCluster,
	})
	wantNodePoolPath := builder.TimelineAccumulator.GetPath(wantClusterPath, khifilev6.PathSegment{
		Name: "test-nodepool",
		Type: onpremapiaudit.TimelineTypeOnPremNodePool,
	})

	wantClusterOpPath1 := builder.TimelineAccumulator.GetPath(wantClusterPath, khifilev6.PathSegment{
		Name: "CreateBaremetalAdminCluster-op-1",
		Type: gcpcommon.TimelineTypeOperation,
	})
	wantClusterOpPath2 := builder.TimelineAccumulator.GetPath(wantClusterPath, khifilev6.PathSegment{
		Name: "CreateBaremetalStandaloneCluster-op-1",
		Type: gcpcommon.TimelineTypeOperation,
	})
	wantClusterOpPath3 := builder.TimelineAccumulator.GetPath(wantClusterPath, khifilev6.PathSegment{
		Name: "EnrollBaremetalStandaloneCluster-op-1",
		Type: gcpcommon.TimelineTypeOperation,
	})
	wantClusterOpPath4 := builder.TimelineAccumulator.GetPath(wantClusterPath, khifilev6.PathSegment{
		Name: "UnknownLongRunningOperation-op-2",
		Type: gcpcommon.TimelineTypeOperation,
	})
	wantNodePoolOpPath1 := builder.TimelineAccumulator.GetPath(wantNodePoolPath, khifilev6.PathSegment{
		Name: "CreateBaremetalNodePool-op-2",
		Type: gcpcommon.TimelineTypeOperation,
	})
	wantNodePoolOpPath2 := builder.TimelineAccumulator.GetPath(wantNodePoolPath, khifilev6.PathSegment{
		Name: "CreateVmwareAdminNodePool-op-2",
		Type: gcpcommon.TimelineTypeOperation,
	})
	wantNodePoolOpPath3 := builder.TimelineAccumulator.GetPath(wantNodePoolPath, khifilev6.PathSegment{
		Name: "DeleteVmwareNodePool-op-2",
		Type: gcpcommon.TimelineTypeOperation,
	})
	wantNodePoolOpPath4 := builder.TimelineAccumulator.GetPath(wantNodePoolPath, khifilev6.PathSegment{
		Name: "UnenrollVmwareNodePool-op-2",
		Type: gcpcommon.TimelineTypeOperation,
	})

	testCases := []struct {
		desc          string
		inputResource onpremapiaudit.OnPremAPIAuditResourceFieldSet
		inputAudit    gcpcommon.GCPAuditLogFieldSet
		inputTracker  *gcpcommon.GCPOperationTracker
		assert        func(t *testing.T, cs *khifilev6.TimelineChangeSet)
	}{
		{
			desc: "cluster create started",
			inputResource: onpremapiaudit.OnPremAPIAuditResourceFieldSet{
				Project:      "test-project",
				ClusterName:  "test-cluster",
				NodepoolName: "",
				ClusterType:  onpremapiaudit.ClusterTypeBaremetalAdmin,
			},
			inputAudit: gcpcommon.GCPAuditLogFieldSet{
				OperationID:    "op-1",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "google.cloud.gkeonprem.v1.GkeOnPrem.CreateBaremetalAdminCluster",
				PrincipalEmail: "foobar@qux.test",
				Request: testReaderFromYAML(t, `cluster:
  initialNodeCount: 1
  name: test-cluster`),
			},
			inputTracker: gcpcommon.NewGCPOperationTracker(),
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				var reqNode structured.Node
				if node, err := structured.FromYAML("initialNodeCount: 1\nname: test-cluster\n"); err == nil {
					reqNode = node
				}
				var opNode structured.Node
				if node, err := structured.FromYAML("cluster:\n  initialNodeCount: 1\n  name: test-cluster\n"); err == nil {
					opNode = node
				}

				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantClusterPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: reqNode,
						Principal:    "foobar@qux.test",
						VerbType:     k8saudit.VerbCreate,
						StateType:    k8saudit.RevisionStateK8sClusterProvisioning,
					}, cmpNode).
					HasRevision(wantClusterOpPath1, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: opNode,
						Principal:    "foobar@qux.test",
						VerbType:     gcpcommon.VerbOperationStart,
						StateType:    gcpcommon.RevisionStateOperationStarted,
					}, cmpNode)
			},
		},
		{
			desc: "cluster create finished",
			inputResource: onpremapiaudit.OnPremAPIAuditResourceFieldSet{
				Project:      "test-project",
				ClusterName:  "test-cluster",
				NodepoolName: "",
				ClusterType:  onpremapiaudit.ClusterTypeBaremetalStandalone,
			},
			inputAudit: gcpcommon.GCPAuditLogFieldSet{
				OperationID:    "op-1",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.cloud.gkeonprem.v1.GkeOnPrem.CreateBaremetalStandaloneCluster",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
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
					}, cmpNode).
					HasRevision(wantClusterOpPath2, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: nil,
						Principal:    "foobar@qux.test",
						VerbType:     gcpcommon.VerbOperationFinish,
						StateType:    gcpcommon.RevisionStateOperationSucceed,
					}, cmpNode)
			},
		},
		{
			desc: "cluster enroll finished",
			inputResource: onpremapiaudit.OnPremAPIAuditResourceFieldSet{
				Project:      "test-project",
				ClusterName:  "test-cluster",
				NodepoolName: "",
				ClusterType:  onpremapiaudit.ClusterTypeBaremetalStandalone,
			},
			inputAudit: gcpcommon.GCPAuditLogFieldSet{
				OperationID:    "op-1",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.cloud.gkeonprem.v1.GkeOnPrem.EnrollBaremetalStandaloneCluster",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
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
					}, cmpNode).
					HasRevision(wantClusterOpPath3, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: nil,
						Principal:    "foobar@qux.test",
						VerbType:     gcpcommon.VerbOperationFinish,
						StateType:    gcpcommon.RevisionStateOperationSucceed,
					}, cmpNode)
			},
		},
		{
			desc: "nodepool create started",
			inputResource: onpremapiaudit.OnPremAPIAuditResourceFieldSet{
				Project:      "test-project",
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
				ClusterType:  onpremapiaudit.ClusterTypeBaremetalUser,
			},
			inputAudit: gcpcommon.GCPAuditLogFieldSet{
				OperationID:    "op-2",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "google.cloud.gkeonprem.v1.GkeOnPrem.CreateBaremetalNodePool",
				PrincipalEmail: "foobar@qux.test",
				Request: testReaderFromYAML(t, `nodePool:
  initialNodeCount: 1
  name: test-nodepool`),
			},
			inputTracker: gcpcommon.NewGCPOperationTracker(),
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				var reqNode structured.Node
				if node, err := structured.FromYAML("initialNodeCount: 1\nname: test-nodepool\n"); err == nil {
					reqNode = node
				}
				var opNode structured.Node
				if node, err := structured.FromYAML("nodePool:\n  initialNodeCount: 1\n  name: test-nodepool\n"); err == nil {
					opNode = node
				}

				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantNodePoolPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: reqNode,
						Principal:    "foobar@qux.test",
						VerbType:     k8saudit.VerbCreate,
						StateType:    k8saudit.RevisionStateK8sNodepoolProvisioning,
					}, cmpNode).
					HasRevision(wantNodePoolOpPath1, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: opNode,
						Principal:    "foobar@qux.test",
						VerbType:     gcpcommon.VerbOperationStart,
						StateType:    gcpcommon.RevisionStateOperationStarted,
					}, cmpNode)
			},
		},
		{
			desc: "nodepool create finished",
			inputResource: onpremapiaudit.OnPremAPIAuditResourceFieldSet{
				Project:      "test-project",
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
				ClusterType:  onpremapiaudit.ClusterTypeVMWareAdmin,
			},
			inputAudit: gcpcommon.GCPAuditLogFieldSet{
				OperationID:    "op-2",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.cloud.gkeonprem.v1.GkeOnPrem.CreateVmwareAdminNodePool",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
			},
			inputTracker: gcpcommon.NewGCPOperationTracker(),
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantNodePoolPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: nil,
						Principal:    "foobar@qux.test",
						VerbType:     k8saudit.VerbCreate,
						StateType:    k8saudit.RevisionStateK8sNodepoolProvisioningLogNotFound,
					}, cmpNode).
					HasRevision(wantNodePoolPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: nil,
						Principal:    "foobar@qux.test",
						VerbType:     k8saudit.VerbCreate,
						StateType:    k8saudit.RevisionStateK8sNodepoolExisting,
					}, cmpNode).
					HasRevision(wantNodePoolOpPath2, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: nil,
						Principal:    "foobar@qux.test",
						VerbType:     gcpcommon.VerbOperationFinish,
						StateType:    gcpcommon.RevisionStateOperationSucceed,
					}, cmpNode)
			},
		},
		{
			desc: "nodepool deletion finished",
			inputResource: onpremapiaudit.OnPremAPIAuditResourceFieldSet{
				Project:      "test-project",
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
				ClusterType:  onpremapiaudit.ClusterTypeVMWareUser,
			},
			inputAudit: gcpcommon.GCPAuditLogFieldSet{
				OperationID:    "op-2",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.cloud.gkeonprem.v1.GkeOnPrem.DeleteVmwareNodePool",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
			},
			inputTracker: gcpcommon.NewGCPOperationTracker(),
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantNodePoolPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: nil,
						Principal:    "foobar@qux.test",
						VerbType:     k8saudit.VerbDelete,
						StateType:    k8saudit.RevisionStateK8sNodepoolDeletingLogNotFound,
					}, cmpNode).
					HasRevision(wantNodePoolPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: nil,
						Principal:    "foobar@qux.test",
						VerbType:     k8saudit.VerbDelete,
						StateType:    k8saudit.RevisionStateK8sNodepoolDeleted,
					}, cmpNode).
					HasRevision(wantNodePoolOpPath3, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: nil,
						Principal:    "foobar@qux.test",
						VerbType:     gcpcommon.VerbOperationFinish,
						StateType:    gcpcommon.RevisionStateOperationSucceed,
					}, cmpNode)
			},
		},
		{
			desc: "nodepool unenroll finished",
			inputResource: onpremapiaudit.OnPremAPIAuditResourceFieldSet{
				Project:      "test-project",
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
				ClusterType:  onpremapiaudit.ClusterTypeVMWareUser,
			},
			inputAudit: gcpcommon.GCPAuditLogFieldSet{
				OperationID:    "op-2",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.cloud.gkeonprem.v1.GkeOnPrem.UnenrollVmwareNodePool",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
			},
			inputTracker: gcpcommon.NewGCPOperationTracker(),
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantNodePoolPath, &khifilev6.StagingRevision{
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: nil,
						Principal:    "foobar@qux.test",
						VerbType:     k8saudit.VerbDelete,
						StateType:    k8saudit.RevisionStateK8sNodepoolDeletingLogNotFound,
					}, cmpNode).
					HasRevision(wantNodePoolPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: nil,
						Principal:    "foobar@qux.test",
						VerbType:     k8saudit.VerbDelete,
						StateType:    k8saudit.RevisionStateK8sNodepoolDeleted,
					}, cmpNode).
					HasRevision(wantNodePoolOpPath4, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: nil,
						Principal:    "foobar@qux.test",
						VerbType:     gcpcommon.VerbOperationFinish,
						StateType:    gcpcommon.RevisionStateOperationSucceed,
					}, cmpNode)
			},
		},
		{
			desc: "immediate action",
			inputResource: onpremapiaudit.OnPremAPIAuditResourceFieldSet{
				Project:      "test-project",
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
				ClusterType:  onpremapiaudit.ClusterTypeVMWareUser,
			},
			inputAudit: gcpcommon.GCPAuditLogFieldSet{
				OperationID:    "op-2",
				OperationFirst: true,
				OperationLast:  true,
				MethodName:     "google.cloud.gkeonprem.v1.GkeOnPrem.UpdateVmwareCluster",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
			},
			inputTracker: gcpcommon.NewGCPOperationTracker(),
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantNodePoolPath)
			},
		},
		{
			desc: "long running operation for unknown cluster",
			inputResource: onpremapiaudit.OnPremAPIAuditResourceFieldSet{
				Project:      "test-project",
				ClusterName:  "test-cluster",
				NodepoolName: "",
				ClusterType:  onpremapiaudit.ClusterTypeUnknown,
			},
			inputAudit: gcpcommon.GCPAuditLogFieldSet{
				OperationID:    "op-2",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "google.cloud.gkeonprem.v1.GkeOnPrem.UnknownLongRunningOperation",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
			},
			inputTracker: gcpcommon.NewGCPOperationTracker(),
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantClusterOpPath4, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						ResourceBody: nil,
						Principal:    "foobar@qux.test",
						VerbType:     gcpcommon.VerbOperationStart,
						StateType:    gcpcommon.RevisionStateOperationStarted,
					}, cmpNode)
			},
		},
	}

	mapper := &OnPremAPIAuditTimelineMapper{}
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
