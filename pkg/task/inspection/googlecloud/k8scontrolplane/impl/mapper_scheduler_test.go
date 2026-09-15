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

func TestSchedulerLogToTimelineMapperTask(t *testing.T) {
	builder := khifilev6.NewTestBuilder(id.NewGenerator())

	projectTimeline := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
		Name: "test-project",
		Type: gcpcommon.TimelineTypeGCPProject,
	})
	gkeClusterTimeline := builder.TimelineAccumulator.GetPath(projectTimeline, khifilev6.PathSegment{
		Name: "test-cluster",
		Type: gcpcommon.TimelineTypeGKE,
	})
	wantControlPlanesTimeline := builder.TimelineAccumulator.GetPath(gkeClusterTimeline, khifilev6.PathSegment{
		Name: "controlplanes",
		Type: gcpcommon.TimelineTypeGKEControlPlanes,
	})
	wantCompTimeline := builder.TimelineAccumulator.GetPath(wantControlPlanesTimeline, khifilev6.PathSegment{
		Name: "scheduler",
		Type: k8scontrolplane.TimelineTypeControlPlaneComponent,
	})

	k8sClusterTimeline := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
		Name: "test-cluster",
		Type: inspectioncore.TimelineTypeK8sCluster,
	})

	apiVersionTimeline := builder.TimelineAccumulator.GetPath(k8sClusterTimeline, khifilev6.PathSegment{
		Name: "core/v1",
		Type: inspectioncore.TimelineTypeAPIVersion,
	})
	kindTimeline := builder.TimelineAccumulator.GetPath(apiVersionTimeline, khifilev6.PathSegment{
		Name: "pod",
		Type: inspectioncore.TimelineTypeKind,
	})
	namespaceTimeline := builder.TimelineAccumulator.GetPath(kindTimeline, khifilev6.PathSegment{
		Name: "test-namespace",
		Type: inspectioncore.TimelineTypeNamespace,
	})
	wantPodTimeline := builder.TimelineAccumulator.GetPath(namespaceTimeline, khifilev6.PathSegment{
		Name: "test-pod",
		Type: inspectioncore.TimelineTypeResource,
	})

	testCases := []struct {
		desc                   string
		inputComponentField    k8scontrolplane.K8sControlplaneComponentFieldSet
		inputMessageField      k8scontrolplane.K8sControlplaneCommonMessageFieldSet
		inputSchedulerFieldSet k8scontrolplane.K8sSchedulerComponentFieldSet
		assert                 func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet)
	}{
		{
			desc: "with pod name and namespace given",
			inputComponentField: k8scontrolplane.K8sControlplaneComponentFieldSet{
				ProjectID:     "test-project",
				ClusterName:   "test-cluster",
				ComponentName: "scheduler",
			},
			inputMessageField: k8scontrolplane.K8sControlplaneCommonMessageFieldSet{
				Message: "foo",
			},
			inputSchedulerFieldSet: k8scontrolplane.K8sSchedulerComponentFieldSet{
				PodName:      "test-pod",
				PodNamespace: "test-namespace",
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantCompTimeline).
					HasEvent(wantPodTimeline)
			},
		},
		{
			desc: "without pod name and namespace",
			inputComponentField: k8scontrolplane.K8sControlplaneComponentFieldSet{
				ProjectID:     "test-project",
				ClusterName:   "test-cluster",
				ComponentName: "scheduler",
			},
			inputMessageField: k8scontrolplane.K8sControlplaneCommonMessageFieldSet{
				Message: "foo",
			},
			inputSchedulerFieldSet: k8scontrolplane.K8sSchedulerComponentFieldSet{},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantCompTimeline).
					HasNoEvent(wantPodTimeline)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := khictx.WithValue(t.Context(), inspectioncore.Builder, builder)
			l := testlog.NewMockLog(tc.inputComponentField, tc.inputSchedulerFieldSet, tc.inputMessageField)
			mapper := &SchedulerTimelineMapper{}
			cs, _, err := mapper.ProcessLogByGroup(ctx, l, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLogByGroup() returned an unexpected error, err=%v", err)
			}
			tc.assert(t, ctx, cs)
		})
	}
}
