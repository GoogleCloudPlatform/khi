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
	"github.com/GoogleCloudPlatform/khi/pkg/common/patternfinder"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogk8scontrolplane_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontrolplane/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
)

func TestControllerManagerLogToTimelineMapperTask(t *testing.T) {
	builder := khifilev6.NewBuilder()

	gkeClusterTimeline := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
		Name: "test-cluster",
		Type: googlecloudcommon_contract.TimelineTypeGKE,
	})
	wantCompTimeline := builder.TimelineAccumulator.GetPath(gkeClusterTimeline, khifilev6.PathSegment{
		Name: "deployment-controller(controller-manager)",
		Type: googlecloudlogk8scontrolplane_contract.TimelineTypeControlPlaneComponent,
	})
	wantControlManagerTimeline := builder.TimelineAccumulator.GetPath(gkeClusterTimeline, khifilev6.PathSegment{
		Name: "controller-manager",
		Type: googlecloudlogk8scontrolplane_contract.TimelineTypeControlPlaneComponent,
	})

	clusterTimeline := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
		Name: "test-cluster",
		Type: inspectioncore_contract.TimelineTypeK8sCluster,
	})
	corev1Timeline := builder.TimelineAccumulator.GetPath(clusterTimeline, khifilev6.PathSegment{
		Name: "core/v1",
		Type: inspectioncore_contract.TimelineTypeAPIVersion,
	})
	podKindTimeline := builder.TimelineAccumulator.GetPath(corev1Timeline, khifilev6.PathSegment{
		Name: "pod",
		Type: inspectioncore_contract.TimelineTypeKind,
	})
	nsTimeline := builder.TimelineAccumulator.GetPath(podKindTimeline, khifilev6.PathSegment{
		Name: "default",
		Type: inspectioncore_contract.TimelineTypeNamespace,
	})
	wantPodTimeline := builder.TimelineAccumulator.GetPath(nsTimeline, khifilev6.PathSegment{
		Name: "pod-foo",
		Type: inspectioncore_contract.TimelineTypeResource,
	})

	nodeKindTimeline := builder.TimelineAccumulator.GetPath(corev1Timeline, khifilev6.PathSegment{
		Name: "node",
		Type: inspectioncore_contract.TimelineTypeKind,
	})
	nodeNamespaceTimeline := builder.TimelineAccumulator.GetPath(nodeKindTimeline, khifilev6.PathSegment{
		Name: "cluster-scope",
		Type: inspectioncore_contract.TimelineTypeNamespace,
	})
	wantNodeTimeline := builder.TimelineAccumulator.GetPath(nodeNamespaceTimeline, khifilev6.PathSegment{
		Name: "node-1",
		Type: inspectioncore_contract.TimelineTypeResource,
	})

	testCases := []struct {
		desc                           string
		inputComponentField            googlecloudlogk8scontrolplane_contract.K8sControlplaneComponentFieldSet
		inputMessageField              googlecloudlogk8scontrolplane_contract.K8sControlplaneCommonMessageFieldSet
		inputControllerManagerFieldSet googlecloudlogk8scontrolplane_contract.K8sControllerManagerComponentFieldSet
		assert                         func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet)
	}{
		{
			desc: "with standard input",
			inputComponentField: googlecloudlogk8scontrolplane_contract.K8sControlplaneComponentFieldSet{
				ClusterName:   "test-cluster",
				ComponentName: "controller-manager",
			},
			inputMessageField: googlecloudlogk8scontrolplane_contract.K8sControlplaneCommonMessageFieldSet{
				Message: "foo",
			},
			inputControllerManagerFieldSet: googlecloudlogk8scontrolplane_contract.K8sControllerManagerComponentFieldSet{
				Controller: "deployment-controller",
				AssociatedResources: []resourcepath.ResourcePath{
					resourcepath.Pod("default", "pod-foo"),
					resourcepath.Node("node-1"),
				},
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantCompTimeline).
					HasEvent(wantPodTimeline).
					HasEvent(wantNodeTimeline)
			},
		},
		{
			desc: "with unknown controller input",
			inputComponentField: googlecloudlogk8scontrolplane_contract.K8sControlplaneComponentFieldSet{
				ClusterName:   "test-cluster",
				ComponentName: "controller-manager",
			},
			inputMessageField: googlecloudlogk8scontrolplane_contract.K8sControlplaneCommonMessageFieldSet{
				Message: "foo",
			},
			inputControllerManagerFieldSet: googlecloudlogk8scontrolplane_contract.K8sControllerManagerComponentFieldSet{
				Controller: "",
				AssociatedResources: []resourcepath.ResourcePath{
					resourcepath.Pod("default", "pod-foo"),
					resourcepath.Node("node-1"),
				},
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantControlManagerTimeline).
					HasEvent(wantPodTimeline).
					HasEvent(wantNodeTimeline)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)
			finder := patternfinder.NewTriePatternFinder[*commonlogk8saudit_contract.ResourceIdentity]()
			ctx = tasktest.WithTaskResult(ctx, commonlogk8saudit_contract.ResourceUIDPatternFinderTaskID.Ref(), finder)

			l := log.NewLogWithFieldSetsForTest(&tc.inputComponentField, &tc.inputControllerManagerFieldSet, &tc.inputMessageField)
			mapper := &ControllerManagerTimelineMapper{}
			cs, _, err := mapper.ProcessLogByGroup(ctx, l, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLogByGroup() returned an unexpected error, err=%v", err)
			}
			tc.assert(t, ctx, cs)
		})
	}
}
