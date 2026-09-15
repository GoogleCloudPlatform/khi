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

package googlecloudloggkeapiaudit_impl

import (
	"fmt"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/model/id"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudloggkeapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudloggkeapiaudit/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
	"github.com/google/go-cmp/cmp"
)

type mockInitialResourceStateProvider struct {
	clusterStates  map[string]*googlecloudloggkeapiaudit_contract.InitialResourceState
	nodePoolStates map[string]*googlecloudloggkeapiaudit_contract.InitialResourceState
}

func (m *mockInitialResourceStateProvider) ClusterInitialState(clusterName string) (*googlecloudloggkeapiaudit_contract.InitialResourceState, bool) {
	if m.clusterStates == nil {
		return nil, false
	}
	state, ok := m.clusterStates[clusterName]
	return state, ok
}

func (m *mockInitialResourceStateProvider) NodePoolInitialState(clusterName, nodePoolName string) (*googlecloudloggkeapiaudit_contract.InitialResourceState, bool) {
	if m.nodePoolStates == nil {
		return nil, false
	}
	state, ok := m.nodePoolStates[fmt.Sprintf("%s/%s", clusterName, nodePoolName)]
	return state, ok
}

var (
	pathTestCluster  = structured.CompileFieldPath("cluster")
	pathTestNodePool = structured.CompileFieldPath("nodePool")
)

func testReaderFromYAML(t *testing.T, yaml string) *structured.NodeReader {
	t.Helper()
	node, err := structured.FromYAML(yaml)
	if err != nil {
		t.Fatalf("failed to parse yaml: %v", err)
	}
	return structured.NewNodeReader(node)
}

var compareNodeOption = cmp.Transformer("StructuredNodeToYAML", func(n structured.Node) string {
	if n == nil {
		return ""
	}
	serializer := &structured.YAMLNodeSerializer{}
	bytes, err := serializer.Serialize(n)
	if err != nil {
		return "serialization error"
	}
	return string(bytes)
})

