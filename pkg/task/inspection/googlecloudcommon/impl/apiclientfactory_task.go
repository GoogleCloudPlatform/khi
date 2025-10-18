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

package googlecloudcommon_impl

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
)

// APIClientFactoryTask is a task to inject googlecloud.ClientFactory to the later tasks. The instance is singleton in an inspection and the instance is cached on inspection cache after the first generation.
var APIClientFactoryTask = inspectiontaskbase.NewCachedTask(googlecloudcommon_contract.APIClientFactoryTaskID, []taskid.UntypedTaskReference{
	googlecloudcommon_contract.APIClientFactoryOptionsTaskID.Ref(),
}, func(ctx context.Context, prevValue inspectiontaskbase.CacheableTaskResult[*googlecloud.ClientFactory]) (inspectiontaskbase.CacheableTaskResult[*googlecloud.ClientFactory], error) {
	// Use cached client if it was set already.
	if prevValue.DependencyDigest != "" {
		return prevValue, nil
	}
	opts := coretask.GetTaskResult(ctx, googlecloudcommon_contract.APIClientFactoryOptionsTaskID.Ref())

	clientFactory, err := googlecloud.NewClientFactory(opts...)
	if err != nil {
		return inspectiontaskbase.CacheableTaskResult[*googlecloud.ClientFactory]{}, err
	}
	return inspectiontaskbase.CacheableTaskResult[*googlecloud.ClientFactory]{
		DependencyDigest: "singleton", // the client don't need to recreate and it's singleton because the options are not expected to be refresh.
		Value:            clientFactory,
	}, nil
})
