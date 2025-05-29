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

package componentparser

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/log"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
)

func TestPodRelatedLogsToResourcePath(t *testing.T) {
	testCases := []struct {
		testName      string
		inputLog      string
		expectedPath  string
		expectedError bool
	}{
		{
			testName: "Normal case",
			inputLog: `insertId: zi5s4vf0ywd0ou9w
jsonPayload:
  message: '"Add event for scheduled pod" pod="1-2-deployment-update/nginx-deployment-non-surge-5455c7f485-2cfpz"'
  pid: "11"
logName: projects/khi-testing/logs/container.googleapis.com%2Fscheduler
receiveTimestamp: "2024-08-19T10:31:14.802511598Z"
resource:
  labels:
    cluster_name: gke-basic-1
    component_location: us-central1-a
    component_name: scheduler
    location: us-central1-a
    project_id: khi-testing
  type: k8s_control_plane_component
severity: INFO
sourceLocation:
  file: eventhandlers.go
  line: "197"
timestamp: "2024-08-19T10:31:12.865780Z"`,
			expectedPath:  "core/v1#pod#1-2-deployment-update#nginx-deployment-non-surge-5455c7f485-2cfpz",
			expectedError: false,
		},
		{
			testName: "Missing pod field",
			inputLog: `insertId: 1hyqhhwvyaqo49zm
jsonPayload:
  message: To require authentication configuration lookup to succeed, set --authentication-tolerate-lookup-failure=false
  pid: "11"
logName: projects/khi-testing/logs/container.googleapis.com%2Fscheduler
receiveTimestamp: "2024-08-19T10:07:36.527462487Z"
resource:
  labels:
    cluster_name: gke-basic-1
    component_location: us-central1-a
    component_name: scheduler
    location: us-central1-a
    project_id: khi-testing
  type: k8s_control_plane_component
severity: WARNING
sourceLocation:
  file: authentication.go
  line: "370"
timestamp: "2024-08-19T10:06:31.833958Z"`,
			expectedPath:  "",
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			parser := &SchedulerComponentParser{}
			l := testlog.MustLogFromYAML(tc.inputLog, &log.GCPCommonFieldSetReader{}, &log.GCPMainMessageFieldSetReader{})
			path, err := parser.podRelatedLogsToResourcePath(context.Background(), l)
			if tc.expectedError {
				if err == nil {
					t.Errorf("expected an error but no error returned")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
			if diff := cmp.Diff(tc.expectedPath, path.Path); diff != "" {
				t.Errorf("the result path is not valid:\nInput:\n%v\nActual:\n%s\nExpected:\n%s", tc.inputLog, path.Path, tc.expectedPath)
			}
		})
	}
}
