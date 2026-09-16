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
	"context"
	"fmt"
	"slices"
	"strings"

	assetpb "cloud.google.com/go/asset/apiv1/assetpb"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/progress"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

var defaultSupportedKindsToAssetTypes = map[string]string{
	"pod":                            "k8s.io/Pod",
	"node":                           "k8s.io/Node",
	"namespace":                      "k8s.io/Namespace",
	"service":                        "k8s.io/Service",
	"serviceaccount":                 "k8s.io/ServiceAccount",
	"endpoints":                      "k8s.io/Endpoints",
	"persistentvolume":               "k8s.io/PersistentVolume",
	"persistentvolumeclaim":          "k8s.io/PersistentVolumeClaim",
	"podtemplate":                    "k8s.io/PodTemplate",
	"resourcequota":                  "k8s.io/ResourceQuota",
	"deployment":                     "apps.k8s.io/Deployment",
	"statefulset":                    "apps.k8s.io/StatefulSet",
	"daemonset":                      "apps.k8s.io/DaemonSet",
	"replicaset":                     "apps.k8s.io/ReplicaSet",
	"job":                            "batch.k8s.io/Job",
	"cronjob":                        "batch.k8s.io/CronJob",
	"ingress":                        "networking.k8s.io/Ingress",
	"networkpolicy":                  "networking.k8s.io/NetworkPolicy",
	"poddisruptionbudget":            "policy.k8s.io/PodDisruptionBudget",
	"horizontalpodautoscaler":        "autoscaling.k8s.io/HorizontalPodAutoscaler",
	"role":                           "rbac.authorization.k8s.io/Role",
	"rolebinding":                    "rbac.authorization.k8s.io/RoleBinding",
	"clusterrole":                    "rbac.authorization.k8s.io/ClusterRole",
	"clusterrolebinding":             "rbac.authorization.k8s.io/ClusterRoleBinding",
	"storageclass":                   "storage.k8s.io/StorageClass",
	"mutatingwebhookconfiguration":   "admissionregistration.k8s.io/MutatingWebhookConfiguration",
	"validatingwebhookconfiguration": "admissionregistration.k8s.io/ValidatingWebhookConfiguration",
}

func resolveClusterResourceSearchTarget(ctx context.Context, _ inspectioncore.InspectionTaskModeType) (string, gcpcommon.CAIAssetSearchTarget, bool, error) {
	cluster := coretask.GetTaskResult(ctx, k8scommon.ClusterIdentityTaskID.Ref())
	if !cluster.IsComplete() {
		return "", gcpcommon.CAIAssetSearchTarget{}, true, nil
	}

	kindFilter := coretask.GetTaskResult(ctx, k8scommon.InputKindFilterTaskID.Ref())
	assetTypes := resolveAssetTypes(kindFilter)
	if len(assetTypes) == 0 {
		return "", gcpcommon.CAIAssetSearchTarget{}, true, nil
	}

	namespaceFilter := coretask.GetTaskResult(ctx, k8scommon.InputNamespaceFilterTaskID.Ref())
	target := clusterResourceDiscoveryTarget{
		scope:                      fmt.Sprintf("projects/%s", cluster.ProjectID),
		clusterAssetNameCandidates: clusterAssetNameCandidates(cluster),
		assetTypes:                 assetTypes,
		namespaceFilter:            namespaceFilter,
	}

	return cluster.ProjectID, gcpcommon.CAIAssetSearchTarget{
		Scope: target.scope,
		Discover: func(ctx context.Context, fetcher gcpcommon.CAIFetcher) ([]string, error) {
			return discoverClusterResourceAssetNames(ctx, fetcher, target)
		},
	}, false, nil
}

// clusterResourceDiscoveryTarget carries the resolved inputs for discovering cluster resource assets in CAI.
type clusterResourceDiscoveryTarget struct {
	scope                      string
	clusterAssetNameCandidates []string
	assetTypes                 []string
	namespaceFilter            *gcpqueryutil.SetFilterParseResult
}

// namespaceAssetType is the CAI asset type of Kubernetes Namespace resources.
const namespaceAssetType = "k8s.io/Namespace"

// clusterAssetNameCandidates returns the CAI full resource names the cluster may be named by.
func clusterAssetNameCandidates(cluster k8scommon.GoogleCloudClusterIdentity) []string {
	return []string{
		fmt.Sprintf("//container.googleapis.com/projects/%s/locations/%s/clusters/%s", cluster.ProjectID, cluster.Location, cluster.ClusterName),
		fmt.Sprintf("//container.googleapis.com/projects/%s/zones/%s/clusters/%s", cluster.ProjectID, cluster.Location, cluster.ClusterName),
	}
}

// assetTypesWithNamespace returns the asset types to search in the cluster parented phase.
func assetTypesWithNamespace(assetTypes []string) []string {
	if slices.Contains(assetTypes, namespaceAssetType) {
		return slices.Clone(assetTypes)
	}
	return append(slices.Clone(assetTypes), namespaceAssetType)
}

// targetNamespaceParents returns the CAI full resource names of the namespaces to search.
func targetNamespaceParents(clusterParentedResults []*assetpb.ResourceSearchResult, filter *gcpqueryutil.SetFilterParseResult) []string {
	var parents []string
	for _, searchResult := range clusterParentedResults {
		if searchResult.AssetType != namespaceAssetType {
			continue
		}
		_, namespaceName := parseK8sAssetName(searchResult.Name)
		if matchesNamespaceFilter(filter, namespaceName) {
			parents = append(parents, searchResult.Name)
		}
	}
	return parents
}

