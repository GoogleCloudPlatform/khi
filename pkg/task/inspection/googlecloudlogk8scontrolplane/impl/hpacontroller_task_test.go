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

package googlecloudlogk8scontrolplane_impl

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/id"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogk8scontrolplane_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontrolplane/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
)

func TestHpaControllerTimelineMapper_ProcessLogByGroup(t *testing.T) {
	builder := khifilev6.NewTestBuilder(id.NewGenerator())

	projectTimeline := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
		Name: "test-project",
		Type: googlecloudcommon_contract.TimelineTypeGCPProject,
	})
	gkeClusterTimeline := builder.TimelineAccumulator.GetPath(projectTimeline, khifilev6.PathSegment{
		Name: "test-cluster",
		Type: googlecloudcommon_contract.TimelineTypeGKE,
	})
	controlPlanesTimeline := builder.TimelineAccumulator.GetPath(gkeClusterTimeline, khifilev6.PathSegment{
		Name: "controlplanes",
		Type: googlecloudcommon_contract.TimelineTypeGKEControlPlanes,
	})
	wantCompTimeline := builder.TimelineAccumulator.GetPath(controlPlanesTimeline, khifilev6.PathSegment{
		Name: "hpa-controller",
		Type: googlecloudlogk8scontrolplane_contract.TimelineTypeControlPlaneComponent,
	})

	k8sClusterTimeline := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
		Name: "test-cluster",
		Type: inspectioncore_contract.TimelineTypeK8sCluster,
	})

	autoscalingApiTimeline := builder.TimelineAccumulator.GetPath(k8sClusterTimeline, khifilev6.PathSegment{
		Name: "autoscaling/v2",
		Type: inspectioncore_contract.TimelineTypeAPIVersion,
	})
	hpaKindTimeline := builder.TimelineAccumulator.GetPath(autoscalingApiTimeline, khifilev6.PathSegment{
		Name: "horizontalpodautoscaler",
		Type: inspectioncore_contract.TimelineTypeKind,
	})
	hpaNamespaceTimeline := builder.TimelineAccumulator.GetPath(hpaKindTimeline, khifilev6.PathSegment{
		Name: "gke-managed-cim",
		Type: inspectioncore_contract.TimelineTypeNamespace,
	})
	wantHpaTimeline := builder.TimelineAccumulator.GetPath(hpaNamespaceTimeline, khifilev6.PathSegment{
		Name: "kube-state-metrics",
		Type: inspectioncore_contract.TimelineTypeResource,
	})

	appsApiTimeline := builder.TimelineAccumulator.GetPath(k8sClusterTimeline, khifilev6.PathSegment{
		Name: "apps/v1",
		Type: inspectioncore_contract.TimelineTypeAPIVersion,
	})
	statefulSetKindTimeline := builder.TimelineAccumulator.GetPath(appsApiTimeline, khifilev6.PathSegment{
		Name: "statefulset",
		Type: inspectioncore_contract.TimelineTypeKind,
	})
	statefulSetNamespaceTimeline := builder.TimelineAccumulator.GetPath(statefulSetKindTimeline, khifilev6.PathSegment{
		Name: "gke-managed-cim",
		Type: inspectioncore_contract.TimelineTypeNamespace,
	})
	wantTargetTimeline := builder.TimelineAccumulator.GetPath(statefulSetNamespaceTimeline, khifilev6.PathSegment{
		Name: "kube-state-metrics",
		Type: inspectioncore_contract.TimelineTypeResource,
	})

	testCases := []struct {
		desc                string
		inputComponentField googlecloudlogk8scontrolplane_contract.K8sControlplaneComponentFieldSet
		inputHPAField       googlecloudlogk8scontrolplane_contract.K8sHPAControllerFieldSet
		assert              func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet)
	}{
		{
			desc: "final recommendation with targetRef maps to controlplane, HPA, and target workload",
			inputComponentField: googlecloudlogk8scontrolplane_contract.K8sControlplaneComponentFieldSet{
				ProjectID:     "test-project",
				ClusterName:   "test-cluster",
				ComponentName: "hpa-controller",
			},
			inputHPAField: googlecloudlogk8scontrolplane_contract.K8sHPAControllerFieldSet{
				FinalRecommendation: &googlecloudlogk8scontrolplane_contract.HPAFinalRecommendation{
					HPA:            "gke-managed-cim/kube-state-metrics",
					HPANamespace:   "gke-managed-cim",
					HPAName:        "kube-state-metrics",
					ConfiguredSize: 1,
					Replicas:       1,
					TargetRef: googlecloudlogk8scontrolplane_contract.HPATargetRef{
						APIVersion: "apps/v1",
						Kind:       "StatefulSet",
						Name:       "kube-state-metrics",
					},
				},
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantCompTimeline).
					HasEvent(wantHpaTimeline).
					HasEvent(wantTargetTimeline)
			},
		},
		{
			desc: "atomic recommendation maps to controlplane and HPA",
			inputComponentField: googlecloudlogk8scontrolplane_contract.K8sControlplaneComponentFieldSet{
				ProjectID:     "test-project",
				ClusterName:   "test-cluster",
				ComponentName: "hpa-controller",
			},
			inputHPAField: googlecloudlogk8scontrolplane_contract.K8sHPAControllerFieldSet{
				AtomicRecommendation: &googlecloudlogk8scontrolplane_contract.HPAAtomicRecommendation{
					HPA:          "gke-managed-cim/kube-state-metrics",
					HPANamespace: "gke-managed-cim",
					HPAName:      "kube-state-metrics",
					MetricName:   "memory",
					Replicas:     1,
				},
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantCompTimeline).
					HasEvent(wantHpaTimeline).
					HasNoEvent(wantTargetTimeline)
			},
		},
		{
			desc: "fallback message maps only to controlplane",
			inputComponentField: googlecloudlogk8scontrolplane_contract.K8sControlplaneComponentFieldSet{
				ProjectID:     "test-project",
				ClusterName:   "test-cluster",
				ComponentName: "hpa-controller",
			},
			inputHPAField: googlecloudlogk8scontrolplane_contract.K8sHPAControllerFieldSet{
				Message: "starting hpa controller",
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantCompTimeline).
					HasNoEvent(wantHpaTimeline).
					HasNoEvent(wantTargetTimeline)
			},
		},
		{
			desc: "final recommendation with missing targetRef apiVersion does not map target workload timeline",
			inputComponentField: googlecloudlogk8scontrolplane_contract.K8sControlplaneComponentFieldSet{
				ProjectID:     "test-project",
				ClusterName:   "test-cluster",
				ComponentName: "hpa-controller",
			},
			inputHPAField: googlecloudlogk8scontrolplane_contract.K8sHPAControllerFieldSet{
				FinalRecommendation: &googlecloudlogk8scontrolplane_contract.HPAFinalRecommendation{
					HPA:            "gke-managed-cim/kube-state-metrics",
					HPANamespace:   "gke-managed-cim",
					HPAName:        "kube-state-metrics",
					ConfiguredSize: 1,
					Replicas:       1,
					TargetRef: googlecloudlogk8scontrolplane_contract.HPATargetRef{
						Kind: "StatefulSet",
						Name: "kube-state-metrics",
					},
				},
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantCompTimeline).
					HasEvent(wantHpaTimeline).
					HasNoEvent(wantTargetTimeline)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)
			l := testlog.NewMockLog(tc.inputComponentField, tc.inputHPAField)
			mapper := &HpaControllerTimelineMapper{}
			cs, _, err := mapper.ProcessLogByGroup(ctx, l, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLogByGroup() returned an unexpected error, err=%v", err)
			}
			tc.assert(t, ctx, cs)
		})
	}
}
