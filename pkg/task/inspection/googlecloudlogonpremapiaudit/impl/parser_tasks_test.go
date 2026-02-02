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

package googlecloudlogonpremapiaudit_impl

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogonpremapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogonpremapiaudit/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
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
	testTime := time.Date(2025, time.January, 1, 1, 1, 1, 1, time.UTC)
	testCommonFieldSet := &log.CommonFieldSet{
		Timestamp: testTime,
	}
	testCases := []struct {
		desc          string
		inputResource googlecloudlogonpremapiaudit_contract.OnPremAPIAuditResourceFieldSet
		inputAudit    googlecloudcommon_contract.GCPAuditLogFieldSet
		asserters     []testchangeset.ChangeSetAsserter
	}{
		{
			desc: "cluster create started",
			inputResource: googlecloudlogonpremapiaudit_contract.OnPremAPIAuditResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "",
				ClusterType:  googlecloudlogonpremapiaudit_contract.ClusterTypeBaremetalAdmin,
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				OperationID:    "op-1",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "google.cloud.gkeonprem.v1.GkeOnPrem.CreateBaremetalAdminCluster",
				PrincipalEmail: "foobar@qux.test",
				Request: testReaderFromYAML(t, `cluster:
  initialNodeCount: 1
  name: test-cluster`),
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "@Cluster#controlplane#cluster-scope#test-cluster",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbCreate,
						State:      enum.RevisionStateProvisioning,
						Requestor:  "foobar@qux.test",
						Body:       "initialNodeCount: 1\nname: test-cluster\n",
						ChangeTime: testTime,
					},
				},
				&testchangeset.HasRevision{
					ResourcePath: "@Cluster#controlplane#cluster-scope#test-cluster#CreateBaremetalAdminCluster-op-1",
					WantRevision: history.StagingResourceRevision{
						Verb:      enum.RevisionVerbOperationStart,
						State:     enum.RevisionStateOperationStarted,
						Requestor: "foobar@qux.test",
						Body: `cluster:
  initialNodeCount: 1
  name: test-cluster
`,
						ChangeTime: testTime,
					},
				},
			},
		},
		{
			desc: "cluster create finished",
			inputResource: googlecloudlogonpremapiaudit_contract.OnPremAPIAuditResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "",
				ClusterType:  googlecloudlogonpremapiaudit_contract.ClusterTypeBaremetalStandalone,
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				OperationID:    "op-1",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.cloud.gkeonprem.v1.GkeOnPrem.CreateBaremetalStandaloneCluster",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "@Cluster#controlplane#cluster-scope#test-cluster",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbCreate,
						State:      enum.RevisionStateExisting,
						Requestor:  "foobar@qux.test",
						Body:       "",
						ChangeTime: testTime,
					},
				},
				&testchangeset.HasRevision{
					ResourcePath: "@Cluster#controlplane#cluster-scope#test-cluster#CreateBaremetalStandaloneCluster-op-1",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbOperationFinish,
						State:      enum.RevisionStateOperationFinished,
						Requestor:  "foobar@qux.test",
						Body:       "",
						ChangeTime: testTime,
					},
				},
			},
		},
		{
			desc: "cluster create finished",
			inputResource: googlecloudlogonpremapiaudit_contract.OnPremAPIAuditResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "",
				ClusterType:  googlecloudlogonpremapiaudit_contract.ClusterTypeBaremetalStandalone,
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				OperationID:    "op-1",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.cloud.gkeonprem.v1.GkeOnPrem.EnrollBaremetalStandaloneCluster",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "@Cluster#controlplane#cluster-scope#test-cluster",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbCreate,
						State:      enum.RevisionStateExisting,
						Requestor:  "foobar@qux.test",
						Body:       "",
						ChangeTime: testTime,
					},
				},
				&testchangeset.HasRevision{
					ResourcePath: "@Cluster#controlplane#cluster-scope#test-cluster#EnrollBaremetalStandaloneCluster-op-1",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbOperationFinish,
						State:      enum.RevisionStateOperationFinished,
						Requestor:  "foobar@qux.test",
						Body:       "",
						ChangeTime: testTime,
					},
				},
			},
		},
		{
			desc: "nodepool create started",
			inputResource: googlecloudlogonpremapiaudit_contract.OnPremAPIAuditResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
				ClusterType:  googlecloudlogonpremapiaudit_contract.ClusterTypeBaremetalUser,
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				OperationID:    "op-2",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "google.cloud.gkeonprem.v1.GkeOnPrem.CreateBaremetalNodePool",
				PrincipalEmail: "foobar@qux.test",
				Request: testReaderFromYAML(t, `nodePool:
  initialNodeCount: 1
  name: test-nodepool`),
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "@Cluster#nodepool#test-cluster#test-nodepool",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbCreate,
						State:      enum.RevisionStateProvisioning,
						Requestor:  "foobar@qux.test",
						Body:       "initialNodeCount: 1\nname: test-nodepool\n",
						ChangeTime: testTime,
					},
				},
				&testchangeset.HasRevision{
					ResourcePath: "@Cluster#nodepool#test-cluster#test-nodepool#CreateBaremetalNodePool-op-2",
					WantRevision: history.StagingResourceRevision{
						Verb:      enum.RevisionVerbOperationStart,
						State:     enum.RevisionStateOperationStarted,
						Requestor: "foobar@qux.test",
						Body: `nodePool:
  initialNodeCount: 1
  name: test-nodepool
`,
						ChangeTime: testTime,
					},
				},
			},
		},
		{
			desc: "nodepool create finished",
			inputResource: googlecloudlogonpremapiaudit_contract.OnPremAPIAuditResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
				ClusterType:  googlecloudlogonpremapiaudit_contract.ClusterTypeVMWareAdmin,
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				OperationID:    "op-2",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.cloud.gkeonprem.v1.GkeOnPrem.CreateVmwareAdminNodePool",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "@Cluster#nodepool#test-cluster#test-nodepool",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbCreate,
						State:      enum.RevisionStateExisting,
						Requestor:  "foobar@qux.test",
						Body:       "",
						ChangeTime: testTime,
					},
				},
				&testchangeset.HasRevision{
					ResourcePath: "@Cluster#nodepool#test-cluster#test-nodepool#CreateVmwareAdminNodePool-op-2",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbOperationFinish,
						State:      enum.RevisionStateOperationFinished,
						Requestor:  "foobar@qux.test",
						Body:       "",
						ChangeTime: testTime,
					},
				},
			},
		},
		{
			desc: "nodepool deletion finished",
			inputResource: googlecloudlogonpremapiaudit_contract.OnPremAPIAuditResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
				ClusterType:  googlecloudlogonpremapiaudit_contract.ClusterTypeVMWareUser,
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				OperationID:    "op-2",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.cloud.gkeonprem.v1.GkeOnPrem.DeleteVmwareNodePool",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "@Cluster#nodepool#test-cluster#test-nodepool",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbDelete,
						State:      enum.RevisionStateDeleted,
						Requestor:  "foobar@qux.test",
						Body:       "",
						ChangeTime: testTime,
					},
				},
				&testchangeset.HasRevision{
					ResourcePath: "@Cluster#nodepool#test-cluster#test-nodepool#DeleteVmwareNodePool-op-2",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbOperationFinish,
						State:      enum.RevisionStateOperationFinished,
						Requestor:  "foobar@qux.test",
						Body:       "",
						ChangeTime: testTime,
					},
				},
			},
		},
		{
			desc: "nodepool deletion finished",
			inputResource: googlecloudlogonpremapiaudit_contract.OnPremAPIAuditResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
				ClusterType:  googlecloudlogonpremapiaudit_contract.ClusterTypeVMWareUser,
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				OperationID:    "op-2",
				OperationFirst: false,
				OperationLast:  true,
				MethodName:     "google.cloud.gkeonprem.v1.GkeOnPrem.UnenrollVmwareNodePool",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "@Cluster#nodepool#test-cluster#test-nodepool",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbDelete,
						State:      enum.RevisionStateDeleted,
						Requestor:  "foobar@qux.test",
						Body:       "",
						ChangeTime: testTime,
					},
				},
				&testchangeset.HasRevision{
					ResourcePath: "@Cluster#nodepool#test-cluster#test-nodepool#UnenrollVmwareNodePool-op-2",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbOperationFinish,
						State:      enum.RevisionStateOperationFinished,
						Requestor:  "foobar@qux.test",
						Body:       "",
						ChangeTime: testTime,
					},
				},
			},
		},
		{
			desc: "immediate action",
			inputResource: googlecloudlogonpremapiaudit_contract.OnPremAPIAuditResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
				ClusterType:  googlecloudlogonpremapiaudit_contract.ClusterTypeVMWareUser,
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				OperationID:    "op-2",
				OperationFirst: true,
				OperationLast:  true,
				MethodName:     "google.cloud.gkeonprem.v1.GkeOnPrem.UpdateVmwareCluster",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.MatchResourcePathSet{
					WantResourcePaths: []string{"@Cluster#nodepool#test-cluster#test-nodepool"},
				},
			},
		},
		{
			desc: "long running operation for unknown cluster",
			inputResource: googlecloudlogonpremapiaudit_contract.OnPremAPIAuditResourceFieldSet{
				ClusterName:  "test-cluster",
				NodepoolName: "test-nodepool",
				ClusterType:  googlecloudlogonpremapiaudit_contract.ClusterTypeUnknown,
			},
			inputAudit: googlecloudcommon_contract.GCPAuditLogFieldSet{
				OperationID:    "op-2",
				OperationFirst: true,
				OperationLast:  false,
				MethodName:     "google.cloud.gkeonprem.v1.GkeOnPrem.UnknownLongRunningOperation",
				PrincipalEmail: "foobar@qux.test",
				Request:        nil,
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "@Cluster#nodepool#test-cluster#test-nodepool#UnknownLongRunningOperation-op-2",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbOperationStart,
						State:      enum.RevisionStateOperationStarted,
						Requestor:  "foobar@qux.test",
						Body:       "",
						ChangeTime: testTime,
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l := log.NewLogWithFieldSetsForTest(testCommonFieldSet, &tc.inputAudit, &tc.inputResource)
			mapperSetting := &onpremAuditLogLogToTimelineMapperSetting{}
			cs := history.NewChangeSet(l)

			_, err := mapperSetting.ProcessLogByGroup(t.Context(), l, cs, nil, struct{}{})
			if err != nil {
				t.Errorf("got error %v, want nil", err)
			}

			for _, asserter := range tc.asserters {
				asserter.Assert(t, cs)
			}
		})
	}
}
