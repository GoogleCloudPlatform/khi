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

package ossk8s_impl

import (
	"context"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	ossk8s "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/oss/k8s"
)

var EventAuditLogFilterTask = inspectiontaskbase.NewLogFilterTask(
	ossk8s.EventAuditLogFilterTaskID,
	ossk8s.AuditLogFileReaderTaskID.Ref(),
	func(ctx context.Context, l *log.Log) bool {
		isEvent, _ := ossk8s.ExtractOSSK8sIsEventAuditLog(l.NodeReader)
		return isEvent
	},
)
