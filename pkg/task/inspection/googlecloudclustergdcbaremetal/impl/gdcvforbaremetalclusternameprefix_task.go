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
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
)

// GDCVForBaremetalClusterNamePrefixTask is a task that returns a prefix policy as the cluster name prefix for GDCV for Baremetal.
// This task applies "baremetalClusters/" prefix only for platform audit and CSM logs.
var GDCVForBaremetalClusterNamePrefixTask = coretask.NewTask(googlecloudclustergdcbaremetal_contract.ClusterNamePrefixTaskIDForGDCVForBaremetal, []taskid.UntypedTaskReference{}, func(_ context.Context) (googlecloudk8scommon_contract.ClusterPrefixPolicy, error) {
	return googlecloudk8scommon_contract.ClusterPrefixPolicy{
		Prefix: "baremetalClusters/",
		RequiredUsages: []googlecloudk8scommon_contract.ClusterNameUsage{
			googlecloudk8scommon_contract.ClusterNameUsageK8sPlatformAudit,
			googlecloudk8scommon_contract.ClusterNameUsageCSM,
		},
	}, nil
})
