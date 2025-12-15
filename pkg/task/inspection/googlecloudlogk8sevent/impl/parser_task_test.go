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

package googlecloudlogk8sevent_impl

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudlogk8sevent_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8sevent/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
)

func TestLogToTimelineMapperTask(t *testing.T) {
	testCases := []struct {
		desc      string
		input     googlecloudlogk8sevent_contract.KubernetesEventFieldSet
		asserters []testchangeset.ChangeSetAsserter
	}{
		{
			desc: "simple event",
			input: googlecloudlogk8sevent_contract.KubernetesEventFieldSet{
				ClusterName:  "test-cluster",
				APIVersion:   "apps/v1",
				ResourceKind: "deployment",
				Namespace:    "default",
				Resource:     "test-deployment",
				Reason:       "ScalingReplicaSet",
				Message:      "Scaled up replica set test-deployment-xyz to 3",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.MatchResourcePathSet{
					WantResourcePaths: []string{"apps/v1#deployment#default#test-deployment"},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l := log.NewLogWithFieldSetsForTest(&tc.input)
			cs := history.NewChangeSet(l)
			modifier := KubernetesEventLogToTimelineMapperSetting{}

			_, err := modifier.ProcessLogByGroup(t.Context(), l, cs, nil, struct{}{})
			if err != nil {
				t.Errorf("ProcessLogByGroup returned an unexpected error: %v", err)
			}

			for _, asserter := range tc.asserters {
				asserter.Assert(t, cs)
			}
		})
	}
}
