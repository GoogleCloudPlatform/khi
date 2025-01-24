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

package k8s_container

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/google/go-cmp/cmp"
)

func TestMetricsLogType(t *testing.T) {
	type testCase struct {
		Name     string
		Input    string
		Severity enum.Severity
	}
	testCases := []testCase{
		{
			Name:     "metrics-server error sample log",
			Input:    "2024-04-05T04:14:26.845Z\terror\tcollectors/node.go:159\tFailed to query apiserver for node data\t{\"kind\": \"receiver\", \"name\": \"kubenode\", \"error\":\"Get \\\"https://10.86.112.1:443/api/v1/nodes/gke-gke-basic-2-default-4af15592-q4rf?timeout=4.5s\\\":net/http: request canceled while waiting for connection (Client.Timeout exceeded while awaiting headers)\"}",
			Severity: enum.SeverityError,
		},
		{
			Name:     "metrics-server error sample warn log",
			Input:    "2024-04-05T04:38:01.905Z\twarn\tscrape/scrape.go:1340\tStale report failed\t{\"kind\": \"receiver\", \"name\": \"prometheus\", \"scrape_pool\": \"addons\", \"target\": \"http://10.224.2.4:19092/metrics\", \"err\": \"unable to find a target with job=addons, and instance=10.224.2.4:19092\"}",
			Severity: enum.SeverityWarning,
		},
		{
			Name:     "non metrics-server error sample log",
			Input:    "foo bar qux",
			Severity: enum.SeverityUnknown,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			parser := MetricsContainerLogSeverityParser{}
			actual := parser.TryParse(tc.Input)
			if diff := cmp.Diff(tc.Severity, actual); diff != "" {
				t.Errorf("severity is not matching with the expected value\n%s", diff)
			}
		})
	}
}
