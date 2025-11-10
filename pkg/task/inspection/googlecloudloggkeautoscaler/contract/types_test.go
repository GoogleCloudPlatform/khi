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

package googlecloudloggkeautoscaler_contract

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestAutoscalerDecisionLogTypesToMatchLogString(t *testing.T) {
	testCases := []struct {
		Name           string
		InputJSON      string
		ExpectedResult any
	}{
		{
			Name: "Scale Up",
			InputJSON: `{
				"decideTime": "1582124907",
				"eventId": "ed5cb16d-b06f-457c-a46d-f75dcca1f1ee",
				"scaleUp": {
				  "increasedMigs": [
					{
					  "mig": {
						"name": "test-cluster-default-pool-a0c72690-grp",
						"nodepool": "default-pool",
						"zone": "us-central1-c"
					  },
					  "requestedNodes": 1
					}
				  ],
				  "triggeringPods": [
					{
					  "controller": {
						"apiVersion": "apps/v1",
						"kind": "ReplicaSet",
						"name": "test-85958b848b"
					  },
					  "name": "test-85958b848b-ptc7n",
					  "namespace": "default"
					}
				  ],
				  "triggeringPodsTotalCount": 1
				}
			  }`,
			ExpectedResult: DecisionLog{
				DecideTime: "1582124907",
				EventID:    "ed5cb16d-b06f-457c-a46d-f75dcca1f1ee",
				ScaleUp: &ScaleUpItem{
					IncreasedMigs: []IncreasedMIGItem{
						{
							Mig: MIGItem{
								Name:     "test-cluster-default-pool-a0c72690-grp",
								Nodepool: "default-pool",
								Zone:     "us-central1-c",
							},
							RequestedNodes: 1,
						},
					},
					TriggeringPods: []PodItem{
						{
							Controller: ControllerItem{
								ApiVersion: "apps/v1",
								Kind:       "ReplicaSet",
								Name:       "test-85958b848b",
							},
							Name:      "test-85958b848b-ptc7n",
							Namespace: "default",
						},
					},
					TriggeringPodsTotalCount: 1,
				},
			},
		},
		{
			Name: "Scale Down",
			InputJSON: `{
				  "decideTime": "1580594665",
				  "eventId": "340dac18-8152-46ff-b79a-747f70854c81",
				  "scaleDown": {
					"nodesToBeRemoved": [
					  {
						"evictedPods": [
						  {
							"controller": {
							  "apiVersion": "apps/v1",
							  "kind": "ReplicaSet",
							  "name": "kube-dns-5c44c7b6b6"
							},
							"name": "kube-dns-5c44c7b6b6-xvpbk",
							"namespace": "kube-system"
						  }
						],
						"evictedPodsTotalCount": 1,
						"node": {
						  "cpuRatio": 23,
						  "memRatio": 5,
						  "mig": {
							"name": "test-cluster-default-pool-c47ef39f-grp",
							"nodepool": "default-pool",
							"zone": "us-central1-f"
						  },
						  "name": "test-cluster-default-pool-c47ef39f-p395"
						}
					  }
					]
				  }
			  }`,
			ExpectedResult: DecisionLog{
				DecideTime: "1580594665",
				EventID:    "340dac18-8152-46ff-b79a-747f70854c81",
				ScaleDown: &ScaleDownItem{
					NodesToBeRemoved: []NodeToBeRemovedItem{
						{
							EvictedPods: []PodItem{
								{
									Controller: ControllerItem{
										ApiVersion: "apps/v1",
										Kind:       "ReplicaSet",
										Name:       "kube-dns-5c44c7b6b6",
									},
									Name:      "kube-dns-5c44c7b6b6-xvpbk",
									Namespace: "kube-system",
								},
							},
							EvictedPodsTotalCount: 1,
							Node: NodeItem{
								CpuRatio: 23,
								MemRatio: 5,
								Mig: MIGItem{
									Name:     "test-cluster-default-pool-c47ef39f-grp",
									Nodepool: "default-pool",
									Zone:     "us-central1-f",
								},
								Name: "test-cluster-default-pool-c47ef39f-p395",
							},
						},
					},
				},
			},
		},
		{
			Name: "Node Pool Creation",
			InputJSON: `{
				"decideTime": "1585838544",
				"eventId": "822d272c-f4f3-44cf-9326-9cad79c58718",
				"nodePoolCreated": {
				  "nodePools": [
					{
					  "migs": [
						{
						  "name": "test-cluster-nap-n1-standard--b4fcc348-grp",
						  "nodepool": "nap-n1-standard-1-1kwag2qv",
						  "zone": "us-central1-f"
						},
						{
						  "name": "test-cluster-nap-n1-standard--jfla8215-grp",
						  "nodepool": "nap-n1-standard-1-1kwag2qv",
						  "zone": "us-central1-c"
						}
					  ],
					  "name": "nap-n1-standard-1-1kwag2qv"
					}
				  ],
				  "triggeringScaleUpId": "d25e0e6e-25e3-4755-98eb-49b38e54a728"
				}
			}`,
			ExpectedResult: DecisionLog{
				DecideTime: "1585838544",
				EventID:    "822d272c-f4f3-44cf-9326-9cad79c58718",
				NodePoolCreated: &NodepoolCreatedItem{
					TriggeringScaleUpId: "d25e0e6e-25e3-4755-98eb-49b38e54a728",
					NodePools: []NodepoolItem{
						{
							Name: "nap-n1-standard-1-1kwag2qv",
							Migs: []MIGItem{
								{
									Name:     "test-cluster-nap-n1-standard--b4fcc348-grp",
									Nodepool: "nap-n1-standard-1-1kwag2qv",
									Zone:     "us-central1-f",
								},
								{
									Name:     "test-cluster-nap-n1-standard--jfla8215-grp",
									Nodepool: "nap-n1-standard-1-1kwag2qv",
									Zone:     "us-central1-c",
								},
							},
						},
					},
				},
			},
		},
		{
			Name: "Node Pool Deletion",
			InputJSON: `{
				"decideTime": "1585830461",
				"eventId": "68b0d1c7-b684-4542-bc19-f030922fb820",
				"nodePoolDeleted": {
				  "nodePoolNames": [
					"nap-n1-highcpu-8-ydj4ewil"
				  ]
				}
            }`,
			ExpectedResult: DecisionLog{
				DecideTime: "1585830461",
				EventID:    "68b0d1c7-b684-4542-bc19-f030922fb820",
				NodePoolDeleted: &NodepoolDeletedItem{
					NodePoolNames: []string{"nap-n1-highcpu-8-ydj4ewil"},
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			var result DecisionLog
			err := json.Unmarshal([]byte(testCase.InputJSON), &result)
			if err != nil {
				t.Errorf("unexpected error\n%v", err)
			}
			if diff := cmp.Diff(testCase.ExpectedResult, result); diff != "" {
				t.Errorf("the generated result is not matching with the expected value\n%s", diff)
			}
		})
	}
}

func TestAutoscalerNoDecisionLogTypesToMatchLogString(t *testing.T) {
	testCases := []struct {
		Name           string
		InputJSON      string
		ExpectedResult NoDecisionStatusLog
	}{
		{
			Name: "noScaleUp",
			InputJSON: `{
				  "measureTime": "1582523362",
				  "noScaleUp": {
					"skippedMigs": [
					  {
						"mig": {
						  "name": "test-cluster-nap-n1-highmem-4-fbdca585-grp",
						  "nodepool": "nap-n1-highmem-4-1cywzhvf",
						  "zone": "us-central1-f"
						},
						"reason": {
						  "messageId": "no.scale.up.mig.skipped",
						  "parameters": [
							"max cluster cpu limit reached"
						  ]
						}
					  }
					],
					"unhandledPodGroups": [
					  {
						"napFailureReasons": [
						  {
							"messageId": "no.scale.up.nap.pod.zonal.resources.exceeded",
							"parameters": [
							  "us-central1-f"
							]
						  }
						],
						"podGroup": {
						  "samplePod": {
							"controller": {
							  "apiVersion": "v1",
							  "kind": "ReplicationController",
							  "name": "memory-reservation2"
							},
							"name": "memory-reservation2-6zg8m",
							"namespace": "autoscaling-1661"
						  },
						  "totalPodCount": 1
						},
						"rejectedMigs": [
						  {
							"mig": {
							  "name": "test-cluster-default-pool-b1808ff9-grp",
							  "nodepool": "default-pool",
							  "zone": "us-central1-f"
							},
							"reason": {
							  "messageId": "no.scale.up.mig.failing.predicate",
							  "parameters": [
								"NodeResourcesFit",
								"Insufficient memory"
							  ]
							}
						  }
						]
					  }
					],
					"unhandledPodGroupsTotalCount": 1
				  }
			  }`,
			ExpectedResult: NoDecisionStatusLog{
				MeasureTime: "1582523362",
				NoScaleUp: &NoScaleUpItem{
					SkippedMigs: []SkippedMIGItem{
						{
							Mig: MIGItem{
								Name:     "test-cluster-nap-n1-highmem-4-fbdca585-grp",
								Nodepool: "nap-n1-highmem-4-1cywzhvf",
								Zone:     "us-central1-f",
							},
							Reason: ReasonItem{
								MessageId:  "no.scale.up.mig.skipped",
								Parameters: []string{"max cluster cpu limit reached"},
							},
						},
					},
					UnhandledPodGroups: []UnhandledPodGroupItem{
						{
							NAPFailureReasons: []NapFailureReasonItem{
								{
									MessageId:  "no.scale.up.nap.pod.zonal.resources.exceeded",
									Parameters: []string{"us-central1-f"},
								},
							},
							PodGroup: PodGroup{
								SamplePod: PodItem{
									Controller: ControllerItem{
										ApiVersion: "v1",
										Kind:       "ReplicationController",
										Name:       "memory-reservation2",
									},
									Name:      "memory-reservation2-6zg8m",
									Namespace: "autoscaling-1661",
								},
								TotalPodCount: 1,
							},
							RejectedMigs: []RejectedMIGItem{
								{
									Mig: MIGItem{
										Name:     "test-cluster-default-pool-b1808ff9-grp",
										Nodepool: "default-pool",
										Zone:     "us-central1-f",
									},
									Reason: ReasonItem{
										MessageId:  "no.scale.up.mig.failing.predicate",
										Parameters: []string{"NodeResourcesFit", "Insufficient memory"},
									},
								},
							},
						},
					},
					UnhandledPodGroupsTotalCount: 1,
				},
			},
		},
		{
			Name: "noScaleDown",
			InputJSON: `{
				  "measureTime": "1582858723",
				  "noScaleDown": {
					"nodes": [
					  {
						"node": {
						  "cpuRatio": 42,
						  "mig": {
							"name": "test-cluster-default-pool-f74c1617-grp",
							"nodepool": "default-pool",
							"zone": "us-central1-c"
						  },
						  "name": "test-cluster-default-pool-f74c1617-fbhk"
						},
						"reason": {
						  "messageId": "no.scale.down.node.no.place.to.move.pods"
						}
					  }
					],
					"nodesTotalCount": 1,
					"reason": {
					  "messageId": "no.scale.down.in.backoff"
					}
				  }
			  }`,
			ExpectedResult: NoDecisionStatusLog{
				MeasureTime: "1582858723",
				NoScaleDown: &NoScaleDownItem{
					Nodes: []NoScaleDownNodeItem{{
						Node: NodeItem{
							CpuRatio: 42,
							Mig: MIGItem{
								Name:     "test-cluster-default-pool-f74c1617-grp",
								Nodepool: "default-pool",
								Zone:     "us-central1-c",
							},
							Name: "test-cluster-default-pool-f74c1617-fbhk",
						},
					},
					},
					NodesTotalCount: 1,
					Reason: ReasonItem{
						MessageId: "no.scale.down.in.backoff",
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			var result NoDecisionStatusLog
			err := json.Unmarshal([]byte(testCase.InputJSON), &result)
			if err != nil {
				t.Errorf("unexpected error\n%v", err)
			}
			if diff := cmp.Diff(testCase.ExpectedResult, result); diff != "" {
				t.Errorf("the generated result is not matching with the expected value\n%s", diff)
			}
		})
	}
}

func TestAutoscalerResultInfoLogTypesToMatchLogString(t *testing.T) {
	testCases := []struct {
		Name           string
		InputJSON      string
		ExpectedResult any
	}{{
		Name: "ResultInfo",
		InputJSON: `{
		"measureTime": "1582878896",
		"results": [
		  {
			"eventId": "2fca91cd-7345-47fc-9770-838e05e28b17"
		  },
		  {
			"errorMsg": {
			  "messageId": "scale.down.error.failed.to.delete.node.min.size.reached",
			  "parameters": [
				"test-cluster-default-pool-5c90f485-nk80"
			  ]
			},
			"eventId": "ea2e964c-49b8-4cd7-8fa9-fefb0827f9a6"
		  }
		]
	  }`,
		ExpectedResult: ResultInfoLog{
			MeasureTime: "1582878896",
			Results: []Result{
				{
					EventID: "2fca91cd-7345-47fc-9770-838e05e28b17",
				},
				{
					EventID: "ea2e964c-49b8-4cd7-8fa9-fefb0827f9a6",
					ErrorMsg: &ErrorMessageItem{
						MessageId:  "scale.down.error.failed.to.delete.node.min.size.reached",
						Parameters: []string{"test-cluster-default-pool-5c90f485-nk80"},
					},
				},
			},
		},
	}}
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			var result ResultInfoLog
			err := json.Unmarshal([]byte(testCase.InputJSON), &result)
			if err != nil {
				t.Errorf("unexpected error\n%v", err)
			}
			if diff := cmp.Diff(testCase.ExpectedResult, result); diff != "" {
				t.Errorf("the generated result is not matching with the expected value\n%s", diff)
			}
		})
	}
}
