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

package googlecloudlogk8scontainer_impl

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudlogk8scontainer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontainer/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
)

func TestLogToTimelineMapperTask(t *testing.T) {
	testCases := []struct {
		desc     string
		input    googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet
		asserter []testchangeset.ChangeSetAsserter
	}{
		{
			desc: "simple container log",
			input: googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{
				Namespace:     "test-namespace",
				PodName:       "test-pod",
				ContainerName: "test-container",
				Message:       "test message",
			},
			asserter: []testchangeset.ChangeSetAsserter{
				&testchangeset.MatchResourcePathSet{
					WantResourcePaths: []string{"core/v1#pod#test-namespace#test-pod#test-container"},
				},
				&testchangeset.HasEvent{
					ResourcePath: "core/v1#pod#test-namespace#test-pod#test-container",
				},
				&testchangeset.HasLogSummary{
					WantLogSummary: "test message",
				},
			},
		},
		{
			desc: "container log with empty message",
			input: googlecloudlogk8scontainer_contract.K8sContainerLogFieldSet{
				Namespace:     "test-namespace",
				PodName:       "test-pod",
				ContainerName: "test-container",
				Message:       "",
			},
			asserter: []testchangeset.ChangeSetAsserter{
				&testchangeset.MatchResourcePathSet{
					WantResourcePaths: []string{"core/v1#pod#test-namespace#test-pod#test-container"},
				},
				&testchangeset.HasEvent{
					ResourcePath: "core/v1#pod#test-namespace#test-pod#test-container",
				},
				&testchangeset.HasLogSummary{
					WantLogSummary: "",
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l := log.NewLogWithFieldSetsForTest(&tc.input)
			cs := history.NewChangeSet(l)
			modifier := containerLogLogToTimelineMapperSetting{}

			_, err := modifier.ProcessLogByGroup(t.Context(), l, cs, nil, struct{}{})

			if err != nil {
				t.Errorf("ProcessLogByGroup() returned an unexpected error, err=%v", err)
			}

			for _, asserter := range tc.asserter {
				asserter.Assert(t, cs)
			}
		})
	}
}
