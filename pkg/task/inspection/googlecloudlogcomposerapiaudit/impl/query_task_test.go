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

package googlecloudlogcomposerapiaudit_impl

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGenerateComposerAuditQuery(t *testing.T) {
	testCases := []struct {
		name            string
		projectID       string
		location        string
		environmentName string
		want            string
	}{
		{
			name:            "with location and environment name",
			projectID:       "my-project",
			location:        "us-central1",
			environmentName: "env-1",
			want: `(log_id("cloudaudit.googleapis.com/activity") OR log_id("cloudaudit.googleapis.com/data_access"))
resource.type="cloud_composer_environment"
resource.labels.project_id="my-project"
resource.labels.location="us-central1"
resource.labels.environment_name="env-1"
protoPayload.serviceName="composer.googleapis.com"
`,
		},
		{
			name:            "empty location and environment name",
			projectID:       "my-project",
			location:        "",
			environmentName: "",
			want: `(log_id("cloudaudit.googleapis.com/activity") OR log_id("cloudaudit.googleapis.com/data_access"))
resource.type="cloud_composer_environment"
resource.labels.project_id="my-project"
protoPayload.serviceName="composer.googleapis.com"
`,
		},
		{
			name:            "location all",
			projectID:       "my-project",
			location:        "all",
			environmentName: "env-1",
			want: `(log_id("cloudaudit.googleapis.com/activity") OR log_id("cloudaudit.googleapis.com/data_access"))
resource.type="cloud_composer_environment"
resource.labels.project_id="my-project"
resource.labels.environment_name="env-1"
protoPayload.serviceName="composer.googleapis.com"
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := GenerateComposerAuditQuery(tc.projectID, tc.location, tc.environmentName)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GenerateComposerAuditQuery() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
