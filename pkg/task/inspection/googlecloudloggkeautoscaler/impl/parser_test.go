// Copyright 2025 Google LLC
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

package googlecloudloggkeautoscaler_impl

import (
	"testing"
	"time"

	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudloggkeautoscaler_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudloggkeautoscaler/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
)

func TestHistoryModifierTask(t *testing.T) {
	testTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	testCases := []struct {
		desc     string
		input    *googlecloudloggkeautoscaler_contract.AutoscalerLogFieldSet
		asserter []testchangeset.ChangeSetAsserter
	}{
		{
			desc: "scale up",
			input: &googlecloudloggkeautoscaler_contract.AutoscalerLogFieldSet{
				DecisionLog: &googlecloudloggkeautoscaler_contract.DecisionLog{
					ScaleUp: &googlecloudloggkeautoscaler_contract.ScaleUpItem{
						IncreasedMigs: []googlecloudloggkeautoscaler_contract.IncreasedMIGItem{
							{
								Mig: googlecloudloggkeautoscaler_contract.MIGItem{
									Nodepool: "default-pool",
									Name:     "test-cluster-default-pool-a0c72690-grp",
								},
								RequestedNodes: 1,
							},
						},
						TriggeringPods: []googlecloudloggkeautoscaler_contract.PodItem{
							{
								Name:      "test-85958b848b-ptc7n",
								Namespace: "default",
							},
						},
					},
				},
			},
			asserter: []testchangeset.ChangeSetAsserter{
				&testchangeset.MatchResourcePathSet{
					WantResourcePaths: []string{
						"@Cluster#controlplane#cluster-scope#test-cluster#autoscaler",
						"@Cluster#nodepool#test-cluster#default-pool#test-cluster-default-pool-a0c72690-grp",
						"core/v1#pod#default#test-85958b848b-ptc7n",
					},
				},
				&testchangeset.HasLogSummary{
					WantLogSummary: "Scaling up nodepools by autoscaler: default-pool (requested: 1 in total)",
				},
			},
		},
		{
			desc: "scale down",
			input: &googlecloudloggkeautoscaler_contract.AutoscalerLogFieldSet{
				DecisionLog: &googlecloudloggkeautoscaler_contract.DecisionLog{
					ScaleDown: &googlecloudloggkeautoscaler_contract.ScaleDownItem{
						NodesToBeRemoved: []googlecloudloggkeautoscaler_contract.NodeToBeRemovedItem{
							{
								Node: googlecloudloggkeautoscaler_contract.NodeItem{
									Name: "test-cluster-default-pool-c47ef39f-p395",
									Mig: googlecloudloggkeautoscaler_contract.MIGItem{
										Nodepool: "default-pool",
										Name:     "test-cluster-default-pool-c47ef39f-grp",
									},
								},
								EvictedPods: []googlecloudloggkeautoscaler_contract.PodItem{
									{
										Name:      "kube-dns-5c44c7b6b6-xvpbk",
										Namespace: "kube-system",
									},
								},
							},
						},
					},
				},
			},
			asserter: []testchangeset.ChangeSetAsserter{
				&testchangeset.MatchResourcePathSet{
					WantResourcePaths: []string{
						"@Cluster#controlplane#cluster-scope#test-cluster#autoscaler",
						"core/v1#node#cluster-scope#test-cluster-default-pool-c47ef39f-p395",
						"@Cluster#nodepool#test-cluster#default-pool#test-cluster-default-pool-c47ef39f-grp",
						"core/v1#pod#kube-system#kube-dns-5c44c7b6b6-xvpbk",
					},
				},
				&testchangeset.HasLogSummary{
					WantLogSummary: "Scaling down nodepools by autoscaler: default-pool (Removing 1 nodes in total)",
				},
			},
		},
		{
			desc: "nodepool created",
			input: &googlecloudloggkeautoscaler_contract.AutoscalerLogFieldSet{
				DecisionLog: &googlecloudloggkeautoscaler_contract.DecisionLog{
					NodePoolCreated: &googlecloudloggkeautoscaler_contract.NodepoolCreatedItem{
						NodePools: []googlecloudloggkeautoscaler_contract.NodepoolItem{
							{
								Name: "nap-n1-standard-1-1kwag2qv",
								Migs: []googlecloudloggkeautoscaler_contract.MIGItem{
									{
										Name:     "test-cluster-nap-n1-standard--b4fcc348-grp",
										Nodepool: "nap-n1-standard-1-1kwag2qv",
									},
								},
							},
						},
					},
				},
			},
			asserter: []testchangeset.ChangeSetAsserter{
				&testchangeset.MatchResourcePathSet{
					WantResourcePaths: []string{
						"@Cluster#controlplane#cluster-scope#test-cluster#autoscaler",
						"@Cluster#nodepool#test-cluster#nap-n1-standard-1-1kwag2qv",
						"@Cluster#nodepool#test-cluster#nap-n1-standard-1-1kwag2qv#test-cluster-nap-n1-standard--b4fcc348-grp",
					},
				},
				&testchangeset.HasLogSummary{
					WantLogSummary: "Nodepool created by node auto provisioner: nap-n1-standard-1-1kwag2qv",
				},
			},
		},
		{
			desc: "nodepool deleted",
			input: &googlecloudloggkeautoscaler_contract.AutoscalerLogFieldSet{
				DecisionLog: &googlecloudloggkeautoscaler_contract.DecisionLog{
					NodePoolDeleted: &googlecloudloggkeautoscaler_contract.NodepoolDeletedItem{
						NodePoolNames: []string{
							"nap-n1-highcpu-8-ydj4ewil",
						},
					},
				},
			},
			asserter: []testchangeset.ChangeSetAsserter{
				&testchangeset.MatchResourcePathSet{
					WantResourcePaths: []string{
						"@Cluster#controlplane#cluster-scope#test-cluster#autoscaler",
						"@Cluster#nodepool#test-cluster#nap-n1-highcpu-8-ydj4ewil",
					},
				},
				&testchangeset.HasLogSummary{
					WantLogSummary: "Nodepool deleted by node auto provisioner: nap-n1-highcpu-8-ydj4ewil",
				},
			},
		},
		{
			desc: "no scale up",
			input: &googlecloudloggkeautoscaler_contract.AutoscalerLogFieldSet{
				NoDecisionLog: &googlecloudloggkeautoscaler_contract.NoDecisionStatusLog{
					NoScaleUp: &googlecloudloggkeautoscaler_contract.NoScaleUpItem{
						SkippedMigs: []googlecloudloggkeautoscaler_contract.SkippedMIGItem{
							{
								Mig: googlecloudloggkeautoscaler_contract.MIGItem{
									Nodepool: "nap-n1-highmem-4-1cywzhvf",
									Name:     "test-cluster-nap-n1-highmem-4-fbdca585-grp",
								},
							},
						},
						UnhandledPodGroups: []googlecloudloggkeautoscaler_contract.UnhandledPodGroupItem{
							{
								PodGroup: googlecloudloggkeautoscaler_contract.PodGroup{
									SamplePod: googlecloudloggkeautoscaler_contract.PodItem{
										Name:      "memory-reservation2-6zg8m",
										Namespace: "autoscaling-1661",
									},
								},
								RejectedMigs: []googlecloudloggkeautoscaler_contract.RejectedMIGItem{
									{
										Mig: googlecloudloggkeautoscaler_contract.MIGItem{
											Nodepool: "default-pool",
											Name:     "test-cluster-default-pool-b1808ff9-grp",
										},
									},
								},
							},
						},
					},
				},
			},
			asserter: []testchangeset.ChangeSetAsserter{
				&testchangeset.MatchResourcePathSet{
					WantResourcePaths: []string{
						"@Cluster#controlplane#cluster-scope#test-cluster#autoscaler",
						"@Cluster#nodepool#test-cluster#default-pool#test-cluster-default-pool-b1808ff9-grp",
						"@Cluster#nodepool#test-cluster#nap-n1-highmem-4-1cywzhvf#test-cluster-nap-n1-highmem-4-fbdca585-grp",
						"core/v1#pod#autoscaling-1661#memory-reservation2-6zg8m",
					},
				},
				&testchangeset.HasLogSummary{
					WantLogSummary: "autoscaler decided not to scale up",
				},
			},
		},
		{
			desc: "no scale down",
			input: &googlecloudloggkeautoscaler_contract.AutoscalerLogFieldSet{
				NoDecisionLog: &googlecloudloggkeautoscaler_contract.NoDecisionStatusLog{
					NoScaleDown: &googlecloudloggkeautoscaler_contract.NoScaleDownItem{
						Nodes: []googlecloudloggkeautoscaler_contract.NoScaleDownNodeItem{
							{
								Node: googlecloudloggkeautoscaler_contract.NodeItem{
									Name: "test-cluster-default-pool-f74c1617-fbhk",
									Mig: googlecloudloggkeautoscaler_contract.MIGItem{
										Nodepool: "default-pool",
										Name:     "test-cluster-default-pool-f74c1617-grp",
									},
								},
							},
						},
						Reason: googlecloudloggkeautoscaler_contract.ReasonItem{
							MessageId:  "no.scale.down.in.backoff",
							Parameters: []string{"param1", "param2"},
						},
					},
				},
			},
			asserter: []testchangeset.ChangeSetAsserter{
				&testchangeset.MatchResourcePathSet{
					WantResourcePaths: []string{
						"@Cluster#controlplane#cluster-scope#test-cluster#autoscaler",
						"core/v1#node#cluster-scope#test-cluster-default-pool-f74c1617-fbhk",
						"@Cluster#nodepool#test-cluster#default-pool#test-cluster-default-pool-f74c1617-grp",
					},
				},
				&testchangeset.HasLogSummary{
					WantLogSummary: "autoscaler decided not to scale down: no.scale.down.in.backoff(param1,param2)",
				},
			},
		},
		{
			desc: "no scale down wihout param",
			input: &googlecloudloggkeautoscaler_contract.AutoscalerLogFieldSet{
				NoDecisionLog: &googlecloudloggkeautoscaler_contract.NoDecisionStatusLog{
					NoScaleDown: &googlecloudloggkeautoscaler_contract.NoScaleDownItem{
						Nodes: []googlecloudloggkeautoscaler_contract.NoScaleDownNodeItem{
							{
								Node: googlecloudloggkeautoscaler_contract.NodeItem{
									Name: "test-cluster-default-pool-f74c1617-fbhk",
									Mig: googlecloudloggkeautoscaler_contract.MIGItem{
										Nodepool: "default-pool",
										Name:     "test-cluster-default-pool-f74c1617-grp",
									},
								},
							},
						},
						Reason: googlecloudloggkeautoscaler_contract.ReasonItem{
							MessageId: "no.scale.down.in.backoff",
						},
					},
				},
			},
			asserter: []testchangeset.ChangeSetAsserter{
				&testchangeset.MatchResourcePathSet{
					WantResourcePaths: []string{
						"@Cluster#controlplane#cluster-scope#test-cluster#autoscaler",
						"core/v1#node#cluster-scope#test-cluster-default-pool-f74c1617-fbhk",
						"@Cluster#nodepool#test-cluster#default-pool#test-cluster-default-pool-f74c1617-grp",
					},
				},
				&testchangeset.HasLogSummary{
					WantLogSummary: "autoscaler decided not to scale down: no.scale.down.in.backoff",
				},
			},
		},
		{
			desc: "result info success",
			input: &googlecloudloggkeautoscaler_contract.AutoscalerLogFieldSet{
				ResultInfoLog: &googlecloudloggkeautoscaler_contract.ResultInfoLog{
					Results: []googlecloudloggkeautoscaler_contract.Result{
						{
							EventID: "2fca91cd-7345-47fc-9770-838e05e28b17",
						},
					},
				},
			},
			asserter: []testchangeset.ChangeSetAsserter{
				&testchangeset.MatchResourcePathSet{
					WantResourcePaths: []string{
						"@Cluster#controlplane#cluster-scope#test-cluster#autoscaler",
					},
				},
				&testchangeset.HasLogSummary{
					WantLogSummary: "autoscaler finished events: 2fca91cd-7345-47fc-9770-838e05e28b17(Success)",
				},
				&testchangeset.HasRevision{
					ResourcePath: "@Cluster#controlplane#cluster-scope#test-cluster#autoscaler",
					WantRevision: history.StagingResourceRevision{
						ChangeTime: testTime,
						Requestor:  "cluster-autoscaler",
						State:      enum.RevisionAutoscalerNoError,
						Body: `measureTime: ""
results:
    - eventId: 2fca91cd-7345-47fc-9770-838e05e28b17
`,
					},
				},
			},
		},
		{
			desc: "result info error",
			input: &googlecloudloggkeautoscaler_contract.AutoscalerLogFieldSet{
				ResultInfoLog: &googlecloudloggkeautoscaler_contract.ResultInfoLog{
					Results: []googlecloudloggkeautoscaler_contract.Result{
						{
							EventID: "ea2e964c-49b8-4cd7-8fa9-fefb0827f9a6",
							ErrorMsg: &googlecloudloggkeautoscaler_contract.ErrorMessageItem{
								MessageId:  "scale.down.error.failed.to.delete.node.min.size.reached",
								Parameters: []string{"test-cluster-default-pool-5c90f485-nk80"},
							},
						},
					},
				},
			},
			asserter: []testchangeset.ChangeSetAsserter{
				&testchangeset.MatchResourcePathSet{
					WantResourcePaths: []string{
						"@Cluster#controlplane#cluster-scope#test-cluster#autoscaler",
					},
				},
				&testchangeset.HasLogSummary{
					WantLogSummary: "autoscaler finished events: ea2e964c-49b8-4cd7-8fa9-fefb0827f9a6(Error:scale.down.error.failed.to.delete.node.min.size.reached(test-cluster-default-pool-5c90f485-nk80))",
				},
				&testchangeset.HasRevision{
					ResourcePath: "@Cluster#controlplane#cluster-scope#test-cluster#autoscaler",
					WantRevision: history.StagingResourceRevision{
						ChangeTime: testTime,
						Requestor:  "cluster-autoscaler",
						State:      enum.RevisionAutoscalerHasErrors,
						Body: `measureTime: ""
results:
    - eventId: ea2e964c-49b8-4cd7-8fa9-fefb0827f9a6
      errorMsg:
        messageId: scale.down.error.failed.to.delete.node.min.size.reached
        parameters:
            - test-cluster-default-pool-5c90f485-nk80
`,
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l := log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{Timestamp: testTime},
				tc.input,
			)
			cs := history.NewChangeSet(l)
			ctx := tasktest.WithTaskResult(t.Context(), googlecloudk8scommon_contract.InputClusterNameTaskID.Ref(), "test-cluster")
			_, err := (&autoscalerHistoryModifierTaskSetting{}).ModifyChangeSetFromLog(ctx, l, cs, nil, struct{}{})
			if err != nil {
				t.Fatalf("ModifyChangeSetFromLog() error = %v", err)
			}
			for _, asserter := range tc.asserter {
				asserter.Assert(t, cs)
			}
		})

	}
}
