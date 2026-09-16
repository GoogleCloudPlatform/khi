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

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/caik8s"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

func formatClusterResourceLogSummary(identity *k8saudit.ResourceIdentity) string {
	return fmt.Sprintf("CAI resource snapshot: %s/%s", identity.Kind, identity.Name)
}

// ClusterResourceSuite bundles the 5 CAI tasks for Kubernetes cluster resource snapshots.
var ClusterResourceSuite = gcpcommon.NewCAITaskSuite(gcpcommon.CAITaskSuiteConfig[*k8saudit.ResourceIdentity]{
	TaskIDs: caik8s.ClusterResourceTaskIDs,
	FetcherDependencies: []coretask.Dependency{
		k8scommon.ClusterIdentityTaskID.Ref(),
		k8scommon.InputKindFilterTaskID.Ref(),
		k8scommon.InputNamespaceFilterTaskID.Ref(),
	},
	ResolveSearchTarget: resolveClusterResourceSearchTarget,
	PreprocessRawMap:    preprocessK8sTemporalAssetMap,
	ExtractIdentity:     extractK8sIdentity,
	IdentityGroupKey: func(identity *k8saudit.ResourceIdentity) string {
		return identity.String()
	},
	FormatLogSummary: formatClusterResourceLogSummary,
	MapperDependencies: []coretask.Dependency{
		k8scommon.ClusterIdentityTaskID.Ref(),
	},
	MapInitialRevision: mapClusterResourceInitialRevision,
})

// GKEResourceSuite bundles the 5 CAI tasks for GKE Cluster and NodePool snapshots.
var GKEResourceSuite = gcpcommon.NewCAITaskSuite(gcpcommon.CAITaskSuiteConfig[gkeResourceIdentity]{
	TaskIDs: caik8s.GKEResourceTaskIDs,
	FetcherDependencies: []coretask.Dependency{
		k8scommon.ClusterIdentityTaskID.Ref(),
	},
	ResolveSearchTarget: resolveGKEResourceSearchTarget,
	ExtractIdentity:     extractGKEIdentity,
	IdentityGroupKey:    gkeIdentityGroupKey,
	FormatLogSummary:    formatGKEResourceLogSummary,
	MapperDependencies: []coretask.Dependency{
		k8scommon.ClusterIdentityTaskID.Ref(),
		caik8s.GKEResourceTaskIDs.RawLog.Ref(),
	},
	MapInitialRevision: mapGKEResourceInitialRevision,
})

// Register registers all googlecloudcaik8s inspection tasks to the registry.
func Register(registry coreinspection.InspectionTaskRegistry) error {
	scoped := coreinspection.NewScopedRegistry(
		registry,
		inspectioncore.InspectionTypeLabelSelector(map[string]string{
			inspectioncore.InspectionTypeLabelKeyEnvironment:  "googlecloud",
			inspectioncore.InspectionTypeLabelKeyBasePlatform: "kubernetes",
			gcpcommon.InspectionTypeLabelKeyClusterType:       "gke",
		}),
	)

	if err := ClusterResourceSuite.Register(scoped); err != nil {
		return err
	}
	if err := GKEResourceSuite.Register(scoped); err != nil {
		return err
	}
	return coretask.RegisterTasks(
		scoped,
		ClusterResourceInitialStateProviderTask,
		GKEResourceInitialStateProviderTask,
	)
}
