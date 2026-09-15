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

package k8saudit_impl

import (
	"testing"

	inspectiontaskbasetest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbasetest"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil/testlog"
)

func TestSuccessLogFilterTask(t *testing.T) {
	inspectiontaskbasetest.AssertFilterTask(t, SuccessLogFilterTask, k8saudit.K8sAuditLogProviderRef, []inspectiontaskbasetest.FilterTaskTestCase{
		{
			Description: "success log",
			Log: testlog.NewMockLog(k8saudit.K8sAuditLogFieldSet{
				IsError: false,
			}),
			WantIncluded: true,
		},
		{
			Description: "non-success log",
			Log: testlog.NewMockLog(k8saudit.K8sAuditLogFieldSet{
				IsError: true,
			}),
			WantIncluded: false,
		},
	})
}

func TestNonSuccessLogFilterTask(t *testing.T) {
	inspectiontaskbasetest.AssertFilterTask(t, NonSuccessLogFilterTask, k8saudit.K8sAuditLogProviderRef, []inspectiontaskbasetest.FilterTaskTestCase{
		{
			Description: "success log",
			Log: testlog.NewMockLog(k8saudit.K8sAuditLogFieldSet{
				IsError: false,
			}),
			WantIncluded: false,
		},
		{
			Description: "non-success log",
			Log: testlog.NewMockLog(k8saudit.K8sAuditLogFieldSet{
				IsError: true,
			}),
			WantIncluded: true,
		},
	})
}
