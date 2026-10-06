// Copyright 2026 Google LLC
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

package caik8s_impl

import (
	"fmt"
	"slices"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/caik8s"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

func formatClusterResourceLogSummary(identity *k8saudit.ResourceIdentity) string {
	return fmt.Sprintf("CAI resource snapshot: %s/%s", identity.Kind, identity.Name)
}

// clusterResourceSuite bundles the 5 CAI tasks for Kubernetes cluster resource snapshots.
var clusterResourceSuite = gcpcommon.DefineCAITaskSuite(gcpcommon.CAITaskSuiteConfig[*k8saudit.ResourceIdentity]{
	TaskIDs:                  caik8s.ClusterResourceTaskIDs,
	BindSearchTargetResolver: bindClusterResourceSearchTargetResolver,
	PreprocessRawMap:         preprocessK8sTemporalAssetMap,
	ExtractIdentity:          extractK8sIdentity,
	IdentityGroupKey: func(identity *k8saudit.ResourceIdentity) string {
		return identity.String()
	},
	FormatLogSummary:          formatClusterResourceLogSummary,
	BindInitialRevisionMapper: bindClusterResourceInitialRevisionMapper,
})

// gkeResourceSuite bundles the 5 CAI tasks for GKE Cluster and NodePool snapshots.
var gkeResourceSuite = gcpcommon.DefineCAITaskSuite(gcpcommon.CAITaskSuiteConfig[gkeResourceIdentity]{
	TaskIDs:                   caik8s.GKEResourceTaskIDs,
	BindSearchTargetResolver:  bindGKEResourceSearchTargetResolver,
	ExtractIdentity:           extractGKEIdentity,
	IdentityGroupKey:          gkeIdentityGroupKey,
	FormatLogSummary:          formatGKEResourceLogSummary,
	BindInitialRevisionMapper: bindGKEResourceInitialRevisionMapper,
})

// Module declares the Cloud Asset Inventory tasks for GKE clusters and Kubernetes resources.
var Module = coreinspection.Module{
	Name: "googlecloud/caik8s",
	Scope: coreinspection.Scope{
		inspectioncore.InspectionTypeLabelKeyEnvironment:  "googlecloud",
		inspectioncore.InspectionTypeLabelKeyBasePlatform: "kubernetes",
		gcpcommon.InspectionTypeLabelKeyClusterType:       "gke",
	},
	Tasks: slices.Concat(
		clusterResourceSuite.Tasks(),
		gkeResourceSuite.Tasks(),
		[]coretask.UntypedTask{
			clusterResourceInitialStateProviderTask,
			gkeResourceInitialStateProviderTask,
		},
	),
}