func TestLogToTimelineMapperTask(t *testing.T) {
	// 1. Initialize the Builder.
	builder := khifilev6.NewTestBuilder(id.NewGenerator())

	// 2. Set up expected path references.
	wantProjectPath := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
		Name: "test-project",
		Type: googlecloudcommon_contract.TimelineTypeGCPProject,
	})
	wantClusterPath := builder.TimelineAccumulator.GetPath(wantProjectPath, khifilev6.PathSegment{
		Name: "test-cluster",
		Type: googlecloudcommon_contract.TimelineTypeGKE,
	})
	wantNodepoolsPath := builder.TimelineAccumulator.GetPath(wantClusterPath, khifilev6.PathSegment{
		Name: "nodepools",
		Type: googlecloudcommon_contract.TimelineTypeGKENodePools,
	})
	wantNodepoolPath := builder.TimelineAccumulator.GetPath(wantNodepoolsPath, khifilev6.PathSegment{
		Name: "test-nodepool",
		Type: googlecloudcommon_contract.TimelineTypeGKENodePool,
	})
	wantClusterOpPath := builder.TimelineAccumulator.GetPath(wantClusterPath, khifilev6.PathSegment{
		Name: "CreateCluster-op-1",
		Type: googlecloudcommon_contract.TimelineTypeOperation,
	})
	wantNodepoolOp1Path := builder.TimelineAccumulator.GetPath(wantNodepoolPath, khifilev6.PathSegment{
		Name: "CreateNodePool-op-2",
		Type: googlecloudcommon_contract.TimelineTypeOperation,
	})
	wantNodepoolOp2Path := builder.TimelineAccumulator.GetPath(wantNodepoolPath, khifilev6.PathSegment{
		Name: "DeleteNodePool-op-2",
		Type: googlecloudcommon_contract.TimelineTypeOperation,
	})
	wantClusterOpUpdatePath := builder.TimelineAccumulator.GetPath(wantClusterPath, khifilev6.PathSegment{
		Name: "UpdateCluster-op-3",
		Type: googlecloudcommon_contract.TimelineTypeOperation,
	})
	wantNodepoolOpUpdatePath := builder.TimelineAccumulator.GetPath(wantNodepoolPath, khifilev6.PathSegment{
		Name: "UpdateNodePool-op-3",
		Type: googlecloudcommon_contract.TimelineTypeOperation,
	})

	testTime := time.Date(2025, time.January, 1, 1, 1, 1, 1, time.UTC)

	testCases := []struct {
		desc                 string
		inputResource        googlecloudloggkeapiaudit_contract.GKEAuditLogResourceFieldSet
		inputAudit           googlecloudcommon_contract.GCPAuditLogFieldSet
		inputTracker         *googlecloudcommon_contract.GCPOperationTracker
		initialStateProvider googlecloudloggkeapiaudit_contract.InitialResourceStateProvider
		assert               func(t *testing.T, cs *khifilev6.TimelineChangeSet)
		assertTracker        func(t *testing.T, tracker *googlecloudcommon_contract.GCPOperationTracker)
	}{
		{
			desc: "cluster create started",
			inputResource: googlecloudloggkeapiaudit_contract.GKEAuditLogResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "",
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "op-1",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "google.container.v1.ClusterManager.CreateCluster",
				PrincipalEmail: "foobar@qux.test",
				Request: testReaderFromYAML(t, `cluster:
  initialNodeCount: 1
  name: test-cluster`),
			},
			inputTracker: googlecloudcommon_contract.NewGCPOperationTracker(),
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				var bodyNode structured.Node
				if subReader, err := testReaderFromYAML(t, `cluster:
  initialNodeCount: 1
  name: test-cluster`).GetReader(pathTestCluster); err == nil {
					bodyNode = subReader.Node
				}

				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantClusterPath, &khifilev6.StagingRevision{
						VerbType:     k8saudit.VerbCreate,
						StateType:    k8saudit.RevisionStateK8sClusterProvisioning,
						Principal:    "foobar@qux.test",
						ChangedTime:  testTime,
						ResourceBody: bodyNode,
					}, compareNodeOption).
					HasRevision(wantClusterOpPath, &khifilev6.StagingRevision{
						VerbType:    googlecloudcommon_contract.VerbOperationStart,
						StateType:   googlecloudcommon_contract.RevisionStateOperationStarted,
						Principal:   "foobar@qux.test",
						ChangedTime: testTime,
						ResourceBody: testReaderFromYAML(t, `cluster:
  initialNodeCount: 1
  name: test-cluster`).Node,
					}, compareNodeOption)
			},
		},
		{
			desc: "cluster create finished",
			inputResource: googlecloudloggkeapiaudit_contract.GKEAuditLogResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "",
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "op-1",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.container.v1.ClusterManager.CreateCluster",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
			},
			inputTracker: googlecloudcommon_contract.NewGCPOperationTracker(),
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantClusterPath, &khifilev6.StagingRevision{
						VerbType:    k8saudit.VerbCreate,
						StateType:   k8saudit.RevisionStateK8sClusterExisting,
						Principal:   "foobar@qux.test",
						ChangedTime: testTime,
					}, compareNodeOption).
					HasRevision(wantClusterOpPath, &khifilev6.StagingRevision{
						VerbType:    googlecloudcommon_contract.VerbOperationFinish,
						StateType:   googlecloudcommon_contract.RevisionStateOperationSucceed,
						Principal:   "foobar@qux.test",
						ChangedTime: testTime,
					}, compareNodeOption)
			},
		},
		{
			desc: "nodepool create started",
			inputResource: googlecloudloggkeapiaudit_contract.GKEAuditLogResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "op-2",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "google.container.v1.ClusterManager.CreateNodePool",
				PrincipalEmail: "foobar@qux.test",
				Request: testReaderFromYAML(t, `nodePool:
  initialNodeCount: 1
  name: test-nodepool`),
			},
			inputTracker: googlecloudcommon_contract.NewGCPOperationTracker(),
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				var bodyNode structured.Node
				if subReader, err := testReaderFromYAML(t, `nodePool:
  initialNodeCount: 1
  name: test-nodepool`).GetReader(pathTestNodePool); err == nil {
					bodyNode = subReader.Node
				}

				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantNodepoolPath, &khifilev6.StagingRevision{
						VerbType:     k8saudit.VerbCreate,
						StateType:    k8saudit.RevisionStateK8sNodepoolProvisioning,
						Principal:    "foobar@qux.test",
						ChangedTime:  testTime,
						ResourceBody: bodyNode,
					}, compareNodeOption).
					HasRevision(wantNodepoolOp1Path, &khifilev6.StagingRevision{
						VerbType:    googlecloudcommon_contract.VerbOperationStart,
						StateType:   googlecloudcommon_contract.RevisionStateOperationStarted,
						Principal:   "foobar@qux.test",
						ChangedTime: testTime,
						ResourceBody: testReaderFromYAML(t, `nodePool:
  initialNodeCount: 1
  name: test-nodepool`).Node,
					}, compareNodeOption)
			},
		},
		{
			desc: "nodepool create finished",
			inputResource: googlecloudloggkeapiaudit_contract.GKEAuditLogResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "op-2",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.container.v1.ClusterManager.CreateNodePool",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
			},
			inputTracker: googlecloudcommon_contract.NewGCPOperationTracker(),
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantNodepoolPath, &khifilev6.StagingRevision{
						VerbType:    k8saudit.VerbCreate,
						StateType:   k8saudit.RevisionStateK8sNodepoolProvisioningLogNotFound,
						Principal:   "foobar@qux.test",
						ChangedTime: time.Unix(0, 0),
					}, compareNodeOption).
					HasRevision(wantNodepoolPath, &khifilev6.StagingRevision{
						VerbType:    k8saudit.VerbCreate,
						StateType:   k8saudit.RevisionStateK8sNodepoolExisting,
						Principal:   "foobar@qux.test",
						ChangedTime: testTime,
					}, compareNodeOption).
					HasRevision(wantNodepoolOp1Path, &khifilev6.StagingRevision{
						VerbType:    googlecloudcommon_contract.VerbOperationFinish,
						StateType:   googlecloudcommon_contract.RevisionStateOperationSucceed,
						Principal:   "foobar@qux.test",
						ChangedTime: testTime,
					}, compareNodeOption)
			},
		},
		{
			desc: "nodepool deletion finished",
			inputResource: googlecloudloggkeapiaudit_contract.GKEAuditLogResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "op-2",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.container.v1.ClusterManager.DeleteNodePool",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
			},
			inputTracker: googlecloudcommon_contract.NewGCPOperationTracker(),
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantNodepoolPath, &khifilev6.StagingRevision{
						VerbType:    k8saudit.VerbDelete,
						StateType:   k8saudit.RevisionStateK8sNodepoolDeletingLogNotFound,
						Principal:   "foobar@qux.test",
						ChangedTime: time.Unix(0, 0),
					}, compareNodeOption).
					HasRevision(wantNodepoolPath, &khifilev6.StagingRevision{
						VerbType:    k8saudit.VerbDelete,
						StateType:   k8saudit.RevisionStateK8sNodepoolDeleted,
						Principal:   "foobar@qux.test",
						ChangedTime: testTime,
					}, compareNodeOption).
					HasRevision(wantNodepoolOp2Path, &khifilev6.StagingRevision{
						VerbType:    googlecloudcommon_contract.VerbOperationFinish,
						StateType:   googlecloudcommon_contract.RevisionStateOperationSucceed,
						Principal:   "foobar@qux.test",
						ChangedTime: testTime,
					}, compareNodeOption)
			},
		},
		{
			desc: "immediate action",
			inputResource: googlecloudloggkeapiaudit_contract.GKEAuditLogResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "op-2",
				OperationFirst: true,
				OperationLast:  true,
				MethodName:     "google.container.v1.ClusterManager.UpdateCluster",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
			},
			inputTracker: googlecloudcommon_contract.NewGCPOperationTracker(),
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantNodepoolPath)
			},
		},
		{
			desc: "cluster update finished with initial state merges manifest",
			inputResource: googlecloudloggkeapiaudit_contract.GKEAuditLogResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "",
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "op-3",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.container.v1.ClusterManager.UpdateCluster",
				PrincipalEmail: "foobar@qux.test",
				Request: testReaderFromYAML(t, `update:
  desiredNodePoolId: np-1`),
			},
			inputTracker: googlecloudcommon_contract.NewGCPOperationTracker(),
			initialStateProvider: &mockInitialResourceStateProvider{
				clusterStates: map[string]*googlecloudloggkeapiaudit_contract.InitialResourceState{
					"test-cluster": {
						ResourceBody: testReaderFromYAML(t, "initialNodeCount: 1\nname: test-cluster").Node,
					},
				},
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				wantBody := testReaderFromYAML(t, `initialNodeCount: 1
name: test-cluster
desiredNodePoolId: np-1`).Node

				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantClusterPath, &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbUpdate,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sClusterExisting,
						Principal:    "foobar@qux.test",
						ChangedTime:  testTime,
						ResourceBody: wantBody,
					}, compareNodeOption).
					HasRevision(wantClusterOpUpdatePath, &khifilev6.StagingRevision{
						VerbType:    googlecloudcommon_contract.VerbOperationFinish,
						StateType:   googlecloudcommon_contract.RevisionStateOperationSucceed,
						Principal:   "foobar@qux.test",
						ChangedTime: testTime,
						ResourceBody: testReaderFromYAML(t, `update:
  desiredNodePoolId: np-1`).Node,
					}, compareNodeOption)
			},
		},
		{
			desc: "cluster update finished without initial state stages LogNotFound dummy revision",
			inputResource: googlecloudloggkeapiaudit_contract.GKEAuditLogResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "",
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "op-3",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.container.v1.ClusterManager.UpdateCluster",
				PrincipalEmail: "foobar@qux.test",
				Request: testReaderFromYAML(t, `update:
  desiredNodePoolId: np-1`),
			},
			inputTracker: googlecloudcommon_contract.NewGCPOperationTracker(),
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				var patchBody structured.Node
				if subReader, err := testReaderFromYAML(t, `update:
  desiredNodePoolId: np-1`).GetReader(pathUpdate); err == nil {
					patchBody = subReader.Node
				}

				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantClusterPath, &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbCreate,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sClusterExistingLogNotFound,
						Principal:    "foobar@qux.test",
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: nil,
					}, compareNodeOption).
					HasRevision(wantClusterPath, &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbUpdate,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sClusterExisting,
						Principal:    "foobar@qux.test",
						ChangedTime:  testTime,
						ResourceBody: patchBody,
					}, compareNodeOption).
					HasRevision(wantClusterOpUpdatePath, &khifilev6.StagingRevision{
						VerbType:    googlecloudcommon_contract.VerbOperationFinish,
						StateType:   googlecloudcommon_contract.RevisionStateOperationSucceed,
						Principal:   "foobar@qux.test",
						ChangedTime: testTime,
						ResourceBody: testReaderFromYAML(t, `update:
  desiredNodePoolId: np-1`).Node,
					}, compareNodeOption)
			},
		},
		{
			desc: "nodepool update finished with initial state merges manifest",
			inputResource: googlecloudloggkeapiaudit_contract.GKEAuditLogResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "op-3",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.container.v1.ClusterManager.UpdateNodePool",
				PrincipalEmail: "foobar@qux.test",
				Request: testReaderFromYAML(t, `nodePool:
  initialNodeCount: 5`),
			},
			inputTracker: googlecloudcommon_contract.NewGCPOperationTracker(),
			initialStateProvider: &mockInitialResourceStateProvider{
				nodePoolStates: map[string]*googlecloudloggkeapiaudit_contract.InitialResourceState{
					"test-cluster/test-nodepool": {
						ResourceBody: testReaderFromYAML(t, "initialNodeCount: 3\nname: test-nodepool").Node,
					},
				},
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				wantBody := testReaderFromYAML(t, `initialNodeCount: 5
name: test-nodepool`).Node

				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantNodepoolPath, &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbUpdate,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sNodepoolExisting,
						Principal:    "foobar@qux.test",
						ChangedTime:  testTime,
						ResourceBody: wantBody,
					}, compareNodeOption).
					HasRevision(wantNodepoolOpUpdatePath, &khifilev6.StagingRevision{
						VerbType:    googlecloudcommon_contract.VerbOperationFinish,
						StateType:   googlecloudcommon_contract.RevisionStateOperationSucceed,
						Principal:   "foobar@qux.test",
						ChangedTime: testTime,
						ResourceBody: testReaderFromYAML(t, `nodePool:
  initialNodeCount: 5`).Node,
					}, compareNodeOption)
			},
		},
		{
			desc: "nodepool update finished without initial state stages LogNotFound dummy revision",
			inputResource: googlecloudloggkeapiaudit_contract.GKEAuditLogResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "op-3",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.container.v1.ClusterManager.UpdateNodePool",
				PrincipalEmail: "foobar@qux.test",
				Request: testReaderFromYAML(t, `nodePool:
  initialNodeCount: 5`),
			},
			inputTracker: googlecloudcommon_contract.NewGCPOperationTracker(),
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				var patchBody structured.Node
				if subReader, err := testReaderFromYAML(t, `nodePool:
  initialNodeCount: 5`).GetReader(pathTestNodePool); err == nil {
					patchBody = subReader.Node
				}

				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantNodepoolPath, &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbCreate,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sNodepoolExistingLogNotFound,
						Principal:    "foobar@qux.test",
						ChangedTime:  time.Unix(0, 0),
						ResourceBody: nil,
					}, compareNodeOption).
					HasRevision(wantNodepoolPath, &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbUpdate,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sNodepoolExisting,
						Principal:    "foobar@qux.test",
						ChangedTime:  testTime,
						ResourceBody: patchBody,
					}, compareNodeOption).
					HasRevision(wantNodepoolOpUpdatePath, &khifilev6.StagingRevision{
						VerbType:    googlecloudcommon_contract.VerbOperationFinish,
						StateType:   googlecloudcommon_contract.RevisionStateOperationSucceed,
						Principal:   "foobar@qux.test",
						ChangedTime: testTime,
						ResourceBody: testReaderFromYAML(t, `nodePool:
  initialNodeCount: 5`).Node,
					}, compareNodeOption)
			},
		},
		{
			desc: "nodepool delete with initial state suppresses duplicate LogNotFound revision",
			inputResource: googlecloudloggkeapiaudit_contract.GKEAuditLogResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "op-2",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "google.container.v1.ClusterManager.DeleteNodePool",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
			},
			inputTracker: googlecloudcommon_contract.NewGCPOperationTracker(),
			initialStateProvider: &mockInitialResourceStateProvider{
				nodePoolStates: map[string]*googlecloudloggkeapiaudit_contract.InitialResourceState{
					"test-cluster/test-nodepool": {
						ResourceBody: testReaderFromYAML(t, "initialNodeCount: 3\nname: test-nodepool").Node,
					},
				},
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				revs := cs.GetRevisions(wantNodepoolPath)
				if len(revs) != 1 {
					t.Fatalf("cs.GetRevisions(%v) count = %d, want 1 (LogNotFound was not suppressed)", wantNodepoolPath, len(revs))
				}
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantNodepoolPath, &khifilev6.StagingRevision{
						VerbType:    commonlogk8saudit_contract.VerbDelete,
						StateType:   commonlogk8saudit_contract.RevisionStateK8sNodepoolDeleting,
						Principal:   "foobar@qux.test",
						ChangedTime: testTime,
					}, compareNodeOption).
					HasRevision(wantNodepoolOp2Path, &khifilev6.StagingRevision{
						VerbType:    googlecloudcommon_contract.VerbOperationStart,
						StateType:   googlecloudcommon_contract.RevisionStateOperationStarted,
						Principal:   "foobar@qux.test",
						ChangedTime: testTime,
					}, compareNodeOption)
			},
		},
		{
			desc: "cluster create started with request manifest caches manifest in tracker",
			inputResource: googlecloudloggkeapiaudit_contract.GKEAuditLogResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "",
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "op-1",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "google.container.v1.ClusterManager.CreateCluster",
				PrincipalEmail: "foobar@qux.test",
				Request: testReaderFromYAML(t, `cluster:
  initialNodeCount: 1
  name: test-cluster`),
			},
			inputTracker: googlecloudcommon_contract.NewGCPOperationTracker(),
			assertTracker: func(t *testing.T, tracker *googlecloudcommon_contract.GCPOperationTracker) {
				wantManifest := testReaderFromYAML(t, `initialNodeCount: 1
name: test-cluster`).Node
				if diff := cmp.Diff(wantManifest, tracker.CurrentManifest(), compareNodeOption); diff != "" {
					t.Errorf("tracker.CurrentManifest() mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			desc: "cluster create finished with request manifest caches manifest in tracker",
			inputResource: googlecloudloggkeapiaudit_contract.GKEAuditLogResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "",
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "op-1",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.container.v1.ClusterManager.CreateCluster",
				PrincipalEmail: "foobar@qux.test",
				Request: testReaderFromYAML(t, `cluster:
  initialNodeCount: 1
  name: test-cluster`),
			},
			inputTracker: googlecloudcommon_contract.NewGCPOperationTracker(),
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				wantBody := testReaderFromYAML(t, `initialNodeCount: 1
name: test-cluster`).Node
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantClusterPath, &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbCreate,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sClusterExisting,
						Principal:    "foobar@qux.test",
						ChangedTime:  testTime,
						ResourceBody: wantBody,
					}, compareNodeOption).
					HasRevision(wantClusterOpPath, &khifilev6.StagingRevision{
						VerbType:    googlecloudcommon_contract.VerbOperationFinish,
						StateType:   googlecloudcommon_contract.RevisionStateOperationSucceed,
						Principal:   "foobar@qux.test",
						ChangedTime: testTime,
						ResourceBody: testReaderFromYAML(t, `cluster:
  initialNodeCount: 1
  name: test-cluster`).Node,
					}, compareNodeOption)
			},
			assertTracker: func(t *testing.T, tracker *googlecloudcommon_contract.GCPOperationTracker) {
				wantManifest := testReaderFromYAML(t, `initialNodeCount: 1
name: test-cluster`).Node
				if diff := cmp.Diff(wantManifest, tracker.CurrentManifest(), compareNodeOption); diff != "" {
					t.Errorf("tracker.CurrentManifest() mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			desc: "nodepool create started with request manifest caches manifest in tracker",
			inputResource: googlecloudloggkeapiaudit_contract.GKEAuditLogResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "op-2",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "google.container.v1.ClusterManager.CreateNodePool",
				PrincipalEmail: "foobar@qux.test",
				Request: testReaderFromYAML(t, `nodePool:
  initialNodeCount: 2
  name: test-nodepool`),
			},
			inputTracker: googlecloudcommon_contract.NewGCPOperationTracker(),
			assertTracker: func(t *testing.T, tracker *googlecloudcommon_contract.GCPOperationTracker) {
				wantManifest := testReaderFromYAML(t, `initialNodeCount: 2
name: test-nodepool`).Node
				if diff := cmp.Diff(wantManifest, tracker.CurrentManifest(), compareNodeOption); diff != "" {
					t.Errorf("tracker.CurrentManifest() mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			desc: "nodepool create finished with request manifest caches manifest in tracker",
			inputResource: googlecloudloggkeapiaudit_contract.GKEAuditLogResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "op-2",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.container.v1.ClusterManager.CreateNodePool",
				PrincipalEmail: "foobar@qux.test",
				Request: testReaderFromYAML(t, `nodePool:
  initialNodeCount: 2
  name: test-nodepool`),
			},
			inputTracker: googlecloudcommon_contract.NewGCPOperationTracker(),
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				wantBody := testReaderFromYAML(t, `initialNodeCount: 2
name: test-nodepool`).Node
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantNodepoolPath, &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbCreate,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sNodepoolExisting,
						Principal:    "foobar@qux.test",
						ChangedTime:  testTime,
						ResourceBody: wantBody,
					}, compareNodeOption).
					HasRevision(wantNodepoolOp1Path, &khifilev6.StagingRevision{
						VerbType:    googlecloudcommon_contract.VerbOperationFinish,
						StateType:   googlecloudcommon_contract.RevisionStateOperationSucceed,
						Principal:   "foobar@qux.test",
						ChangedTime: testTime,
						ResourceBody: testReaderFromYAML(t, `nodePool:
  initialNodeCount: 2
  name: test-nodepool`).Node,
					}, compareNodeOption)
			},
			assertTracker: func(t *testing.T, tracker *googlecloudcommon_contract.GCPOperationTracker) {
				wantManifest := testReaderFromYAML(t, `initialNodeCount: 2
name: test-nodepool`).Node
				if diff := cmp.Diff(wantManifest, tracker.CurrentManifest(), compareNodeOption); diff != "" {
					t.Errorf("tracker.CurrentManifest() mismatch (-want +got):\n%s", diff)
				}
			},
		},
		{
			desc: "nodepool deletion finished clears manifest in tracker",
			inputResource: googlecloudloggkeapiaudit_contract.GKEAuditLogResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "op-2",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.container.v1.ClusterManager.DeleteNodePool",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
			},
			inputTracker: func() *googlecloudcommon_contract.GCPOperationTracker {
				tr := googlecloudcommon_contract.NewGCPOperationTracker()
				tr.SetCurrentManifest(testReaderFromYAML(t, `name: test-nodepool`).Node)
				return tr
			}(),
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantNodepoolPath, &khifilev6.StagingRevision{
						VerbType:    commonlogk8saudit_contract.VerbDelete,
						StateType:   commonlogk8saudit_contract.RevisionStateK8sNodepoolDeleted,
						Principal:   "foobar@qux.test",
						ChangedTime: testTime,
					}, compareNodeOption).
					HasRevision(wantNodepoolOp2Path, &khifilev6.StagingRevision{
						VerbType:    googlecloudcommon_contract.VerbOperationFinish,
						StateType:   googlecloudcommon_contract.RevisionStateOperationSucceed,
						Principal:   "foobar@qux.test",
						ChangedTime: testTime,
					}, compareNodeOption)
			},
			assertTracker: func(t *testing.T, tracker *googlecloudcommon_contract.GCPOperationTracker) {
				if tracker.CurrentManifest() != nil {
					t.Errorf("tracker.CurrentManifest() got non-nil, want nil after deletion finished")
				}
			},
		},
		{
			desc: "cluster update started stages operation start without resource update",
			inputResource: googlecloudloggkeapiaudit_contract.GKEAuditLogResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "",
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "op-3",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "google.container.v1.ClusterManager.UpdateCluster",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
			},
			inputTracker: googlecloudcommon_contract.NewGCPOperationTracker(),
			initialStateProvider: &mockInitialResourceStateProvider{
				clusterStates: map[string]*googlecloudloggkeapiaudit_contract.InitialResourceState{
					"test-cluster": {
						ResourceBody: testReaderFromYAML(t, "initialNodeCount: 1\nname: test-cluster").Node,
					},
				},
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantClusterOpUpdatePath, &khifilev6.StagingRevision{
						VerbType:    googlecloudcommon_contract.VerbOperationStart,
						StateType:   googlecloudcommon_contract.RevisionStateOperationStarted,
						Principal:   "foobar@qux.test",
						ChangedTime: testTime,
					}, compareNodeOption).
					HasNoRevision(wantClusterPath)
			},
		},
		{
			desc: "nodepool update started stages operation start without resource update",
			inputResource: googlecloudloggkeapiaudit_contract.GKEAuditLogResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "op-3",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "google.container.v1.ClusterManager.UpdateNodePool",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
			},
			inputTracker: googlecloudcommon_contract.NewGCPOperationTracker(),
			initialStateProvider: &mockInitialResourceStateProvider{
				nodePoolStates: map[string]*googlecloudloggkeapiaudit_contract.InitialResourceState{
					"test-cluster/test-nodepool": {
						ResourceBody: testReaderFromYAML(t, "initialNodeCount: 3\nname: test-nodepool").Node,
					},
				},
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantNodepoolOpUpdatePath, &khifilev6.StagingRevision{
						VerbType:    googlecloudcommon_contract.VerbOperationStart,
						StateType:   googlecloudcommon_contract.RevisionStateOperationStarted,
						Principal:   "foobar@qux.test",
						ChangedTime: testTime,
					}, compareNodeOption).
					HasNoRevision(wantNodepoolPath)
			},
		},
		{
			desc: "nodepool update failed does not stage resource update",
			inputResource: googlecloudloggkeapiaudit_contract.GKEAuditLogResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				ProjectID:      "test-project",
				OperationID:    "op-3",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.container.v1.ClusterManager.UpdateNodePool",
				PrincipalEmail: "foobar@qux.test",
				Status:         3,
				Request: testReaderFromYAML(t, `nodePool:
  initialNodeCount: 5`),
			},
			inputTracker: func() *googlecloudcommon_contract.GCPOperationTracker {
				tr := googlecloudcommon_contract.NewGCPOperationTracker()
				tr.MarkStarted("op-3")
				tr.MarkResourceRevision(wantNodepoolPath)
				tr.SetCurrentManifest(testReaderFromYAML(t, "initialNodeCount: 3\nname: test-nodepool").Node)
				return tr
			}(),
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantNodepoolOpUpdatePath, &khifilev6.StagingRevision{
						VerbType:    googlecloudcommon_contract.VerbOperationFinish,
						StateType:   googlecloudcommon_contract.RevisionStateOperationFailed,
						Principal:   "foobar@qux.test",
						ChangedTime: testTime,
						ResourceBody: testReaderFromYAML(t, `nodePool:
  initialNodeCount: 5`).Node,
					}, compareNodeOption).
					HasNoRevision(wantNodepoolPath)
			},
			assertTracker: func(t *testing.T, tracker *googlecloudcommon_contract.GCPOperationTracker) {
				wantManifest := testReaderFromYAML(t, "initialNodeCount: 3\nname: test-nodepool").Node
				if diff := cmp.Diff(wantManifest, tracker.CurrentManifest(), compareNodeOption); diff != "" {
					t.Errorf("tracker.CurrentManifest() mismatch (-want +got):\n%s", diff)
				}
			},
		},
	}

	mapperSetting := &gkeAuditLogLogToTimelineMapperSetting{}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l := testlog.NewMockLog(testTime, tc.inputAudit, tc.inputResource)
			ctx := khictx.WithValue(t.Context(), inspectioncore.Builder, builder)

			provider := tc.initialStateProvider
			if provider == nil {
				provider = &mockInitialResourceStateProvider{}
			}
			ctx = tasktest.WithTaskResult(ctx, googlecloudloggkeapiaudit_contract.InitialResourceStateProviderRef, provider)

			cs, tracker, err := mapperSetting.ProcessLogByGroup(ctx, l, tc.inputTracker)
			if err != nil {
				t.Errorf("ProcessLogByGroup() returned an unexpected error, err=%v", err)
			}

			if tc.assert != nil {
				tc.assert(t, cs)
			}
			if tc.assertTracker != nil {
				tc.assertTracker(t, tracker)
			}
		})
	}
}
