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

package googlecloudlogmulticloudapiaudit_impl

import (
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogmulticloudapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogmulticloudapiaudit/contract"
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

func TestHistoryModifierTask(t *testing.T) {
	testTime := time.Date(2025, time.January, 1, 1, 1, 1, 1, time.UTC)
	testCommonFieldSet := &log.CommonFieldSet{
		Timestamp: testTime,
	}
	testCases := []struct {
		desc          string
		inputResource googlecloudlogmulticloudapiaudit_contract.MulticloudAPIAuditResourceFieldSet
		inputAudit    googlecloudcommon_contract.GCPAuditLogFieldSet
		asserters     []testchangeset.ChangeSetAsserter
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
					ResourcePath: "@Cluster#controlplane#cluster-scope#test-cluster#CreateAwsCluster-op-1",
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
					ResourcePath: "@Cluster#controlplane#cluster-scope#test-cluster#CreateAzureCluster-op-1",
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
					ResourcePath: "@Cluster#nodepool#test-cluster#test-nodepool#CreateAwsNodePool-op-2",
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
					ResourcePath: "@Cluster#nodepool#test-cluster#test-nodepool#CreateAzureNodePool-op-2",
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
					ResourcePath: "@Cluster#nodepool#test-cluster#test-nodepool#DeleteAwsNodePool-op-2",
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
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.MatchResourcePathSet{
					WantResourcePaths: []string{"@Cluster#nodepool#test-cluster#test-nodepool"},
				},
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
			historyModifierSetting := &multicloudAuditLogHistoryModifierSetting{}
			cs := history.NewChangeSet(l)

			_, err := historyModifierSetting.ModifyChangeSetFromLog(t.Context(), l, cs, nil, struct{}{})
			if err != nil {
				t.Errorf("got error %v, want nil", err)
			}

			for _, asserter := range tc.asserters {
				asserter.Assert(t, cs)
			}
		})
	}
}
