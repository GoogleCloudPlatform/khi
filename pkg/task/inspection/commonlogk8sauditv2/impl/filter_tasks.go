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
	"context"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
)

// SuccessLogFilterTask filters out non-success logs.
var SuccessLogFilterTask = inspectiontaskbase.NewLogFilterTask(
	commonlogk8sauditv2_contract.SuccessLogFilterTaskID,
	commonlogk8sauditv2_contract.K8sAuditLogProviderRef,
	func(ctx context.Context, l *log.Log) bool {
		return !log.MustGetFieldSet(l, &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{}).IsError
	},
)

// NonSuccessLogFilterTask filters out success logs.
var NonSuccessLogFilterTask = inspectiontaskbase.NewLogFilterTask(
	commonlogk8sauditv2_contract.NonSuccessLogFilterTaskID,
	commonlogk8sauditv2_contract.K8sAuditLogProviderRef,
	func(ctx context.Context, l *log.Log) bool {
		return log.MustGetFieldSet(l, &commonlogk8sauditv2_contract.K8sAuditLogFieldSet{}).IsError
	},
)
