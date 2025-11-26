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
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
)

func TestNonSuccessLogHistoryModifierTaskSetting_AddEventForLog(t *testing.T) {
	testCases := []struct {
		desc     string
		input    *commonlogk8sauditv2_contract.K8sAuditLogFieldSet
		wantPath []string
	}{
		{
			desc: "error on name layer resource",
			input: &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{
				K8sOperation: &model.KubernetesObjectOperation{
					APIVersion:      "core/v1",
					PluralKind:      "pods",
					Namespace:       "kube-system",
					Name:            "kube-dns",
					SubResourceName: "",
				},
			},
			wantPath: []string{
				"core/v1#pod#kube-system#kube-dns",
			},
		},
		{
			desc: "error on subresource layer resource but not included in the exception map",
			input: &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{
				K8sOperation: &model.KubernetesObjectOperation{
					APIVersion:      "core/v1",
					PluralKind:      "pods",
					Namespace:       "kube-system",
					Name:            "kube-dns",
					SubResourceName: "binding",
				},
			},
			wantPath: []string{
				"core/v1#pod#kube-system#kube-dns#binding",
			},
		},
		{
			desc: "error on subresource layer resource and included in the exception map",
			input: &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{
				K8sOperation: &model.KubernetesObjectOperation{
					APIVersion:      "core/v1",
					PluralKind:      "pods",
					Namespace:       "kube-system",
					Name:            "kube-dns",
					SubResourceName: "status",
				},
			},
			wantPath: []string{
				"core/v1#pod#kube-system#kube-dns",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			l := log.NewLogWithFieldSetsForTest(tc.input)
			cs := history.NewChangeSet(l)

			setting := &nonSuccessLogHistoryModifierTaskSetting{
				subresourceMapToWriteToParent: map[string]struct{}{
					"status": {},
				},
			}
			err := setting.addEventForLog(l, cs)
			if err != nil {
				t.Fatalf("failed to add event for log: %v", err)
			}
			asserter := testchangeset.MatchResourcePathSet{
				WantResourcePaths: tc.wantPath,
			}

			asserter.Assert(t, cs)
		})
	}
}
