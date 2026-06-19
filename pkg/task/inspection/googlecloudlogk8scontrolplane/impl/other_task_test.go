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

func TestOtherLogToTimelineMapperTask(t *testing.T) {
	builder := khifilev6.NewBuilder()

	projectTimeline := builder.TimelineAccumulator.GetPath(nil, khifilev6.PathSegment{
		Name: "test-project",
		Type: googlecloudcommon_contract.TimelineTypeGCPProject,
	})
	gkeClusterTimeline := builder.TimelineAccumulator.GetPath(projectTimeline, khifilev6.PathSegment{
		Name: "test-cluster",
		Type: googlecloudcommon_contract.TimelineTypeGKE,
	})
	wantControlPlanesTimeline := builder.TimelineAccumulator.GetPath(gkeClusterTimeline, khifilev6.PathSegment{
		Name: "controlplanes",
		Type: googlecloudcommon_contract.TimelineTypeGKEControlPlanes,
	})
	wantCompTimeline := builder.TimelineAccumulator.GetPath(wantControlPlanesTimeline, khifilev6.PathSegment{
		Name: "apiserver",
		Type: googlecloudlogk8scontrolplane_contract.TimelineTypeControlPlaneComponent,
	})

	testCases := []struct {
		desc                string
		inputComponentField googlecloudlogk8scontrolplane_contract.K8sControlplaneComponentFieldSet
		inputMessageField   googlecloudlogk8scontrolplane_contract.K8sControlplaneCommonMessageFieldSet
		assert              func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet)
	}{
		{
			desc: "with standard message",
			inputComponentField: googlecloudlogk8scontrolplane_contract.K8sControlplaneComponentFieldSet{
				ProjectID:     "test-project",
				ClusterName:   "test-cluster",
				ComponentName: "apiserver",
			},
			inputMessageField: googlecloudlogk8scontrolplane_contract.K8sControlplaneCommonMessageFieldSet{
				Message: "foo",
			},
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantCompTimeline)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)
			l := log.NewLogWithFieldSetsForTest(&tc.inputComponentField, &tc.inputMessageField)
			mapper := &OtherTimelineMapper{}
			cs, _, err := mapper.ProcessLogByGroup(ctx, l, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLogByGroup() returned an unexpected error, err=%v", err)
			}
			tc.assert(t, ctx, cs)
		})
	}
}
