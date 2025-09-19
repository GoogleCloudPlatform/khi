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
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourceinfo/resourcelease"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/impl/recorder"
)

func Register(manager *recorder.RecorderTaskManager) error {
	manager.AddRecorder("sneg-fields", []taskid.UntypedTaskReference{}, func(ctx context.Context, req *recorder.RecorderRequest) (any, error) {
		commonFieldSet := log.MustGetFieldSet(req.LogParseResult.Log, &log.CommonFieldSet{})
		// record node name for querying compute engine api later.
		req.Builder.ClusterResource.NEGs.TouchResourceLease(req.LogParseResult.Operation.Name, commonFieldSet.Timestamp, resourcelease.NewK8sResourceLeaseHolder(
			req.LogParseResult.Operation.PluralKind,
			req.LogParseResult.Operation.Namespace,
			req.LogParseResult.Operation.Name,
		))
		return nil, nil
	}, recorder.ResourceKindLogGroupFilter("servicenetworkendpointgroup"), recorder.AndLogFilter(recorder.OnlySucceedLogs(), recorder.OnlyWithResourceBody()))
	return nil
}
