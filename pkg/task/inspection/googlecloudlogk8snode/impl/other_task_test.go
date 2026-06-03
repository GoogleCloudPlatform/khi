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

package googlecloudlogk8snode_impl

import (
	"context"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	googlecloudlogk8snode_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8snode/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
)

func TestOtherLogLogToTimelineMapper_ProcessLogByGroup(t *testing.T) {
	mapper := &otherNodeLogLogToTimelineMapperSetting{
		StartingMessagesByComponent: map[string]string{
			"component-A": "component-A start",
		},
		TerminatingMessagesByComponent: map[string]string{
			"component-A": "component-A terminate",
		},
	}
	testTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// 1. Initialize Builder
	builder := khifilev6.NewBuilder()

	testCases := []struct {
		desc         string
		inputMessage string
		component    string
		nodeName     string
		assert       func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet)
	}{
		{
			desc:         "starting log for component-A adds Create revision and event",
			inputMessage: "component-A start",
			component:    "component-A",
			nodeName:     "node-1",
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				wantNodePath := MustK8sNodeTimeline(ctx, "test-cluster", "node-1")
				wantComponentPath := googlecloudlogk8snode_contract.MustNodeComponentTimeline(ctx, wantNodePath, "component-A")

				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantComponentPath).
					HasRevision(wantComponentPath, &khifilev6.StagingRevision{
						VerbType:    commonlogk8saudit_contract.VerbCreate,
						StateType:   googlecloudlogk8snode_contract.RevisionStateComponentRunning,
						Principal:   "component-A",
						ChangedTime: testTime,
					})
			},
		},
		{
			desc:         "terminating log for component-A adds Delete revision and event",
			inputMessage: "component-A terminate",
			component:    "component-A",
			nodeName:     "node-1",
			assert: func(t *testing.T, ctx context.Context, cs *khifilev6.TimelineChangeSet) {
				wantNodePath := MustK8sNodeTimeline(ctx, "test-cluster", "node-1")
				wantComponentPath := googlecloudlogk8snode_contract.MustNodeComponentTimeline(ctx, wantNodePath, "component-A")

				testchangeset.AssertTimeline(t, cs).
					HasEvent(wantComponentPath).
					HasRevision(wantComponentPath, &khifilev6.StagingRevision{
						VerbType:    commonlogk8saudit_contract.VerbDelete,
						StateType:   googlecloudlogk8snode_contract.RevisionStateComponentTerminated,
						Principal:   "component-A",
						ChangedTime: testTime,
					})
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			parser := logutil.FallbackRawTextLogParser{}
			message := parser.TryParse(tc.inputMessage)

			l := log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{Timestamp: testTime},
				&googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
					Component: tc.component,
					NodeName:  tc.nodeName,
					Message:   message,
				},
			)

			// 2. Setup context with SAME builder and mock tasks
			ctx := khictx.WithValue(t.Context(), inspectioncore_contract.Builder, builder)
			ctx = tasktest.WithTaskResult(ctx, googlecloudk8scommon_contract.InputClusterNameTaskID.Ref(), "test-cluster")

			cs, _, err := mapper.ProcessLogByGroup(ctx, l, struct{}{})
			if err != nil {
				t.Fatalf("ProcessLogByGroup() returned unexpected error: %v", err)
			}

			tc.assert(t, ctx, cs)
		})
	}
}
