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

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudlogk8snode_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8snode/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
)

func TestOtherLogHistoryModifier(t *testing.T) {
	histoyModifier := otherNodeLogHistoryModifierSetting{
		StartingMessagesByComponent: map[string]string{
			"component-A": "component-A start",
		},
		TerminatingMessagesByComponent: map[string]string{
			"component-A": "component-A terminate",
		},
	}
	testTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	testCase := []struct {
		desc                 string
		inputMessage         string
		inputNodeLogFieldSet *googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet
		asserter             []testchangeset.ChangeSetAsserter
	}{
		{
			desc:         "starting log for component-A",
			inputMessage: "component-A start",
			inputNodeLogFieldSet: &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
				Component: "component-A",
				NodeName:  "node-1",
			},
			asserter: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#node#cluster-scope#node-1#component-A",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbCreate,
						State:      enum.RevisionStateExisting,
						Requestor:  "component-A",
						ChangeTime: testTime,
					},
				},
				&testchangeset.HasEvent{
					ResourcePath: "core/v1#node#cluster-scope#node-1#component-A",
				},
				&testchangeset.HasLogSummary{
					WantLogSummary: "component-A start",
				},
			},
		},
		{
			desc:         "terminating log for component-A",
			inputMessage: "component-A terminate",
			inputNodeLogFieldSet: &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
				Component: "component-A",
				NodeName:  "node-1",
			},
			asserter: []testchangeset.ChangeSetAsserter{
				&testchangeset.HasRevision{
					ResourcePath: "core/v1#node#cluster-scope#node-1#component-A",
					WantRevision: history.StagingResourceRevision{
						Verb:       enum.RevisionVerbDelete,
						State:      enum.RevisionStateDeleted,
						Requestor:  "component-A",
						ChangeTime: testTime,
					},
				},
				&testchangeset.HasEvent{
					ResourcePath: "core/v1#node#cluster-scope#node-1#component-A",
				},
				&testchangeset.HasLogSummary{
					WantLogSummary: "component-A terminate",
				},
			},
		},
		{
			desc:         "no matching log",
			inputMessage: "component-A doing something",
			inputNodeLogFieldSet: &googlecloudlogk8snode_contract.K8sNodeLogCommonFieldSet{
				Component: "component-A",
				NodeName:  "node-1",
			},
			asserter: []testchangeset.ChangeSetAsserter{
				&testchangeset.MatchResourcePathSet{
					WantResourcePaths: []string{
						"core/v1#node#cluster-scope#node-1#component-A",
					},
				},
				&testchangeset.HasEvent{
					ResourcePath: "core/v1#node#cluster-scope#node-1#component-A",
				},
				&testchangeset.HasLogSummary{
					WantLogSummary: "component-A doing something",
				},
			},
		},
	}

	for _, tc := range testCase {
		t.Run(tc.desc, func(t *testing.T) {
			parser := logutil.FallbackRawTextLogParser{}
			message := parser.TryParse(tc.inputMessage)
			tc.inputNodeLogFieldSet.Message = message
			l := log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{Timestamp: testTime},
				tc.inputNodeLogFieldSet,
			)
			cs := history.NewChangeSet(l)
			_, err := histoyModifier.ModifyChangeSetFromLog(context.Background(), l, cs, nil, struct{}{})
			if err != nil {
				t.Fatalf("ModifyChangeSetFromLog() error = %v", err)
			}
			for _, asserter := range tc.asserter {
				asserter.Assert(t, cs)
			}
		})

	}
}
