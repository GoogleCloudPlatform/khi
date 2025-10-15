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

package googlecloudclustergkeonazure_impl

import (
	"context"

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudclustergkeonazure_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustergkeonazure/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
)

// ClusterListFetcherTask is a task to inject ClusterListFetcher reference.
var ClusterListFetcherTask = coretask.NewTask(googlecloudclustergkeonazure_contract.ClusterListFetcherTaskID, []taskid.UntypedTaskReference{
	googlecloudcommon_contract.APIClientFactoryTaskID.Ref(),
},
	func(ctx context.Context) (googlecloudclustergkeonazure_contract.ClusterListFetcher, error) {
		return &googlecloudclustergkeonazure_contract.ClusterListFetcherImpl{}, nil
	})
