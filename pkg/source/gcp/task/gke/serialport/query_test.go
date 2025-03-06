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

package serialport

import (
	"fmt"
	"math/rand"
	"testing"

	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"

	gcp_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/gcp"
	"github.com/google/go-cmp/cmp"
)

func TestGenerateSerialPortQuery(t *testing.T) {
	testCases := []struct {
		name               string
		taskMode           int
		nodeNames          []string
		nodeNameSubstrings []string
		wantQuery          string
	}{
		{
			name:               "dryrun",
			taskMode:           inspection_task.TaskModeDryRun,
			nodeNames:          []string{"node-1", "node-2"},
			nodeNameSubstrings: []string{},
			wantQuery: `LOG_ID("serialconsole.googleapis.com%2Fserial_port_1_output") OR
LOG_ID("serialconsole.googleapis.com%2Fserial_port_2_output") OR
LOG_ID("serialconsole.googleapis.com%2Fserial_port_3_output") OR
LOG_ID("serialconsole.googleapis.com%2Fserial_port_debug_output")

-- instance name filters to be determined after audit log query

-- No node name substring filters are specified.`,
		},
		{
			name:               "with single node",
			taskMode:           inspection_task.TaskModeRun,
			nodeNames:          []string{"node-1"},
			nodeNameSubstrings: []string{},
			wantQuery: `LOG_ID("serialconsole.googleapis.com%2Fserial_port_1_output") OR
LOG_ID("serialconsole.googleapis.com%2Fserial_port_2_output") OR
LOG_ID("serialconsole.googleapis.com%2Fserial_port_3_output") OR
LOG_ID("serialconsole.googleapis.com%2Fserial_port_debug_output")

labels."compute.googleapis.com/resource_name"=("node-1")

-- No node name substring filters are specified.`,
		},
		{
			name:               "with multiple nodes",
			taskMode:           inspection_task.TaskModeRun,
			nodeNames:          []string{"node-1", "node-2", "node-3"},
			nodeNameSubstrings: []string{},
			wantQuery: `LOG_ID("serialconsole.googleapis.com%2Fserial_port_1_output") OR
LOG_ID("serialconsole.googleapis.com%2Fserial_port_2_output") OR
LOG_ID("serialconsole.googleapis.com%2Fserial_port_3_output") OR
LOG_ID("serialconsole.googleapis.com%2Fserial_port_debug_output")

labels."compute.googleapis.com/resource_name"=("node-1" OR "node-2" OR "node-3")

-- No node name substring filters are specified.`,
		},
		{
			name:               "with node name substring",
			taskMode:           inspection_task.TaskModeRun,
			nodeNames:          []string{"node-1", "node-2", "node-3"},
			nodeNameSubstrings: []string{"node-1"},
			wantQuery: `LOG_ID("serialconsole.googleapis.com%2Fserial_port_1_output") OR
LOG_ID("serialconsole.googleapis.com%2Fserial_port_2_output") OR
LOG_ID("serialconsole.googleapis.com%2Fserial_port_3_output") OR
LOG_ID("serialconsole.googleapis.com%2Fserial_port_debug_output")

labels."compute.googleapis.com/resource_name"=("node-1" OR "node-2" OR "node-3")

labels."compute.googleapis.com/resource_name":("node-1")`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			query := GenerateSerialPortQuery(tc.taskMode, tc.nodeNames, tc.nodeNameSubstrings)
			if diff := cmp.Diff(tc.wantQuery, query[0]); diff != "" {
				t.Errorf("the generated query is not matching with the expected query\n%s", diff)
			}
			err := gcp_test.IsValidLogQuery(t, query[0])
			if err != nil {
				t.Errorf("the generated query is invalid. error:%v", err)
			}
		})
	}
}

func TestMaximumNodeCountNotHittingQueryLengthLimit(t *testing.T) {
	nodeNames := []string{}
	for i := 0; i < MaxNodesPerQuery*2+1; i++ { // This query must be splitted with 3 sub groups.
		nodeNames = append(nodeNames, fmt.Sprintf(`gke-%s-%s-%s`, randomString(46), randomString(8), randomString(4)))
	}
	query := GenerateSerialPortQuery(inspection_task.TaskModeRun, nodeNames, []string{})
	if len(query) != 3 {
		t.Errorf("len(GenerateSerialPortQuery())=%d, want %d", len(query), 3)
	}
	for _, subquery := range query {
		err := gcp_test.IsValidLogQuery(t, subquery)
		if err != nil {
			t.Errorf("the generated query is invalid. error:%v", err)
		}
	}
}

func Test_generateNodeNameSubstringLogFilter(t *testing.T) {
	tests := []struct {
		name               string
		nodeNameSubstrings []string
		want               string
	}{
		{
			name:               "empty",
			nodeNameSubstrings: []string{},
			want:               "-- No node name substring filters are specified.",
		},
		{
			name:               "single",
			nodeNameSubstrings: []string{"substring1"},
			want:               `labels."compute.googleapis.com/resource_name":("substring1")`,
		},
		{
			name:               "multiple",
			nodeNameSubstrings: []string{"substring1", "substring2", "substring3"},
			want:               `labels."compute.googleapis.com/resource_name":("substring1" OR "substring2" OR "substring3")`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateNodeNameSubstringLogFilter(tt.nodeNameSubstrings)
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("generateNodeNameSubstringLogFilter() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func randomString(length int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	randomid := make([]rune, length)
	for i := range randomid {
		randomid[i] = letters[rand.Intn(len(letters))]
	}
	return string(randomid)
}
