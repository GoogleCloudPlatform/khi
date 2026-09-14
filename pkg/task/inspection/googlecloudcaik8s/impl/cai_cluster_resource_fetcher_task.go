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

package googlecloudcaik8s_impl

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	assetpb "cloud.google.com/go/asset/apiv1/assetpb"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudcaik8s_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcaik8s/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"google.golang.org/protobuf/types/known/timestamppb"
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

// ClusterResourceFetcherTask queries CAI for existing Kubernetes resources in a GKE cluster.
var ClusterResourceFetcherTask = inspectiontaskbase.NewProgressReportableInspectionTask(
	googlecloudcaik8s_contract.ClusterResourceFetcherTaskID,
	[]taskid.UntypedTaskReference{
		googlecloudk8scommon_contract.ClusterIdentityTaskID.Ref(),
		googlecloudcommon_contract.APIClientFactoryTaskID.Ref(),
		googlecloudcommon_contract.APIClientCallOptionsInjectorTaskID.Ref(),
		googlecloudcommon_contract.InputStartTimeTaskID.Ref(),
		googlecloudcommon_contract.InputEndTimeTaskID.Ref(),
		googlecloudk8scommon_contract.InputKindFilterTaskID.Ref(),
		googlecloudk8scommon_contract.InputNamespaceFilterTaskID.Ref(),
	},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType, progress *inspectionmetadata.TaskProgressMetadata) ([]*googlecloudcaik8s_contract.ClusterResourceSnapshot, error) {
		cluster := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.ClusterIdentityTaskID.Ref())
		factory := coretask.GetTaskResult(ctx, googlecloudcommon_contract.APIClientFactoryTaskID.Ref())
		injector, _ := coretask.GetTaskResultOptional(ctx, googlecloudcommon_contract.APIClientCallOptionsInjectorTaskID.Ref())
		startTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputStartTimeTaskID.Ref())
		endTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputEndTimeTaskID.Ref())
		kindFilter := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.InputKindFilterTaskID.Ref())
		namespaceFilter := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.InputNamespaceFilterTaskID.Ref())

		if taskMode == inspectioncore_contract.TaskModeDryRun || !cluster.IsComplete() {
			return []*googlecloudcaik8s_contract.ClusterResourceSnapshot{}, nil
		}

		assetTypes := resolveAssetTypes(kindFilter)
		if len(assetTypes) == 0 {
			return []*googlecloudcaik8s_contract.ClusterResourceSnapshot{}, nil
		}

		fetcher := NewCAIFetcher(factory, injector, cluster.ProjectID)

		snapshots, err := fetchClusterResourceSnapshots(ctx, fetcher, clusterResourceLookup{
			scope:                   fmt.Sprintf("projects/%s", cluster.ProjectID),
			clusterParentCandidates: clusterParentCandidates(cluster),
			assetTypes:              assetTypes,
			namespaceFilter:         namespaceFilter,
			timeWindow: &assetpb.TimeWindow{
				StartTime: timestamppb.New(startTime),
				EndTime:   timestamppb.New(endTime),
			},
		}, progress)
		if err != nil {
			// A CAI failure must not break the rest of the inspection. The pipeline then behaves as if
			// the inventory covered no resource at all.
			slog.WarnContext(ctx, "failed to fetch cluster resource snapshots from CAI", "error", err)
			return []*googlecloudcaik8s_contract.ClusterResourceSnapshot{}, nil
		}
		return snapshots, nil
	},
)

// clusterResourceLookup carries the resolved inputs of a single CAI cluster resource lookup.
type clusterResourceLookup struct {
	// scope is the CAI search scope and the parent of the history lookup, in the form "projects/{id}".
	scope string
	// clusterParentCandidates holds the CAI full resource names the cluster may be named by.
	clusterParentCandidates []string
	// assetTypes limits the search to the CAI asset types derived from the kind filter.
	assetTypes []string
	// namespaceFilter selects which namespaces to search and which discovered assets to keep.
	namespaceFilter *gcpqueryutil.SetFilterParseResult
	// timeWindow is the inspection time range to read asset history for.
	timeWindow *assetpb.TimeWindow
}

// namespaceAssetType is the CAI asset type of Kubernetes Namespace resources.
const namespaceAssetType = "k8s.io/Namespace"

// CAI rejects search queries that exceed either of these limits.
// See https://cloud.google.com/asset-inventory/docs/query-syntax for the documented values.
const (
	maxSearchQueryComparisons = 10
	maxSearchQueryCharacters  = 2048
)

