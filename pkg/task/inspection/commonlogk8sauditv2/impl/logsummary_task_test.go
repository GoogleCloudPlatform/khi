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

package commonlogk8sauditv2_impl

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
)

func TestLogSummaryHistoryModifierSetting_getLogSummary(t *testing.T) {
	testCases := []struct {
		desc  string
		input *commonlogk8sauditv2_contract.K8sAuditLogFieldSet
		want  string
	}{
		{
			desc: "",
			input: &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{
				IsError:       true,
				StatusMessage: "test",
				StatusCode:    404,
				K8sOperation: &model.KubernetesObjectOperation{
					Verb: enum.RevisionVerbDelete,
				},
				RequestURI: "/test",
			},
			want: "【test(404)】Delete /test",
		},
		{
			desc: "",
			input: &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{
				IsError:       false,
				StatusMessage: "test",
				StatusCode:    200,
				K8sOperation: &model.KubernetesObjectOperation{
					Verb: enum.RevisionVerbDelete,
				},
				RequestURI: "/test",
			},
			want: "Delete /test",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			setting := &logSummaryHistoryModifierSetting{}
			got := setting.logSummary(tc.input)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
