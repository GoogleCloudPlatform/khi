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

package googlecloudloggkeautoscaler_contract

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	"github.com/google/go-cmp/cmp"
)

func TestAutoscalerLogFieldSetReader(t *testing.T) {
	testCases := []struct {
		desc  string
		input string
		want  *AutoscalerLogFieldSet
	}{
		{
			desc: "decision log",
			input: `insertId: 045dff22-0e13-4bd7-b3a1-19e8bb14319b@a1
logName: projects/test-project/logs/container.googleapis.com%2Fcluster-autoscaler-visibility
jsonPayload:
    decision:
        decideTime: "1762654527"
        eventId: be492585-c6ca-4286-ac59-dd846560d64a
        scaleDown:
            nodesToBeRemoved:
                - evictedPods:
                    - controller:
                        apiVersion: apps/v1
                        kind: ReplicaSet
                        name: konnectivity-agent-5d74559d97
                      name: konnectivity-agent-5d74559d97-9kx7g
                      namespace: kube-system
                  evictedPodsTotalCount: 1
                  node:
                    cpuRatio: 2
                    memRatio: 2
                    mig:
                        name: gke-ca-cluster-default-pool-915eef86-grp
                        nodepool: default-pool
                        zone: asia-northeast1-b
                    name: gke-ca-cluster-default-pool-915eef86-hvbn
resource:
    type: k8s_cluster
    labels:
        cluster_name: ca-cluster
        location: asia-northeast1
        project_id: test-project
receiveTimestamp: "2025-11-09T02:15:28.437679991Z"
timestamp: "2025-11-09T02:15:27.768533131Z"
`,
			want: &AutoscalerLogFieldSet{
				DecisionLog: &DecisionLog{
					DecideTime: "1762654527",
					EventID:    "be492585-c6ca-4286-ac59-dd846560d64a",
					ScaleDown: &ScaleDownItem{
						NodesToBeRemoved: []NodeToBeRemovedItem{
							{
								EvictedPods: []PodItem{
									{
										Controller: ControllerItem{
											ApiVersion: "apps/v1",
											Kind:       "ReplicaSet",
											Name:       "konnectivity-agent-5d74559d97",
										},
										Name:      "konnectivity-agent-5d74559d97-9kx7g",
										Namespace: "kube-system",
									},
								},
								EvictedPodsTotalCount: 1,
								Node: NodeItem{
									CpuRatio: 2,
									MemRatio: 2,
									Mig: MIGItem{
										Name:     "gke-ca-cluster-default-pool-915eef86-grp",
										Nodepool: "default-pool",
										Zone:     "asia-northeast1-b",
									},
									Name: "gke-ca-cluster-default-pool-915eef86-hvbn",
								},
							},
						},
					},
				},
			},
		},
		{
			desc: "no decision log",
			input: `insertId: 045dff22-0e13-4bd7-b3a1-19e8bb14319b@a1
logName: projects/test-project/logs/container.googleapis.com%2Fcluster-autoscaler-visibility
jsonPayload:
    noDecisionStatus:
        measureTime: "1762654527"
        noScaleDown:
            nodes:
                - node:
                    cpuRatio: 2
                    memRatio: 2
                    mig:
                        name: gke-ca-cluster-default-pool-915eef86-grp
                        nodepool: default-pool
                        zone: asia-northeast1-b
                    name: gke-ca-cluster-default-pool-915eef86-hvbn
            nodesTotalCount: 1
            reason:
                messageId: no.scale.down.in.backoff
                parameters:
                    - param1
                    - param2
resource:
    type: k8s_cluster
    labels:
        cluster_name: ca-cluster
        location: asia-northeast1
        project_id: test-project
receiveTimestamp: "2025-11-09T02:15:28.437679991Z"
timestamp: "2025-11-09T02:15:27.768533131Z"
`,
			want: &AutoscalerLogFieldSet{
				NoDecisionLog: &NoDecisionStatusLog{
					MeasureTime: "1762654527",
					NoScaleDown: &NoScaleDownItem{
						Nodes: []NoScaleDownNodeItem{
							{
								Node: NodeItem{
									CpuRatio: 2,
									MemRatio: 2,
									Mig: MIGItem{
										Name:     "gke-ca-cluster-default-pool-915eef86-grp",
										Nodepool: "default-pool",
										Zone:     "asia-northeast1-b",
									},
									Name: "gke-ca-cluster-default-pool-915eef86-hvbn",
								},
							},
						},
						NodesTotalCount: 1,
						Reason: ReasonItem{
							MessageId:  "no.scale.down.in.backoff",
							Parameters: []string{"param1", "param2"},
						},
					},
				},
			},
		},
		{
			desc: "result info log",
			input: `insertId: 4821bd8e-2a5f-4c4c-a410-88c5bb192c4e@a1
logName: projects/test-project/logs/container.googleapis.com%2Fcluster-autoscaler-visibility
jsonPayload:
    resultInfo:
        measureTime: "1762654602"
        results:
            - eventId: 0cdbc414-5b87-4d81-badb-c17a438d0c42
resource:
    type: k8s_cluster
    labels:
        cluster_name: ca-cluster
        location: asia-northeast1
        project_id: test-project
receiveTimestamp: "2025-11-09T02:16:43.703446262Z"
timestamp: "2025-11-09T02:16:43.237461906Z"
`,
			want: &AutoscalerLogFieldSet{
				ResultInfoLog: &ResultInfoLog{
					MeasureTime: "1762654602",
					Results: []Result{
						{
							EventID: "0cdbc414-5b87-4d81-badb-c17a438d0c42",
						},
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			node, err := structured.FromYAML(tc.input)
			if err != nil {
				t.Fatalf("failed to create node: %v", err)
			}

			fieldSetReader := &AutoscalerLogFieldSetReader{}
			got, err := fieldSetReader.Read(structured.NewNodeReader(node))
			if err != nil {
				t.Errorf("Read() error = %v", err)
				return
			}

			autoscalerFieldSet, ok := got.(*AutoscalerLogFieldSet)
			if !ok {
				t.Fatalf("expected *AutoscalerLogFieldSet, got %T", got)
			}

			if diff := cmp.Diff(tc.want, autoscalerFieldSet); diff != "" {
				t.Errorf("Read() mismatch (-want +got):\n%s", diff)
			}

		})
	}
}