// clusterParentCandidates returns the CAI full resource names the cluster may be named by.
// CAI names regional clusters under "/locations/" and zonal clusters under "/zones/", and the
// inspection input carries a single location string that does not tell the two apart, so both
// forms are searched. Only the form that exists returns results.
func clusterParentCandidates(cluster googlecloudk8scommon_contract.GoogleCloudClusterIdentity) []string {
	return []string{
		fmt.Sprintf("//container.googleapis.com/projects/%s/locations/%s/clusters/%s", cluster.ProjectID, cluster.Location, cluster.ClusterName),
		fmt.Sprintf("//container.googleapis.com/projects/%s/zones/%s/clusters/%s", cluster.ProjectID, cluster.Location, cluster.ClusterName),
	}
}

// buildParentSearchQueries builds the CAI search expressions matching assets whose parent is exactly
// one of the given full resource names, splitting them to stay within the CAI query limits.
//
// Only the "=" operator is used. The ":" operator tokenizes both operands on every non-alphanumeric
// character and, when the phrase ends with a wildcard, matches those token prefixes in any order.
// A prefix such as ".../clusters/my-cluster/*" therefore also matches assets of "my-cluster-2", so
// it cannot scope a search to one cluster. Exact matching has no such ambiguity.
func buildParentSearchQueries(parents []string) []string {
	var queries []string
	var comparisons []string
	characterCount := 0
	for _, parent := range parents {
		comparison := fmt.Sprintf("parentFullResourceName=%q", parent)
		addedCharacters := len(comparison)
		if len(comparisons) > 0 {
			addedCharacters += len(" OR ")
		}
		// A comparison that alone exceeds the character limit still forms its own query, because
		// splitting it further is impossible.
		if len(comparisons) > 0 && (len(comparisons) == maxSearchQueryComparisons || characterCount+addedCharacters > maxSearchQueryCharacters) {
			queries = append(queries, strings.Join(comparisons, " OR "))
			comparisons = nil
			characterCount = 0
			addedCharacters = len(comparison)
		}
		comparisons = append(comparisons, comparison)
		characterCount += addedCharacters
	}
	if len(comparisons) > 0 {
		queries = append(queries, strings.Join(comparisons, " OR "))
	}
	return queries
}

// assetTypesWithNamespace returns the asset types to search in the cluster parented phase.
// Namespace assets are always searched because their names are the parent values the namespace
// parented phase needs, even when the kind filter excludes them from the result.
func assetTypesWithNamespace(assetTypes []string) []string {
	if slices.Contains(assetTypes, namespaceAssetType) {
		return slices.Clone(assetTypes)
	}
	return append(slices.Clone(assetTypes), namespaceAssetType)
}

// targetNamespaceParents returns the CAI full resource names of the namespaces to search.
// A Namespace asset's own name is exactly the parent value its members carry, so the cluster path
// is reused verbatim and the locations or zones form resolves itself.
func targetNamespaceParents(clusterParentedResults []*assetpb.ResourceSearchResult, filter *gcpqueryutil.SetFilterParseResult) []string {
	var parents []string
	for _, searchResult := range clusterParentedResults {
		if searchResult.AssetType != namespaceAssetType {
			continue
		}
		// A Namespace asset carries no namespace of its own, so parseAssetName reports its
		// resource name, which is the namespace name the filter matches against.
		_, namespaceName := parseAssetName(searchResult.Name)
		if matchesNamespaceFilter(filter, namespaceName) {
			parents = append(parents, searchResult.Name)
		}
	}
	return parents
}

// filterSearchResultsByAssetType drops results of asset types the kind filter did not ask for.
// Namespace assets are searched unconditionally to list the namespaces, so they reach this function
// even when unrequested.
func filterSearchResultsByAssetType(searchResults []*assetpb.ResourceSearchResult, assetTypes []string) []*assetpb.ResourceSearchResult {
	var matched []*assetpb.ResourceSearchResult
	for _, searchResult := range searchResults {
		if slices.Contains(assetTypes, searchResult.AssetType) {
			matched = append(matched, searchResult)
		}
	}
	return matched
}

// searchAssetsByParent runs one CAI search per query and returns the concatenated results.
func searchAssetsByParent(ctx context.Context, fetcher googlecloudcaik8s_contract.CAIFetcher, scope string, assetTypes []string, queries []string) ([]*assetpb.ResourceSearchResult, error) {
	var results []*assetpb.ResourceSearchResult
	for _, query := range queries {
		searchResults, err := fetcher.SearchResources(ctx, scope, query, assetTypes)
		if err != nil {
			return nil, fmt.Errorf("failed to search assets with query %q: %w", query, err)
		}
		results = append(results, searchResults...)
	}
	return results, nil
}

