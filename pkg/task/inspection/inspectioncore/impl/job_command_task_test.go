// Copyright 2026 Google LLC
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

package inspectioncore_impl

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGenerateJobModeCommand(t *testing.T) {
	testCases := []struct {
		name            string
		inspectionType  string
		enabledFeatures []string
		taskInput       any
		want            string
		wantErr         bool
	}{
		{
			name:            "basic command without features and input",
			inspectionType:  "gke",
			enabledFeatures: []string{},
			taskInput:       nil,
			want: `./khi \
  --job-mode \
  --job-inspection-type="gke" \
  --job-inspection-features="" \
  --job-inspection-values='' \
  --job-export-destination="output.khi"`,
			wantErr: false,
		},
		{
			name:            "command with unsorted features",
			inspectionType:  "gke",
			enabledFeatures: []string{"audit", "cluster", "autoscaler"},
			taskInput:       nil,
			want: `./khi \
  --job-mode \
  --job-inspection-type="gke" \
  --job-inspection-features="audit,autoscaler,cluster" \
  --job-inspection-values='' \
  --job-export-destination="output.khi"`,
			wantErr: false,
		},
		{
			name:            "command with task inputs",
			inspectionType:  "composer",
			enabledFeatures: []string{"composer-logs"},
			taskInput: map[string]any{
				"project": "my-project",
				"cluster": "my-cluster",
			},
			want: `./khi \
  --job-mode \
  --job-inspection-type="composer" \
  --job-inspection-features="composer-logs" \
  --job-inspection-values='{
  "cluster": "my-cluster",
  "project": "my-project"
}' \
  --job-export-destination="output.khi"`,
			wantErr: false,
		},
		{
			name:            "command with task inputs containing single quotes",
			inspectionType:  "composer",
			enabledFeatures: []string{"composer-logs"},
			taskInput: map[string]any{
				"project": "my'project",
			},
			want: `./khi \
  --job-mode \
  --job-inspection-type="composer" \
  --job-inspection-features="composer-logs" \
  --job-inspection-values='{
  "project": "my'\''project"
}' \
  --job-export-destination="output.khi"`,
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := GenerateJobModeCommand(tc.inspectionType, tc.enabledFeatures, tc.taskInput)
			if (err != nil) != tc.wantErr {
				t.Fatalf("GenerateJobModeCommand() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr {
				if diff := cmp.Diff(tc.want, got); diff != "" {
					t.Errorf("GenerateJobModeCommand() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
