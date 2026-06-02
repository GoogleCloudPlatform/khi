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

package googlecloudlogk8scontrolplane_impl

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogk8scontrolplane_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontrolplane/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
)

func TestSchedulerLogToTimelineMapperTask(t *testing.T) {
	builder := khifilev6.NewBuilder()

	gkeClusterTimeline := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
		Name: "test-cluster",
		Type: googlecloudcommon_contract.TimelineTypeGKE,
	})
	wantCompTimeline := builder.TimelineAccumulator.GetPath(gkeClusterTimeline, khifilev6.PathSegment{
		Name: "scheduler",
		Type: googlecloudlogk8scontrolplane_contract.TimelineTypeControlPlaneComponent,
	})

	clusterTimeline := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
		Name: "test-cluster",
		Type: inspectioncore_contract.TimelineTypeK8sCluster,
	})

	apiVersionTimeline := builder.TimelineAccumulator.GetPath(clusterTimeline, khifilev6.PathSegment{
		Name: "core/v1",
		Type: inspectioncore_contract.TimelineTypeAPIVersion,
	})
	kindTimeline := builder.TimelineAccumulator.GetPath(apiVersionTimeline, khifilev6.PathSegment{
		Name: "pod",
		Type: inspectioncore_contract.TimelineTypeKind,
	})
	namespaceTimeline := builder.TimelineAccumulator.GetPath(kindTimeline, khifilev6.PathSegment{
		Name: "test-namespace",
		Type: inspectioncore_contract.TimelineTypeNamespace,
	})
	wantPodTimeline := builder.TimelineAccumulator.GetPath(namespaceTimeline, khifilev6.PathSegment{
		Name: "test-pod",
		Type: inspectioncore_contract.TimelineTypeResource,
	})

	testCases := []struct {
		desc                   string
		inputComponentField    googlecloudlogk8scontrolplane_contract.K8sControlplaneComponentFieldSet
		inputMessageField      googlecloudlogk8scontrolplane_contract.K8sControlplaneCommonMessageFieldSet
		inputSchedulerFieldSet googlecloudlogk8scontrolplane_contract.K8sSchedulerComponentFieldSet
		assert                 func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet)
	}{
		{
			desc: "with pod name and namespace given",
			inputComponentField: googlecloudlogk8scontrolplane_contract.K8sControlplaneComponentFieldSet{
				ClusterName:   "test-cluster",
				ComponentName: "scheduler",
			},
			inputMessageField: googlecloudlogk8scontrolplane_contract.K8sControlplaneCommonMessageFieldSet{
				Message: "foo",
			},
			inputSchedulerFieldSet: googlecloudlogk8scontrolplane_contract.K8sSchedulerComponentFieldSet{
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
			inputComponentField: googlecloudlogk8scontrolplane_contract.K8sControlplaneComponentFieldSet{
				ClusterName:   "test-cluster",
				ComponentName: "scheduler",
			},
			inputMessageField: googlecloudlogk8scontrolplane_contract.K8sControlplaneCommonMessageFieldSet{
				Message: "foo",
			},
			inputSchedulerFieldSet: googlecloudlogk8scontrolplane_contract.K8sSchedulerComponentFieldSet{},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantCompTimeline).
					HasNoEvent(wantPodTimeline)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)
			l := log.NewLogWithFieldSetsForTest(&tc.inputComponentField, &tc.inputSchedulerFieldSet, &tc.inputMessageField)
			mapper := &SchedulerTimelineMapper{}
			cs, _, err := mapper.ProcessLogByGroup(ctx, l, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLogByGroup() returned an unexpected error, err=%v", err)
			}
			tc.assert(t, ctx, cs)
		})
	}
}
