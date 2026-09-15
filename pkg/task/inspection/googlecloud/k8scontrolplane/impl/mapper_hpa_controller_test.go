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

package k8scontrolplane_impl

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/id"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scontrolplane"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
)

func TestHpaControllerTimelineMapper_ProcessLogByGroup(t *testing.T) {
	builder := khifilev6.NewTestBuilder(id.NewGenerator())

	projectTimeline := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
		Name: "test-project",
		Type: gcpcommon.TimelineTypeGCPProject,
	})
	gkeClusterTimeline := builder.TimelineAccumulator.GetPath(projectTimeline, khifilev6.PathSegment{
		Name: "test-cluster",
		Type: gcpcommon.TimelineTypeGKE,
	})
	controlPlanesTimeline := builder.TimelineAccumulator.GetPath(gkeClusterTimeline, khifilev6.PathSegment{
		Name: "controlplanes",
		Type: gcpcommon.TimelineTypeGKEControlPlanes,
	})
	wantCompTimeline := builder.TimelineAccumulator.GetPath(controlPlanesTimeline, khifilev6.PathSegment{
		Name: "hpa-controller",
		Type: k8scontrolplane.TimelineTypeControlPlaneComponent,
	})

	k8sClusterTimeline := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
		Name: "test-cluster",
		Type: inspectioncore.TimelineTypeK8sCluster,
	})

	autoscalingApiTimeline := builder.TimelineAccumulator.GetPath(k8sClusterTimeline, khifilev6.PathSegment{
		Name: "autoscaling/v2",
		Type: inspectioncore.TimelineTypeAPIVersion,
	})
	hpaKindTimeline := builder.TimelineAccumulator.GetPath(autoscalingApiTimeline, khifilev6.PathSegment{
		Name: "horizontalpodautoscaler",
		Type: inspectioncore.TimelineTypeKind,
	})
	hpaNamespaceTimeline := builder.TimelineAccumulator.GetPath(hpaKindTimeline, khifilev6.PathSegment{
		Name: "gke-managed-cim",
		Type: inspectioncore.TimelineTypeNamespace,
	})
	wantHpaTimeline := builder.TimelineAccumulator.GetPath(hpaNamespaceTimeline, khifilev6.PathSegment{
		Name: "kube-state-metrics",
		Type: inspectioncore.TimelineTypeResource,
	})

	appsApiTimeline := builder.TimelineAccumulator.GetPath(k8sClusterTimeline, khifilev6.PathSegment{
		Name: "apps/v1",
		Type: inspectioncore.TimelineTypeAPIVersion,
	})
	statefulSetKindTimeline := builder.TimelineAccumulator.GetPath(appsApiTimeline, khifilev6.PathSegment{
		Name: "statefulset",
		Type: inspectioncore.TimelineTypeKind,
	})
	statefulSetNamespaceTimeline := builder.TimelineAccumulator.GetPath(statefulSetKindTimeline, khifilev6.PathSegment{
		Name: "gke-managed-cim",
		Type: inspectioncore.TimelineTypeNamespace,
	})
	wantTargetTimeline := builder.TimelineAccumulator.GetPath(statefulSetNamespaceTimeline, khifilev6.PathSegment{
		Name: "kube-state-metrics",
		Type: inspectioncore.TimelineTypeResource,
	})

	testCases := []struct {
		desc                string
		inputComponentField k8scontrolplane.K8sControlplaneComponentFieldSet
		inputHPAField       k8scontrolplane.K8sHPAControllerFieldSet
		assert              func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet)
	}{
		{
			desc: "final recommendation with targetRef maps to controlplane, HPA, and target workload",
			inputComponentField: k8scontrolplane.K8sControlplaneComponentFieldSet{
				ProjectID:     "test-project",
				ClusterName:   "test-cluster",
				ComponentName: "hpa-controller",
			},
			inputHPAField: k8scontrolplane.K8sHPAControllerFieldSet{
				FinalRecommendation: &k8scontrolplane.HPAFinalRecommendation{
					HPA:            "gke-managed-cim/kube-state-metrics",
					HPANamespace:   "gke-managed-cim",
					HPAName:        "kube-state-metrics",
					ConfiguredSize: 1,
					Replicas:       1,
					TargetRef: k8scontrolplane.HPATargetRef{
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
			inputComponentField: k8scontrolplane.K8sControlplaneComponentFieldSet{
				ProjectID:     "test-project",
				ClusterName:   "test-cluster",
				ComponentName: "hpa-controller",
			},
			inputHPAField: k8scontrolplane.K8sHPAControllerFieldSet{
				AtomicRecommendation: &k8scontrolplane.HPAAtomicRecommendation{
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
			inputComponentField: k8scontrolplane.K8sControlplaneComponentFieldSet{
				ProjectID:     "test-project",
				ClusterName:   "test-cluster",
				ComponentName: "hpa-controller",
			},
			inputHPAField: k8scontrolplane.K8sHPAControllerFieldSet{
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
			inputComponentField: k8scontrolplane.K8sControlplaneComponentFieldSet{
				ProjectID:     "test-project",
				ClusterName:   "test-cluster",
				ComponentName: "hpa-controller",
			},
			inputHPAField: k8scontrolplane.K8sHPAControllerFieldSet{
				FinalRecommendation: &k8scontrolplane.HPAFinalRecommendation{
					HPA:            "gke-managed-cim/kube-state-metrics",
					HPANamespace:   "gke-managed-cim",
					HPAName:        "kube-state-metrics",
					ConfiguredSize: 1,
					Replicas:       1,
					TargetRef: k8scontrolplane.HPATargetRef{
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
			ctx := khictx.WithValue(t.Context(), inspectioncore.Builder, builder)
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
