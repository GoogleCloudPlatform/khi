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

package noderecorder

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/impl/recorder"
)

func Register(manager *recorder.RecorderTaskManager) error {
	manager.AddRecorder("node-fields", []taskid.UntypedTaskReference{}, func(ctx context.Context, req *recorder.RecorderRequest) (any, error) {
		// record node name for querying compute engine api later.
		req.Builder.ClusterResource.AddNode(req.LogParseResult.Operation.Name)
		return nil, nil
	}, recorder.ResourceKindLogGroupFilter("node"), recorder.AndLogFilter(recorder.OnlySucceedLogs(), recorder.OnlyWithResourceBody()))
	return nil
}
