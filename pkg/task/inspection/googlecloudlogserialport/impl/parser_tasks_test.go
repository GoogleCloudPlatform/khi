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

package googlecloudlogserialport_impl

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudlogserialport_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogserialport/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
)

func TestLogToTimelineMapperTask(t *testing.T) {
	testCases := []struct {
		desc     string
		fieldSet googlecloudlogserialport_contract.GCESerialPortLogFieldSet
		asserter []testchangeset.ChangeSetAsserter
	}{
		{
			desc: "with standard input",
			fieldSet: googlecloudlogserialport_contract.GCESerialPortLogFieldSet{
				Message:  "foo",
				NodeName: "node-name-bar",
				Port:     "serial_port_output_qux",
			},
			asserter: []testchangeset.ChangeSetAsserter{
				&testchangeset.MatchResourcePathSet{
					WantResourcePaths: []string{"core/v1#node#cluster-scope#node-name-bar#serial_port_output_qux"},
				},
				&testchangeset.HasEvent{
					ResourcePath: "core/v1#node#cluster-scope#node-name-bar#serial_port_output_qux",
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l := log.NewLogWithFieldSetsForTest(&tc.fieldSet)
			modifier := serialportLogToTimelineMapper{}
			cs := history.NewChangeSet(l)
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
