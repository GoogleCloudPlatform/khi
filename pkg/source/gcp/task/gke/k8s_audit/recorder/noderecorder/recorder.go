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

	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/recorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/types"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

func Register(manager *recorder.RecorderTaskManager) error {
	manager.AddRecorder("node-fields", []string{}, func(ctx context.Context, resourcePath string, currentLog *types.ResourceSpecificParserInput, prevStateInGroup any, cs *history.ChangeSet, builder *history.Builder, vs *task.VariableSet) (any, error) {
		// record node name for querying compute engine api later.
		builder.ClusterResource.AddNode(currentLog.Operation.Name)
		return nil, nil
	}, recorder.ResourceKindLogGroupFilter("node"), recorder.AndLogFilter(recorder.OnlySucceedLogs(), recorder.OnlyWithResourceBody()))
	return nil
}
