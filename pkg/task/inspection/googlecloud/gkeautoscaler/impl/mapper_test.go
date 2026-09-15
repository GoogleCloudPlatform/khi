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

package gkeautoscaler_impl

import (
	"fmt"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/model/id"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gkeautoscaler"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
	"github.com/google/go-cmp/cmp"
)

func TestAutoscalerLogIngester_ProcessLog(t *testing.T) {
	testTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	testCases := []struct {
		name         string
		input        *gkeautoscaler.AutoscalerLogFieldSet
		wantSummary  string
		wantSeverity *pb.Severity
	}{
		{
			name: "scale up",
			input: &gkeautoscaler.AutoscalerLogFieldSet{
				DecisionLog: &gkeautoscaler.DecisionLog{
					ScaleUp: &gkeautoscaler.ScaleUpItem{
						IncreasedMigs: []gkeautoscaler.IncreasedMIGItem{
							{
								Mig: gkeautoscaler.MIGItem{
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
			wantSeverity: inspectioncore.SeverityWarning,
		},
		{
			name: "scale down",
			input: &gkeautoscaler.AutoscalerLogFieldSet{
				DecisionLog: &gkeautoscaler.DecisionLog{
					ScaleDown: &gkeautoscaler.ScaleDownItem{
						NodesToBeRemoved: []gkeautoscaler.NodeToBeRemovedItem{
							{
								Node: gkeautoscaler.NodeItem{
									Name: "test-cluster-default-pool-c47ef39f-p395",
									Mig: gkeautoscaler.MIGItem{
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
			wantSeverity: inspectioncore.SeverityWarning,
		},
		{
			name: "nodepool created",
			input: &gkeautoscaler.AutoscalerLogFieldSet{
				DecisionLog: &gkeautoscaler.DecisionLog{
					NodePoolCreated: &gkeautoscaler.NodepoolCreatedItem{
						NodePools: []gkeautoscaler.NodepoolItem{
							{
								Name: "nap-n1-standard-1-1kwag2qv",
							},
						},
					},
				},
			},
			wantSummary:  "Nodepool created by node auto provisioner: nap-n1-standard-1-1kwag2qv",
			wantSeverity: inspectioncore.SeverityWarning,
		},
		{
			name: "nodepool deleted",
			input: &gkeautoscaler.AutoscalerLogFieldSet{
				DecisionLog: &gkeautoscaler.DecisionLog{
					NodePoolDeleted: &gkeautoscaler.NodepoolDeletedItem{
						NodePoolNames: []string{
							"nap-n1-highcpu-8-ydj4ewil",
						},
					},
				},
			},
			wantSummary:  "Nodepool deleted by node auto provisioner: nap-n1-highcpu-8-ydj4ewil",
			wantSeverity: inspectioncore.SeverityWarning,
		},
		{
			name: "no scale up",
			input: &gkeautoscaler.AutoscalerLogFieldSet{
				NoDecisionLog: &gkeautoscaler.NoDecisionStatusLog{
					NoScaleUp: &gkeautoscaler.NoScaleUpItem{},
				},
			},
			wantSummary:  "autoscaler decided not to scale up",
			wantSeverity: inspectioncore.SeverityInfo,
		},
		{
			name: "no scale down with param",
			input: &gkeautoscaler.AutoscalerLogFieldSet{
				NoDecisionLog: &gkeautoscaler.NoDecisionStatusLog{
					NoScaleDown: &gkeautoscaler.NoScaleDownItem{
						Reason: gkeautoscaler.ReasonItem{
							MessageId:  "no.scale.down.in.backoff",
							Parameters: []string{"param1", "param2"},
						},
					},
				},
			},
			wantSummary:  "autoscaler decided not to scale down: no.scale.down.in.backoff(param1,param2)",
			wantSeverity: inspectioncore.SeverityInfo,
		},
		{
			name: "no scale down without param",
			input: &gkeautoscaler.AutoscalerLogFieldSet{
				NoDecisionLog: &gkeautoscaler.NoDecisionStatusLog{
					NoScaleDown: &gkeautoscaler.NoScaleDownItem{
						Reason: gkeautoscaler.ReasonItem{
							MessageId: "no.scale.down.in.backoff",
						},
					},
				},
			},
			wantSummary:  "autoscaler decided not to scale down: no.scale.down.in.backoff",
			wantSeverity: inspectioncore.SeverityInfo,
		},
		{
			name: "result info success",
			input: &gkeautoscaler.AutoscalerLogFieldSet{
				ResultInfoLog: &gkeautoscaler.ResultInfoLog{
					Results: []gkeautoscaler.Result{
						{
							EventID: "2fca91cd-7345-47fc-9770-838e05e28b17",
						},
					},
				},
			},
			wantSummary:  "autoscaler finished events: 2fca91cd-7345-47fc-9770-838e05e28b17(Success)",
			wantSeverity: inspectioncore.SeverityInfo,
		},
		{
			name: "result info error",
			input: &gkeautoscaler.AutoscalerLogFieldSet{
				ResultInfoLog: &gkeautoscaler.ResultInfoLog{
					Results: []gkeautoscaler.Result{
						{
							EventID: "ea2e964c-49b8-4cd7-8fa9-fefb0827f9a6",
							ErrorMsg: &gkeautoscaler.ErrorMessageItem{
								MessageId:  "scale.down.error.failed.to.delete.node.min.size.reached",
								Parameters: []string{"test-cluster-default-pool-5c90f485-nk80"},
							},
						},
					},
				},
			},
			wantSummary:  "autoscaler finished events: ea2e964c-49b8-4cd7-8fa9-fefb0827f9a6(Error:scale.down.error.failed.to.delete.node.min.size.reached(test-cluster-default-pool-5c90f485-nk80))",
			wantSeverity: inspectioncore.SeverityInfo,
		},
	}

	ingester := &autoscalerLogIngester{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			l := testlog.NewMockLog(
				testTime,
				*tc.input,
			)
			cs, err := ingester.ProcessLog(t.Context(), l)
			if err != nil {
				t.Fatalf("ProcessLog() error = %v", err)
			}

			testchangeset.AssertLog(t, cs).
				HasSummary(tc.wantSummary).
				HasSeverity(tc.wantSeverity).
				HasLogType(gkeautoscaler.LogTypeAutoscaler).
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
	builder := khifilev6.NewTestBuilder(id.NewGenerator())

	// 2. Resolve comparative path instances using the Builder's accumulator.
	ctx := khictx.WithValue(t.Context(), inspectioncore.Builder, builder)

	projectTimeline := gcpcommon.MustGCPProjectTimeline(ctx, "test-project")
	gkeClusterTimeline := gcpcommon.MustGKEClusterTimeline(ctx, projectTimeline, "test-cluster")
	k8sClusterTimeline := k8saudit.MustK8sClusterTimeline(ctx, "test-cluster")
	autoscalerPath := gkeautoscaler.MustAutoscalerTimeline(ctx, gkeClusterTimeline)
	nodepoolTimeline := gcpcommon.MustGKENodePoolTimeline(ctx, gkeClusterTimeline, "default-pool")
	migPath := gkeautoscaler.MustMigTimeline(ctx, nodepoolTimeline, "test-cluster-default-pool-a0c72690-grp")

	apiVersionTimeline := k8saudit.MustK8sAPIVersionTimeline(ctx, k8sClusterTimeline, "core/v1")
	kindTimeline := k8saudit.MustK8sKindTimeline(ctx, apiVersionTimeline, "pod")
	namespaceTimeline := k8saudit.MustK8sNamespaceTimeline(ctx, kindTimeline, "default")
	podPath := k8saudit.MustK8sNamespacedResourceTimeline(ctx, namespaceTimeline, "test-85958b848b-ptc7n")

	// Additional paths for other test cases
	scaleDownMigPath := gkeautoscaler.MustMigTimeline(ctx, nodepoolTimeline, "test-cluster-default-pool-c47ef39f-grp")

	kubeDnsNamespaceTimeline := k8saudit.MustK8sNamespaceTimeline(ctx, kindTimeline, "kube-system")
	kubeDnsPodPath := k8saudit.MustK8sNamespacedResourceTimeline(ctx, kubeDnsNamespaceTimeline, "kube-dns-5c44c7b6b6-xvpbk")

	nodeKindTimeline := k8saudit.MustK8sKindTimeline(ctx, apiVersionTimeline, "node")
	nodePath := k8saudit.MustK8sClusterScopeResourceTimeline(ctx, nodeKindTimeline, "test-cluster-default-pool-c47ef39f-p395")

	// Node auto provisioning creation
	napNodepoolTimeline := gcpcommon.MustGKENodePoolTimeline(ctx, gkeClusterTimeline, "nap-n1-standard-1-1kwag2qv")
	napMigPath := gkeautoscaler.MustMigTimeline(ctx, napNodepoolTimeline, "test-cluster-nap-n1-standard--b4fcc348-grp")

	// Node auto provisioning deletion
	napDeletedNodepoolTimeline := gcpcommon.MustGKENodePoolTimeline(ctx, gkeClusterTimeline, "nap-n1-highcpu-8-ydj4ewil")

	// No scale up
	skippedNodepoolTimeline := gcpcommon.MustGKENodePoolTimeline(ctx, gkeClusterTimeline, "nap-n1-highmem-4-1cywzhvf")
	skippedMigPath := gkeautoscaler.MustMigTimeline(ctx, skippedNodepoolTimeline, "test-cluster-nap-n1-highmem-4-fbdca585-grp")

	unhandledPodNamespaceTimeline := k8saudit.MustK8sNamespaceTimeline(ctx, kindTimeline, "autoscaling-1661")
	unhandledPodPath := k8saudit.MustK8sNamespacedResourceTimeline(ctx, unhandledPodNamespaceTimeline, "memory-reservation2-6zg8m")

	rejectedMigPath := gkeautoscaler.MustMigTimeline(ctx, nodepoolTimeline, "test-cluster-default-pool-b1808ff9-grp")

	// No scale down
	noScaleDownNodePath := k8saudit.MustK8sClusterScopeResourceTimeline(ctx, nodeKindTimeline, "test-cluster-default-pool-f74c1617-fbhk")
	noScaleDownMigPath := gkeautoscaler.MustMigTimeline(ctx, nodepoolTimeline, "test-cluster-default-pool-f74c1617-grp")

	testCases := []struct {
		name   string
		input  *gkeautoscaler.AutoscalerLogFieldSet
		assert func(t *testing.T, cs *khifilev6.TimelineChangeSet)
	}{
		{
			name: "scale up",
			input: &gkeautoscaler.AutoscalerLogFieldSet{
				DecisionLog: &gkeautoscaler.DecisionLog{
					ScaleUp: &gkeautoscaler.ScaleUpItem{
						IncreasedMigs: []gkeautoscaler.IncreasedMIGItem{
							{
								Mig: gkeautoscaler.MIGItem{
									Nodepool: "default-pool",
									Name:     "test-cluster-default-pool-a0c72690-grp",
								},
								RequestedNodes: 1,
							},
						},
						TriggeringPods: []gkeautoscaler.PodItem{
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
			input: &gkeautoscaler.AutoscalerLogFieldSet{
				DecisionLog: &gkeautoscaler.DecisionLog{
					ScaleDown: &gkeautoscaler.ScaleDownItem{
						NodesToBeRemoved: []gkeautoscaler.NodeToBeRemovedItem{
							{
								Node: gkeautoscaler.NodeItem{
									Name: "test-cluster-default-pool-c47ef39f-p395",
									Mig: gkeautoscaler.MIGItem{
										Nodepool: "default-pool",
										Name:     "test-cluster-default-pool-c47ef39f-grp",
									},
								},
								EvictedPods: []gkeautoscaler.PodItem{
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
			input: &gkeautoscaler.AutoscalerLogFieldSet{
				DecisionLog: &gkeautoscaler.DecisionLog{
					NodePoolCreated: &gkeautoscaler.NodepoolCreatedItem{
						NodePools: []gkeautoscaler.NodepoolItem{
							{
								Name: "nap-n1-standard-1-1kwag2qv",
								Migs: []gkeautoscaler.MIGItem{
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
			input: &gkeautoscaler.AutoscalerLogFieldSet{
				DecisionLog: &gkeautoscaler.DecisionLog{
					NodePoolDeleted: &gkeautoscaler.NodepoolDeletedItem{
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
			input: &gkeautoscaler.AutoscalerLogFieldSet{
				NoDecisionLog: &gkeautoscaler.NoDecisionStatusLog{
					NoScaleUp: &gkeautoscaler.NoScaleUpItem{
						SkippedMigs: []gkeautoscaler.SkippedMIGItem{
							{
								Mig: gkeautoscaler.MIGItem{
									Nodepool: "nap-n1-highmem-4-1cywzhvf",
									Name:     "test-cluster-nap-n1-highmem-4-fbdca585-grp",
								},
							},
						},
						UnhandledPodGroups: []gkeautoscaler.UnhandledPodGroupItem{
							{
								PodGroup: gkeautoscaler.PodGroup{
									SamplePod: gkeautoscaler.PodItem{
										Name:      "memory-reservation2-6zg8m",
										Namespace: "autoscaling-1661",
									},
								},
								RejectedMigs: []gkeautoscaler.RejectedMIGItem{
									{
										Mig: gkeautoscaler.MIGItem{
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
			input: &gkeautoscaler.AutoscalerLogFieldSet{
				NoDecisionLog: &gkeautoscaler.NoDecisionStatusLog{
					NoScaleDown: &gkeautoscaler.NoScaleDownItem{
						Nodes: []gkeautoscaler.NoScaleDownNodeItem{
							{
								Node: gkeautoscaler.NodeItem{
									Name: "test-cluster-default-pool-f74c1617-fbhk",
									Mig: gkeautoscaler.MIGItem{
										Nodepool: "default-pool",
										Name:     "test-cluster-default-pool-f74c1617-grp",
									},
								},
							},
						},
						Reason: gkeautoscaler.ReasonItem{
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
			input: &gkeautoscaler.AutoscalerLogFieldSet{
				ResultInfoLog: &gkeautoscaler.ResultInfoLog{
					Results: []gkeautoscaler.Result{
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
						StateType:    gkeautoscaler.RevisionAutoscalerNoError,
						ResourceBody: bodyNode,
					}, nodeCmpOpt)
			},
		},
		{
			name: "result info error",
			input: &gkeautoscaler.AutoscalerLogFieldSet{
				ResultInfoLog: &gkeautoscaler.ResultInfoLog{
					Results: []gkeautoscaler.Result{
						{
							EventID: "ea2e964c-49b8-4cd7-8fa9-fefb0827f9a6",
							ErrorMsg: &gkeautoscaler.ErrorMessageItem{
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
						StateType:    gkeautoscaler.RevisionAutoscalerHasErrors,
						ResourceBody: bodyNode,
					}, nodeCmpOpt)
			},
		},
	}

	mapper := &autoscalerTimelineMapper{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.input.ProjectID = "test-project"
			tc.input.ClusterName = "test-cluster"
			ctx := khictx.WithValue(t.Context(), inspectioncore.Builder, builder)

			l := testlog.NewMockLog(
				testTime,
				*tc.input,
			)

			cs, _, err := mapper.ProcessLogByGroup(ctx, l, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLogByGroup() error = %v", err)
			}

			tc.assert(t, cs)
		})
	}
}
