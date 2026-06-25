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
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudloggkeapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudloggkeapiaudit/contract"
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
	builder := khifilev6.NewBuilder()

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

	testTime := time.Date(2025, time.January, 1, 1, 1, 1, 1, time.UTC)
	testCommonFieldSet := &log.CommonFieldSet{
		Timestamp: testTime,
	}

	testCases := []struct {
		desc          string
		inputResource googlecloudloggkeapiaudit_contract.GKEAuditLogResourceFieldSet
		inputAudit    googlecloudcommon_contract.GCPAuditLogFieldSet
		inputTracker  *googlecloudcommon_contract.GCPOperationTracker
		assert        func(t *testing.T, cs *khifilev6.TimelineChangeSet)
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
  name: test-cluster`).GetReader("cluster"); err == nil {
					bodyNode = subReader.Node
				}

				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantClusterPath, &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbCreate,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sClusterProvisioning,
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
						VerbType:    commonlogk8saudit_contract.VerbCreate,
						StateType:   commonlogk8saudit_contract.RevisionStateK8sClusterExisting,
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
  name: test-nodepool`).GetReader("nodePool"); err == nil {
					bodyNode = subReader.Node
				}

				testchangeset.AssertTimeline(t, cs).
					HasRevision(wantNodepoolPath, &khifilev6.StagingRevision{
						VerbType:     commonlogk8saudit_contract.VerbCreate,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sNodepoolProvisioning,
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
						VerbType:    commonlogk8saudit_contract.VerbCreate,
						StateType:   commonlogk8saudit_contract.RevisionStateK8sNodepoolProvisioningLogNotFound,
						Principal:   "foobar@qux.test",
						ChangedTime: time.Unix(0, 0),
					}, compareNodeOption).
					HasRevision(wantNodepoolPath, &khifilev6.StagingRevision{
						VerbType:    commonlogk8saudit_contract.VerbCreate,
						StateType:   commonlogk8saudit_contract.RevisionStateK8sNodepoolExisting,
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
						VerbType:    commonlogk8saudit_contract.VerbDelete,
						StateType:   commonlogk8saudit_contract.RevisionStateK8sNodepoolDeletingLogNotFound,
						Principal:   "foobar@qux.test",
						ChangedTime: time.Unix(0, 0),
					}, compareNodeOption).
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
	}

	mapperSetting := &gkeAuditLogLogToTimelineMapperSetting{}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l := log.NewLogWithFieldSetsForTest(testCommonFieldSet, &tc.inputAudit, &tc.inputResource)
			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

			cs, _, err := mapperSetting.ProcessLogByGroup(ctx, l, tc.inputTracker)
			if err != nil {
				t.Errorf("ProcessLogByGroup() returned an unexpected error, err=%v", err)
			}

			tc.assert(t, cs)
		})
	}
}
