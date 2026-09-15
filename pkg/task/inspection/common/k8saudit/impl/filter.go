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
	"context"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
)

// SuccessLogFilterTask filters out non-success logs.
var SuccessLogFilterTask = inspectiontaskbase.NewLogFilterTaskWithDependencies(
	k8saudit.SuccessLogFilterTaskID,
	k8saudit.K8sAuditLogProviderRef,
	[]coretask.Dependency{k8saudit.K8sAuditLogErrorExtractorRef.Ref(coretask.FromActiveGraph)},
	func(ctx context.Context, l *log.Log) bool {
		isError, _ := k8saudit.ExtractK8sAuditLogError(ctx, l.NodeReader)
		return !isError
	},
)

// NonSuccessLogFilterTask filters out success logs.
var NonSuccessLogFilterTask = inspectiontaskbase.NewLogFilterTaskWithDependencies(
	k8saudit.NonSuccessLogFilterTaskID,
	k8saudit.K8sAuditLogProviderRef,
	[]coretask.Dependency{k8saudit.K8sAuditLogErrorExtractorRef.Ref(coretask.FromActiveGraph)},
	func(ctx context.Context, l *log.Log) bool {
		isError, _ := k8saudit.ExtractK8sAuditLogError(ctx, l.NodeReader)
		return isError
	},
)
