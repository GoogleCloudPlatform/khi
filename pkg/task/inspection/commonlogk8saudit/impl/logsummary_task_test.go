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

package commonlogk8saudit_impl

import (
	"testing"
	"time"

	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testchangeset"
)

func TestK8sAuditLogIngester_ProcessLog(t *testing.T) {
	testTime := time.Date(2023, 10, 26, 10, 0, 0, 0, time.UTC)

	testCases := []struct {
		name   string
		input  *log.Log
		assert func(t *testing.T, cs *khifilev6.LogChangeSet)
	}{
		{
			name: "successful info log ingestion",
			input: log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{Timestamp: testTime},
				&commonlogk8saudit_contract.K8sAuditLogFieldSet{
					APIVersion:   "core/v1",
					PluralKind:   "pods",
					Namespace:    "default",
					ResourceName: "test-pod",
					RequestURI:   "/api/v1/namespaces/default/pods/test-pod",
					Verb:         commonlogk8saudit_contract.VerbCreate,
					IsError:      false,
				},
			),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasTimestamp(testTime).
					HasSeverity(inspectioncore_contract.SeverityInfo).
					HasLogType(commonlogk8saudit_contract.LogTypeAudit).
					HasSummary("Create /api/v1/namespaces/default/pods/test-pod")
			},
		},
		{
			name: "error log ingestion",
			input: log.NewLogWithFieldSetsForTest(
				&log.CommonFieldSet{Timestamp: testTime},
				&commonlogk8saudit_contract.K8sAuditLogFieldSet{
					APIVersion:    "core/v1",
					PluralKind:    "pods",
					Namespace:     "default",
					ResourceName:  "test-pod",
					RequestURI:    "/api/v1/namespaces/default/pods/test-pod",
					Verb:          commonlogk8saudit_contract.VerbCreate,
					IsError:       true,
					StatusCode:    409,
					StatusMessage: "Conflict",
				},
			),
			assert: func(t *testing.T, cs *khifilev6.LogChangeSet) {
				testchangeset.AssertLog(t, cs).
					HasTimestamp(testTime).
					HasSeverity(inspectioncore_contract.SeverityError).
					HasLogType(commonlogk8saudit_contract.LogTypeAudit).
					HasSummary("【Conflict(409)】Create /api/v1/namespaces/default/pods/test-pod")
			},
		},
	}

	ingester := &k8sAuditLogIngester{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cs, err := ingester.ProcessLog(t.Context(), tc.input)
			if err != nil {
				t.Fatalf("ProcessLog() failed: %v", err)
			}
			tc.assert(t, cs)
		})
	}
}
