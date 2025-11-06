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
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudlogk8scontrolplane_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontrolplane/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
)

func TestOtherHistoryModifierTask(t *testing.T) {
	testCases := []struct {
		desc                string
		inputComponentField googlecloudlogk8scontrolplane_contract.K8sControlplaneComponentFieldSet
		inputMessageField   googlecloudlogk8scontrolplane_contract.K8sControlplaneCommonMessageFieldSet
		asserters           []testchangeset.ChangeSetAsserter
	}{
		{
			desc: "with standard message",
			inputComponentField: googlecloudlogk8scontrolplane_contract.K8sControlplaneComponentFieldSet{
				ClusterName:   "test-cluster",
				ComponentName: "apiserver",
			},
			inputMessageField: googlecloudlogk8scontrolplane_contract.K8sControlplaneCommonMessageFieldSet{
				Message: "foo",
			},
			asserters: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasEvent{
					ResourcePath: "@Cluster#controlplane#cluster-scope#test-cluster#apiserver",
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l := log.NewLogWithFieldSetsForTest(&tc.inputComponentField, &tc.inputMessageField)
			modifier := otherHistoryModifierTaskSetting{}
			cs := history.NewChangeSet(l)
			_, err := modifier.ModifyChangeSetFromLog(t.Context(), l, cs, nil, struct{}{})
			if err != nil {
				t.Errorf("ModifyChangeSetFromLog() returned an unexpected error, err=%v", err)
			}
			for _, asserter := range tc.asserters {
				asserter.Assert(t, cs)
			}
		})
	}
}
