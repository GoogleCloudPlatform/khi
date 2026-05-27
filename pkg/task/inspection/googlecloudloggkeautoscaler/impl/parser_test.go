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
	"fmt"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudloggkeautoscaler_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudloggkeautoscaler/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/google/go-cmp/cmp"
)

func TestAutoscalerLogIngester_ProcessLog(t *testing.T) {
	testTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	testCases := []struct {
		name         string
		input        *googlecloudloggkeautoscaler_contract.AutoscalerLogFieldSet
		wantSummary  string
		wantSeverity *pb.Severity
	}{
		{
			name: "scale up",
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
					},
				},
			},
			wantSummary:  "Scaling up nodepools by autoscaler: default-pool (requested: 1 in total)",
			wantSeverity: inspectioncore_contract.SeverityWarning,
		},
		{
			name: "scale down",
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
							},
						},
					},
				},
			},
			wantSummary:  "Scaling down nodepools by autoscaler: default-pool (Removing 1 nodes in total)",
			wantSeverity: inspectioncore_contract.SeverityWarning,
		},
		{
			name: "nodepool created",
			input: &googlecloudloggkeautoscaler_contract.AutoscalerLogFieldSet{
				DecisionLog: &googlecloudloggkeautoscaler_contract.DecisionLog{
					NodePoolCreated: &googlecloudloggkeautoscaler_contract.NodepoolCreatedItem{
						NodePools: []googlecloudloggkeautoscaler_contract.NodepoolItem{
							{
								Name: "nap-n1-standard-1-1kwag2qv",
							},
						},
					},
				},
			},
			wantSummary:  "Nodepool created by node auto provisioner: nap-n1-standard-1-1kwag2qv",
			wantSeverity: inspectioncore_contract.SeverityWarning,
		},
		{
			name: "nodepool deleted",
			input: &googlecloudloggkeautoscaler_contract.AutoscalerLogFieldSet{
				DecisionLog: &googlecloudloggkeautoscaler_contract.DecisionLog{
					NodePoolDeleted: &googlecloudloggkeautoscaler_contract.NodepoolDeletedItem{
						NodePoolNames: []string{
							"nap-n1-highcpu-8-ydj4ewil",
						},
					},
				},
			},
			wantSummary:  "Nodepool deleted by node auto provisioner: nap-n1-highcpu-8-ydj4ewil",
			wantSeverity: inspectioncore_contract.SeverityWarning,
		},
		{
			name: "no scale up",
			input: &googlecloudloggkeautoscaler_contract.AutoscalerLogFieldSet{
				NoDecisionLog: &googlecloudloggkeautoscaler_contract.NoDecisionStatusLog{
					NoScaleUp: &googlecloudloggkeautoscaler_contract.NoScaleUpItem{},
				},
			},
			wantSummary:  "autoscaler decided not to scale up",
			wantSeverity: inspectioncore_contract.SeverityInfo,
		},
		{
			name: "no scale down with param",
			input: &googlecloudloggkeautoscaler_contract.AutoscalerLogFieldSet{
				NoDecisionLog: &googlecloudloggkeautoscaler_contract.NoDecisionStatusLog{
					NoScaleDown: &googlecloudloggkeautoscaler_contract.NoScaleDownItem{
						Reason: googlecloudloggkeautoscaler_contract.ReasonItem{
							MessageId:  "no.scale.down.in.backoff",
							Parameters: []string{"param1", "param2"},
						},
					},
				},
			},
			wantSummary:  "autoscaler decided not to scale down: no.scale.down.in.backoff(param1,param2)",
			wantSeverity: inspectioncore_contract.SeverityInfo,
		},
		{
			name: "no scale down without param",
			input: &googlecloudloggkeautoscaler_contract.AutoscalerLogFieldSet{
				NoDecisionLog: &googlecloudloggkeautoscaler_contract.NoDecisionStatusLog{
					NoScaleDown: &googlecloudloggkeautoscaler_contract.NoScaleDownItem{
						Reason: googlecloudloggkeautoscaler_contract.ReasonItem{
							MessageId: "no.scale.down.in.backoff",
						},
					},
				},
			},
			wantSummary:  "autoscaler decided not to scale down: no.scale.down.in.backoff",
			wantSeverity: inspectioncore_contract.SeverityInfo,
		},
		{
			name: "result info success",
			input: &googlecloudloggkeautoscaler_contract.AutoscalerLogFieldSet{
				ResultInfoLog: &googlecloudloggkeautoscaler_contract.ResultInfoLog{
					Results: []googlecloudloggkeautoscaler_contract.Result{
						{
							EventID: "2fca91cd-7345-47fc-9770-838e05e28b17",
						},
					},
				},
			},
			wantSummary:  "autoscaler finished events: 2fca91cd-7345-47fc-9770-838e05e28b17(Success)",
			wantSeverity: inspectioncore_contract.SeverityInfo,
		},
		{
			name: "result info error",
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
			wantSummary:  "autoscaler finished events: ea2e964c-49b8-4cd7-8fa9-fefb0827f9a6(Error:scale.down.error.failed.to.delete.node.min.size.reached(test-cluster-default-pool-5c90f485-nk80))",
			wantSeverity: inspectioncore_contract.SeverityInfo,
		},
	}

	ingester := &autoscalerLogIngester{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			l := log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{Timestamp: testTime},
				tc.input,
			)
			cs, err := ingester.ProcessLog(t.Context(), l)
			if err != nil {
				t.Fatalf("ProcessLog() error = %v", err)
			}

			testchangeset.AssertLog(t, cs).
				HasSummary(tc.wantSummary).
				HasSeverity(tc.wantSeverity).
				HasLogType(googlecloudloggkeautoscaler_contract.LogTypeAutoscaler).
				HasTimestamp(testTime)
		})
	}
}

