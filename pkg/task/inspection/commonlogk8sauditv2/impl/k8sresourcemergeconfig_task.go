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

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/k8s"
	commonlogk8sauditv2_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8sauditv2/contract"
)

// DefaultK8sResourceMergeConfigTask is the task that generates the default patch request merge config.
var DefaultK8sResourceMergeConfigTask = coretask.NewTask(commonlogk8sauditv2_contract.K8sResourceMergeConfigTaskID, []taskid.UntypedTaskReference{}, func(ctx context.Context) (*k8s.K8sManifestMergeConfigRegistry, error) {
	return k8s.GenerateDefaultMergeConfig()
})
