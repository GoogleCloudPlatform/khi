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

package googlecloudlogcsm_impl

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
)

func TestCsmAccessLogsFilter(t *testing.T) {
	inputProjectID := "test-project"
	testCases := []struct {
		desc                string
		responseFlagsFilter *gcpqueryutil.SetFilterParseResult
		namespaceFilter     *gcpqueryutil.SetFilterParseResult
		want                string
	}{
		{
			desc: "basic filter",
			responseFlagsFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"UH", "UT"},
			},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"default"},
			},
			want: `LOG_ID("server-accesslog-stackdriver") OR LOG_ID("client-accesslog-stackdriver") 
labels.response_flag:("UH" OR "UT")
resource.labels.namespace_name:("default")
resource.labels.cluster_name="test-project"`,
		},
		{
			desc: "response flags subtractive filter",
			responseFlagsFilter: &gcpqueryutil.SetFilterParseResult{
				Subtractives: []string{"-"},
				SubtractMode: true,
			},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"default"},
			},
			want: `LOG_ID("server-accesslog-stackdriver") OR LOG_ID("client-accesslog-stackdriver") 
-labels.response_flag:("-")
resource.labels.namespace_name:("default")
resource.labels.cluster_name="test-project"`,
		},
		{
			desc: "namespace cluster-scoped filter",
			responseFlagsFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"UH"},
			},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"#cluster-scoped"},
			},
			want: `LOG_ID("server-accesslog-stackdriver") OR LOG_ID("client-accesslog-stackdriver") 
labels.response_flag:("UH")
resource.labels.namespace_name="" -- Invalid: No namespaces are remained to filter for CSM access log.
resource.labels.cluster_name="test-project"`,
		},
		{
			desc: "namespace cluster-scoped and specific namespaces filter",
			responseFlagsFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"UH"},
			},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"#cluster-scoped", "kube-system"},
			},
			want: `LOG_ID("server-accesslog-stackdriver") OR LOG_ID("client-accesslog-stackdriver") 
labels.response_flag:("UH")
resource.labels.namespace_name:("kube-system")
resource.labels.cluster_name="test-project"`,
		},
		{
			desc: "namespace namespaced-scoped filter",
			responseFlagsFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"UH"},
			},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"#namespaced"},
			},
			want: `LOG_ID("server-accesslog-stackdriver") OR LOG_ID("client-accesslog-stackdriver") 
labels.response_flag:("UH")
-- No namespace filter
resource.labels.cluster_name="test-project"`,
		},
		{
			desc: "namespace cluster-scoped and namespaced-scoped filter",
			responseFlagsFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"UH"},
			},
			namespaceFilter: &gcpqueryutil.SetFilterParseResult{
				Additives: []string{"#cluster-scoped", "#namespaced"},
			},
			want: `LOG_ID("server-accesslog-stackdriver") OR LOG_ID("client-accesslog-stackdriver") 
labels.response_flag:("UH")
-- No namespace filter
resource.labels.cluster_name="test-project"`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			got := csmAccessLogsFilter(inputProjectID, tc.responseFlagsFilter, tc.namespaceFilter)
			if got != tc.want {
				t.Errorf("csmAccessLogsFilter() got = %v, want %v", got, tc.want)
			}
		})
	}
}