var nodeCmpOpt = cmp.Transformer("StructuredNodeToJSON", func(n structured.Node) string {
	if n == nil {
		return "nil"
	}
	serializer := &structured.JSONNodeSerializer{}
	bytes, err := serializer.Serialize(n)
	if err != nil {
		return fmt.Sprintf("error serializing structured node: %v", err)
	}
	return string(bytes)
})

func TestAutoscalerTimelineMapper_ProcessLogByGroup(t *testing.T) {
	testTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// 1. Initialize the Builder first.
	builder := khifilev6.NewBuilder()

	// 2. Resolve comparative path instances using the Builder's accumulator.
	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

	clusterTimeline := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, "test-cluster")
	autoscalerPath := googlecloudloggkeautoscaler_contract.MustAutoscalerTimeline(ctx, clusterTimeline)

	gkeClusterTimeline := googlecloudcommon_contract.MustGKEClusterTimeline(ctx, "test-cluster")
	nodepoolTimeline := googlecloudcommon_contract.MustGKENodePoolTimeline(ctx, gkeClusterTimeline, "default-pool")
	migPath := googlecloudloggkeautoscaler_contract.MustMigTimeline(ctx, nodepoolTimeline, "test-cluster-default-pool-a0c72690-grp")

	apiVersionTimeline := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, clusterTimeline, "core/v1")
	kindTimeline := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiVersionTimeline, "pod")
	namespaceTimeline := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindTimeline, "default")
	podPath := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, namespaceTimeline, "test-85958b848b-ptc7n")

	// Additional paths for other test cases
	scaleDownMigPath := googlecloudloggkeautoscaler_contract.MustMigTimeline(ctx, nodepoolTimeline, "test-cluster-default-pool-c47ef39f-grp")

	kubeDnsNamespaceTimeline := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindTimeline, "kube-system")
	kubeDnsPodPath := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, kubeDnsNamespaceTimeline, "kube-dns-5c44c7b6b6-xvpbk")

	nodeKindTimeline := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, apiVersionTimeline, "node")
	nodePath := commonlogk8saudit_contract.MustK8sClusterScopeResourceTimeline(ctx, nodeKindTimeline, "test-cluster-default-pool-c47ef39f-p395")

	// Node auto provisioning creation
	napNodepoolTimeline := googlecloudcommon_contract.MustGKENodePoolTimeline(ctx, gkeClusterTimeline, "nap-n1-standard-1-1kwag2qv")
	napMigPath := googlecloudloggkeautoscaler_contract.MustMigTimeline(ctx, napNodepoolTimeline, "test-cluster-nap-n1-standard--b4fcc348-grp")

	// Node auto provisioning deletion
	napDeletedNodepoolTimeline := googlecloudcommon_contract.MustGKENodePoolTimeline(ctx, gkeClusterTimeline, "nap-n1-highcpu-8-ydj4ewil")

	// No scale up
	skippedNodepoolTimeline := googlecloudcommon_contract.MustGKENodePoolTimeline(ctx, gkeClusterTimeline, "nap-n1-highmem-4-1cywzhvf")
	skippedMigPath := googlecloudloggkeautoscaler_contract.MustMigTimeline(ctx, skippedNodepoolTimeline, "test-cluster-nap-n1-highmem-4-fbdca585-grp")

	unhandledPodNamespaceTimeline := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, kindTimeline, "autoscaling-1661")
	unhandledPodPath := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, unhandledPodNamespaceTimeline, "memory-reservation2-6zg8m")

	rejectedMigPath := googlecloudloggkeautoscaler_contract.MustMigTimeline(ctx, nodepoolTimeline, "test-cluster-default-pool-b1808ff9-grp")

	// No scale down
	noScaleDownNodePath := commonlogk8saudit_contract.MustK8sClusterScopeResourceTimeline(ctx, nodeKindTimeline, "test-cluster-default-pool-f74c1617-fbhk")
	noScaleDownMigPath := googlecloudloggkeautoscaler_contract.MustMigTimeline(ctx, nodepoolTimeline, "test-cluster-default-pool-f74c1617-grp")

	testCases := []struct {
		name   string
		input  *googlecloudloggkeautoscaler_contract.AutoscalerLogFieldSet
		assert func(t *testing.T, cs *khifilev6.TimelineChangeSet)
	}{
		{
			name: "scale up",
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
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(autoscalerPath).
					HasEvent(migPath).
					HasEvent(podPath)
			},
		},
		{
			name: "scale down",
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
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(autoscalerPath).
					HasEvent(nodePath).
					HasEvent(scaleDownMigPath).
					HasEvent(kubeDnsPodPath)
			},
		},
		{
			name: "nodepool created",
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
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(autoscalerPath).
					HasEvent(napNodepoolTimeline).
					HasEvent(napMigPath)
			},
		},
		{
			name: "nodepool deleted",
			input: &googlecloudloggkeautoscaler_contract.AutoscalerLogFieldSet{
				DecisionLog: &googlecloudloggkeautoscaler_contract.DecisionLog{
					NodePoolDeleted: &googlecloudloggkeautoscaler_contract.NodepoolDeletedItem{
						NodePoolNames: []string{
							"nap-n1-highcpu-8-ydj4ewil",
						},
					},
				},
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(autoscalerPath).
					HasEvent(napDeletedNodepoolTimeline)
			},
		},
		{
			name: "no scale up",
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
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(autoscalerPath).
					HasEvent(skippedMigPath).
					HasEvent(unhandledPodPath).
					HasEvent(rejectedMigPath)
			},
		},
		{
			name: "no scale down",
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
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(autoscalerPath).
					HasEvent(noScaleDownNodePath).
					HasEvent(noScaleDownMigPath)
			},
		},
		{
			name: "result info success",
			input: &googlecloudloggkeautoscaler_contract.AutoscalerLogFieldSet{
				ResultInfoLog: &googlecloudloggkeautoscaler_contract.ResultInfoLog{
					Results: []googlecloudloggkeautoscaler_contract.Result{
						{
							EventID: "2fca91cd-7345-47fc-9770-838e05e28b17",
						},
					},
				},
			},
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				bodyYAML := `measureTime: ""
results:
    - eventId: 2fca91cd-7345-47fc-9770-838e05e28b17
`
				bodyNode, err := structured.FromYAML(bodyYAML)
				if err != nil {
					t.Fatalf("failed to parse body YAML: %v", err)
				}
				testchangeset.AssertTimeline(t, cs).
					HasRevision(autoscalerPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						Principal:    "cluster-autoscaler",
						StateType:    googlecloudloggkeautoscaler_contract.RevisionAutoscalerNoError,
						ResourceBody: bodyNode,
					}, nodeCmpOpt)
			},
		},
		{
			name: "result info error",
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
			assert: func(t *testing.T, cs *khifilev6.TimelineChangeSet) {
				bodyYAML := `measureTime: ""
results:
    - eventId: ea2e964c-49b8-4cd7-8fa9-fefb0827f9a6
      errorMsg:
        messageId: scale.down.error.failed.to.delete.node.min.size.reached
        parameters:
            - test-cluster-default-pool-5c90f485-nk80
`
				bodyNode, err := structured.FromYAML(bodyYAML)
				if err != nil {
					t.Fatalf("failed to parse body YAML: %v", err)
				}
				testchangeset.AssertTimeline(t, cs).
					HasRevision(autoscalerPath, &khifilev6.StagingRevision{
						ChangedTime:  testTime,
						Principal:    "cluster-autoscaler",
						StateType:    googlecloudloggkeautoscaler_contract.RevisionAutoscalerHasErrors,
						ResourceBody: bodyNode,
					}, nodeCmpOpt)
			},
		},
	}

	mapper := &autoscalerTimelineMapper{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.input.ClusterName = "test-cluster"
			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

			l := log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{Timestamp: testTime},
				tc.input,
			)

			cs, _, err := mapper.ProcessLogByGroup(ctx, l, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLogByGroup() error = %v", err)
			}

			tc.assert(t, cs)
		})
	}
}
