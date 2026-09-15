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

package gdcvmware_impl

import (
	"context"

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/cluster/gdcvmware"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
)

// GDCVForVMWareClusterNamePrefixTask is a task that returns a prefix policy as the cluster name prefix for GDCV for VMWare.
// This task applies "vmwareClusters/" prefix only for platform audit and CSM logs.
var GDCVForVMWareClusterNamePrefixTask = coretask.NewTask(gdcvmware.ClusterNamePrefixTaskIDForGDCVForVMWare, []coretask.Dependency{}, func(_ context.Context) (k8scommon.ClusterPrefixPolicy, error) {
	return k8scommon.ClusterPrefixPolicy{
		Prefix: "vmwareClusters/",
		RequiredUsages: []k8scommon.ClusterNameUsage{
			k8scommon.ClusterNameUsageK8sPlatformAudit,
			k8scommon.ClusterNameUsageCSM,
		},
	}, nil
})
