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

	inspectiontaskbasetest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbasetest"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
)

func TestSuccessLogFilterTask(t *testing.T) {
	inspectiontaskbasetest.AssertFilterTask(t, SuccessLogFilterTask, commonlogk8sauditv2_contract.K8sAuditLogProviderRef, []inspectiontaskbasetest.FilterTaskTestCase{
		{
			Description: "success log",
			LogFields: []log.FieldSet{
				&commonlogk8sauditv2_contract.K8sAuditLogFieldSet{
					IsError: false,
				},
			},
			WantIncluded: true,
		},
		{
			Description: "non-success log",
			LogFields: []log.FieldSet{
				&commonlogk8sauditv2_contract.K8sAuditLogFieldSet{
					IsError: true,
				},
			},
			WantIncluded: false,
		},
	})
}

func TestNonSuccessLogFilterTask(t *testing.T) {
	inspectiontaskbasetest.AssertFilterTask(t, NonSuccessLogFilterTask, commonlogk8sauditv2_contract.K8sAuditLogProviderRef, []inspectiontaskbasetest.FilterTaskTestCase{
		{
			Description: "success log",
			LogFields: []log.FieldSet{
				&commonlogk8sauditv2_contract.K8sAuditLogFieldSet{
					IsError: false,
				},
			},
			WantIncluded: false,
		},
		{
			Description: "non-success log",
			LogFields: []log.FieldSet{
				&commonlogk8sauditv2_contract.K8sAuditLogFieldSet{
					IsError: true,
				},
			},
			WantIncluded: true,
		},
	})
}