// fetchClusterResourceSnapshots searches CAI for the resources of the cluster and converts their
// temporal history into cluster resource snapshots.
//
// The search runs in two phases because CAI models the parent of a namespaced resource as its
// Namespace and the parent of a cluster-scoped resource as the cluster. There is no single parent
// value covering a whole cluster, so the cluster parented phase runs first and the Namespace assets
// it returns supply the parent values of the namespace parented phase.
func fetchClusterResourceSnapshots(ctx context.Context, fetcher googlecloudcaik8s_contract.CAIFetcher, lookup clusterResourceLookup, progress *inspectionmetadata.TaskProgressMetadata) ([]*googlecloudcaik8s_contract.ClusterResourceSnapshot, error) {
	progress.Indeterminate = true
	progress.Message = "Searching cluster-scoped resources in Cloud Asset Inventory..."

	clusterParentedResults, err := searchAssetsByParent(ctx, fetcher, lookup.scope, assetTypesWithNamespace(lookup.assetTypes), buildParentSearchQueries(lookup.clusterParentCandidates))
	if err != nil {
		return nil, fmt.Errorf("failed to search cluster parented resources from CAI: %w", err)
	}

	namespaceParents := targetNamespaceParents(clusterParentedResults, lookup.namespaceFilter)
	progress.Message = fmt.Sprintf("Searching namespaced resources in Cloud Asset Inventory (%d namespaces)...", len(namespaceParents))

	namespaceParentedResults, err := searchAssetsByParent(ctx, fetcher, lookup.scope, lookup.assetTypes, buildParentSearchQueries(namespaceParents))
	if err != nil {
		return nil, fmt.Errorf("failed to search namespace parented resources from CAI: %w", err)
	}

	searchResults := append(filterSearchResultsByAssetType(clusterParentedResults, lookup.assetTypes), namespaceParentedResults...)
	matchedAssetNames := filterAssetNamesByNamespace(searchResults, lookup.namespaceFilter)
	if len(matchedAssetNames) == 0 {
		return []*googlecloudcaik8s_contract.ClusterResourceSnapshot{}, nil
	}

	totalChunks := (len(matchedAssetNames) + maxBatchHistorySize - 1) / maxBatchHistorySize
	progress.Indeterminate = false
	progress.Percentage = 0.0
	progress.Message = fmt.Sprintf("Fetching asset history (0/%d chunks, %d assets)...", totalChunks, len(matchedAssetNames))

	temporalAssets, err := fetcher.BatchGetAssetsHistory(ctx, lookup.scope, matchedAssetNames, assetpb.ContentType_RESOURCE, lookup.timeWindow, func(completedChunks, totalChunks int) {
		progress.Percentage = float32(completedChunks) / float32(totalChunks)
		progress.Message = fmt.Sprintf("Fetching asset history (%d/%d chunks, %d assets)...", completedChunks, totalChunks, len(matchedAssetNames))
	})
	if err != nil {
		return nil, fmt.Errorf("failed to batch get assets history from CAI: %w", err)
	}

	return convertTemporalAssets(temporalAssets)
}

// filterAssetNamesByNamespace filters search results by the namespace filter and returns matching asset names.
func filterAssetNamesByNamespace(searchResults []*assetpb.ResourceSearchResult, filter *gcpqueryutil.SetFilterParseResult) []string {
	var matched []string
	for _, searchResult := range searchResults {
		namespace, _ := parseAssetName(searchResult.Name)
		if matchesNamespaceFilter(filter, namespace) {
			matched = append(matched, searchResult.Name)
		}
	}
	return matched
}

// convertTemporalAssets converts a slice of temporal assets into cluster resource snapshots.
func convertTemporalAssets(temporalAssets []*assetpb.TemporalAsset) ([]*googlecloudcaik8s_contract.ClusterResourceSnapshot, error) {
	snapshots := make([]*googlecloudcaik8s_contract.ClusterResourceSnapshot, 0, len(temporalAssets))
	for _, temporalAsset := range temporalAssets {
		snapshot, err := ConvertTemporalAssetToClusterResourceSnapshot(temporalAsset)
		if err != nil {
			return nil, fmt.Errorf("failed to convert temporal asset to cluster resource snapshot: %w", err)
		}
		snapshots = append(snapshots, snapshot)
	}
	return snapshots, nil
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
// Kinds without a mapping are ignored because they can never match a default asset type.
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
// Kinds without a mapping are ignored, and an empty result makes the caller skip the CAI lookup entirely.
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
