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

package googlecloudclustercomposer_impl

import (
	"context"

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// ComposerClusterNamePrefixTask is the task that returns the composer cluster name prefix.
var ComposerClusterNamePrefixTask = coretask.NewTask(googlecloudclustercomposer_contract.ComposerClusterNamePrefixTaskID, []taskid.UntypedTaskReference{}, func(ctx context.Context) (string, error) {
	return "", nil
}, inspectioncore_contract.InspectionTypeLabel(googlecloudclustercomposer_contract.InspectionTypeId))
