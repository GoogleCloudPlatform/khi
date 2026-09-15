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

package googlecloudclustergkeonaws_impl

import (
	"context"

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	googlecloudclustergkeonaws_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustergkeonaws/contract"
)

// AnthosOnAWSClusterNamePrefixTask is a task that provides the cluster name prefix policy for GKE on AWS.
// This task applies "awsClusters/" prefix across all usage layers.
var AnthosOnAWSClusterNamePrefixTask = coretask.NewTask(googlecloudclustergkeonaws_contract.ClusterNamePrefixTaskID, []coretask.Dependency{}, func(_ context.Context) (k8scommon.ClusterPrefixPolicy, error) {
	return k8scommon.ClusterPrefixPolicy{
		Prefix: "awsClusters/",
		RequiredUsages: []k8scommon.ClusterNameUsage{
			k8scommon.ClusterNameUsageK8sCluster,
			k8scommon.ClusterNameUsageK8sPlatformAudit,
			k8scommon.ClusterNameUsageCSM,
		},
	}, nil
})