// filterSearchResultsByAssetType drops results of asset types the kind filter did not ask for.
func filterSearchResultsByAssetType(searchResults []*assetpb.ResourceSearchResult, assetTypes []string) []*assetpb.ResourceSearchResult {
	var matched []*assetpb.ResourceSearchResult
	for _, searchResult := range searchResults {
		if slices.Contains(assetTypes, searchResult.AssetType) {
			matched = append(matched, searchResult)
		}
	}
	return matched
}

// discoverClusterResourceAssetNames runs the two-phase CAI search (cluster-parented then namespace-parented)
// and returns the matching full resource names.
func discoverClusterResourceAssetNames(ctx context.Context, fetcher gcpcommon.CAIFetcher, target clusterResourceDiscoveryTarget) ([]string, error) {
	progress.ReportIndeterminate(ctx, "Searching cluster-scoped resources in Cloud Asset Inventory...")

	clusterParentedResults, err := gcpcommon.SearchCAIAssetsByParents(ctx, fetcher, target.scope, assetTypesWithNamespace(target.assetTypes), target.clusterAssetNameCandidates)
	if err != nil {
		return nil, fmt.Errorf("failed to search cluster parented resources from CAI: %w", err)
	}

	namespaceParents := targetNamespaceParents(clusterParentedResults, target.namespaceFilter)
	progress.ReportIndeterminate(ctx, fmt.Sprintf("Searching namespaced resources in Cloud Asset Inventory (%d namespaces)...", len(namespaceParents)))

	namespaceParentedResults, err := gcpcommon.SearchCAIAssetsByParents(ctx, fetcher, target.scope, target.assetTypes, namespaceParents)
	if err != nil {
		return nil, fmt.Errorf("failed to search namespace parented resources from CAI: %w", err)
	}

	searchResults := append(filterSearchResultsByAssetType(clusterParentedResults, target.assetTypes), namespaceParentedResults...)
	return filterAssetNamesByNamespace(searchResults, target.namespaceFilter), nil
}

// filterAssetNamesByNamespace filters search results by the namespace filter and returns matching asset names.
func filterAssetNamesByNamespace(searchResults []*assetpb.ResourceSearchResult, filter *gcpqueryutil.SetFilterParseResult) []string {
	var matched []string
	for _, searchResult := range searchResults {
		namespace, _ := parseK8sAssetName(searchResult.Name)
		if matchesNamespaceFilter(filter, namespace) {
			matched = append(matched, searchResult.Name)
		}
	}
	return matched
}

// resolveAssetTypes computes the list of CAI asset types to search based on the kind filter.
func resolveAssetTypes(filter *gcpqueryutil.SetFilterParseResult) []string {
	if filter.ValidationError != "" {
		return nil
	}
	if filter.SubtractMode {
		return resolveSubtractiveAssetTypes(filter.Subtractives)
	}
	return resolveAdditiveAssetTypes(filter.Additives)
}

// resolveSubtractiveAssetTypes returns the default asset types minus the ones mapped from the excluded kinds.
func resolveSubtractiveAssetTypes(subtractives []string) []string {
	allTypes := defaultAssetTypes()
	subtractSet := make(map[string]struct{})
	for _, subtractive := range subtractives {
		if mapped, found := defaultSupportedKindsToAssetTypes[strings.ToLower(subtractive)]; found {
			subtractSet[mapped] = struct{}{}
		}
	}
	var result []string
	for _, assetType := range allTypes {
		if _, excluded := subtractSet[assetType]; !excluded {
			result = append(result, assetType)
		}
	}
	return result
}

// resolveAdditiveAssetTypes returns the sorted, deduplicated asset types mapped from the requested kinds.
func resolveAdditiveAssetTypes(additives []string) []string {
	if len(additives) == 0 {
		return nil
	}
	addSet := make(map[string]struct{})
	for _, additive := range additives {
		lowerAdditive := strings.ToLower(additive)
		if mapped, found := defaultSupportedKindsToAssetTypes[lowerAdditive]; found {
			addSet[mapped] = struct{}{}
		}
	}
	var result []string
	for t := range addSet {
		result = append(result, t)
	}
	slices.Sort(result)
	return result
}

// defaultAssetTypes returns a sorted slice of all default supported CAI asset types.
func defaultAssetTypes() []string {
	seen := make(map[string]struct{})
	var result []string
	for _, assetType := range defaultSupportedKindsToAssetTypes {
		if _, ok := seen[assetType]; !ok {
			seen[assetType] = struct{}{}
			result = append(result, assetType)
		}
	}
	slices.Sort(result)
	return result
}

// matchesNamespaceFilter returns true if the namespace matches the parsed namespace filter.
func matchesNamespaceFilter(filter *gcpqueryutil.SetFilterParseResult, namespace string) bool {
	if filter.SubtractMode {
		if namespace == "" {
			return !slices.Contains(filter.Subtractives, "#cluster-scoped")
		}
		if slices.Contains(filter.Subtractives, "#namespaced") {
			return false
		}
		return !slices.Contains(filter.Subtractives, strings.ToLower(namespace))
	}
	if namespace == "" {
		return slices.Contains(filter.Additives, "#cluster-scoped")
	}
	if slices.Contains(filter.Additives, "#namespaced") {
		return true
	}
	return slices.Contains(filter.Additives, strings.ToLower(namespace))
}
