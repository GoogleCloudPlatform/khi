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

package googlecloudclustergdcbaremetal_impl

import (
	"context"

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudclustergdcbaremetal_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustergdcbaremetal/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// GDCVForBaremetalClusterNamePrefixTask is a task that returns an empty string as the cluster name prefix for GDCV for Baremetal.
// This task is necessary to satisfy the dependency of the log source profile, but GDCV for Baremetal does not require a prefix.
var GDCVForBaremetalClusterNamePrefixTask = coretask.NewTask(googlecloudclustergdcbaremetal_contract.ClusterNamePrefixTaskIDForGDCVForBaremetal, []taskid.UntypedTaskReference{}, func(ctx context.Context) (string, error) {
	return "", nil
}, inspectioncore_contract.InspectionTypeLabel(googlecloudclustergdcbaremetal_contract.InspectionTypeId))
