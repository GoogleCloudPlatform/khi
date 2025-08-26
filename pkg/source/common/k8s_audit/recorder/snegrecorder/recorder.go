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

package snegrecorder

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourceinfo/resourcelease"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/recorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/types"
)

func Register(manager *recorder.RecorderTaskManager) error {
	manager.AddRecorder("sneg-fields", []taskid.UntypedTaskReference{}, func(ctx context.Context, resourcePath string, currentLog *types.AuditLogParserInput, prevStateInGroup any, cs *history.ChangeSet, builder *history.Builder) (any, error) {
		commonFieldSet := log.MustGetFieldSet(currentLog.Log, &log.CommonFieldSet{})
		// record node name for querying compute engine api later.
		builder.ClusterResource.NEGs.TouchResourceLease(currentLog.Operation.Name, commonFieldSet.Timestamp, resourcelease.NewK8sResourceLeaseHolder(
			currentLog.Operation.PluralKind,
			currentLog.Operation.Namespace,
			currentLog.Operation.Name,
		))
		return nil, nil
	}, recorder.ResourceKindLogGroupFilter("servicenetworkendpointgroup"), recorder.AndLogFilter(recorder.OnlySucceedLogs(), recorder.OnlyWithResourceBody()))
	return nil
}
