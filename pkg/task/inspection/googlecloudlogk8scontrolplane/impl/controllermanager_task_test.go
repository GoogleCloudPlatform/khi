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
	ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)

	gkeClusterTimeline := googlecloudcommon_contract.MustGKEClusterTimeline(ctx, "test-cluster")
	wantCompTimeline := builder.TimelineAccumulator.GetPath(gkeClusterTimeline, khifilev6.PathSegment{
		Name: "deployment-controller(controller-manager)",
		Type: googlecloudlogk8scontrolplane_contract.TimelineTypeControlPlaneComponent,
	})
	wantControlManagerTimeline := builder.TimelineAccumulator.GetPath(gkeClusterTimeline, khifilev6.PathSegment{
		Name: "controller-manager",
		Type: googlecloudlogk8scontrolplane_contract.TimelineTypeControlPlaneComponent,
	})

	k8sClusterTimeline := commonlogk8saudit_contract.MustK8sClusterTimeline(ctx, "test-cluster")
	corev1Timeline := commonlogk8saudit_contract.MustK8sAPIVersionTimeline(ctx, k8sClusterTimeline, "core/v1")
	podKindTimeline := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, corev1Timeline, "pod")
	nsTimeline := commonlogk8saudit_contract.MustK8sNamespaceTimeline(ctx, podKindTimeline, "default")
	wantPodTimeline := commonlogk8saudit_contract.MustK8sNamespacedResourceTimeline(ctx, nsTimeline, "pod-foo")

	nodeKindTimeline := commonlogk8saudit_contract.MustK8sKindTimeline(ctx, corev1Timeline, "node")
	wantNodeTimeline := commonlogk8saudit_contract.MustK8sClusterScopeResourceTimeline(ctx, nodeKindTimeline, "node-1")

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
				AssociatedResources: []*commonlogk8saudit_contract.ResourceIdentity{
					{
						APIVersion: "core/v1",
						Kind:       "pod",
						Namespace:  "default",
						Name:       "pod-foo",
					},
					{
						APIVersion: "core/v1",
						Kind:       "node",
						Namespace:  "cluster-scope",
						Name:       "node-1",
					},
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
				AssociatedResources: []*commonlogk8saudit_contract.ResourceIdentity{
					{
						APIVersion: "core/v1",
						Kind:       "pod",
						Namespace:  "default",
						Name:       "pod-foo",
					},
					{
						APIVersion: "core/v1",
						Kind:       "node",
						Namespace:  "cluster-scope",
						Name:       "node-1",
					},
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
