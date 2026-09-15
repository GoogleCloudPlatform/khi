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

package k8scommon_impl

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// AutocompleteMetricsK8sContainerTask is the task to provide the default metrics type to collect the cluster names.
// The resource type "k8s_container" must be available on the returned metrics type.
// This task is overridden in GKE clusters.
var AutocompleteMetricsK8sContainerTask = coretask.NewTask(k8scommon.AutocompleteMetricsK8sContainerTaskID, []coretask.Dependency{}, func(ctx context.Context) (string, error) {
	// logging.googleapis.com/log_entry_count is better from the perspective of KHI's purpose, but use container metrics for longer retention period(24 months).
	return "kubernetes.io/anthos/up", nil
})

var AutocompleteMetricsK8sNodeTask = coretask.NewTask(k8scommon.AutocompleteMetricsK8sNodeTaskID, []coretask.Dependency{}, func(ctx context.Context) (string, error) {
	return "kubernetes.io/anthos/up", nil
})

var AutocompleteClusterIdentityTask = inspectiontaskbase.NewGlobalCachedTask(k8scommon.AutocompleteClusterIdentityTaskID, []coretask.Dependency{
	k8scommon.ClusterNamePrefixTaskRef,
	gcpcommon.InputProjectIdTaskID.Ref(),
	gcpcommon.InputStartTimeTaskID.Ref(),
	gcpcommon.InputEndTimeTaskID.Ref(),
	k8scommon.AutocompleteMetricsK8sContainerTaskID.Ref(),
	gcpcommon.APIClientFactoryTaskID.Ref(),
	gcpcommon.APIClientCallOptionsInjectorTaskID.Ref(),
}, func(ctx context.Context, prevValue inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity]]) (inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity]], error) {
	clusterNamePrefix := coretask.GetTaskResult(ctx, k8scommon.ClusterNamePrefixTaskRef)
	projectID := coretask.GetTaskResult(ctx, gcpcommon.InputProjectIdTaskID.Ref())
	startTime := coretask.GetTaskResult(ctx, gcpcommon.InputStartTimeTaskID.Ref())
	endTime := coretask.GetTaskResult(ctx, gcpcommon.InputEndTimeTaskID.Ref())
	metricsType := coretask.GetTaskResult(ctx, k8scommon.AutocompleteMetricsK8sContainerTaskID.Ref())
	cf := coretask.GetTaskResult(ctx, gcpcommon.APIClientFactoryTaskID.Ref())
	optionInjector := coretask.GetTaskResult(ctx, gcpcommon.APIClientCallOptionsInjectorTaskID.Ref())

	currentDigest := fmt.Sprintf("%s-%s-%d-%d", clusterNamePrefix.PrefixFor(k8scommon.ClusterNameUsageK8sCluster), projectID, startTime.Unix(), endTime.Unix())
	if currentDigest == prevValue.DependencyDigest {
		return prevValue, nil
	}
	if projectID == "" {
		return inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity]]{
			Value: &inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity]{
				Values: []k8scommon.GoogleCloudClusterIdentity{},
				Error:  "",
				Hint:   "Cluster names are suggested after the project ID is provided.",
			},
			DependencyDigest: currentDigest,
		}, nil
	}

	errorString := ""
	hintString := ""
	if endTime.Before(time.Now().Add(-time.Hour * 24 * 30 * 24)) {
		hintString = "The end time is more than 24 months ago. Suggested cluster names may not be complete."
	}

	client, err := cf.MonitoringMetricClient(ctx, googlecloud.Project(projectID))
	if err != nil {
		return prevValue, fmt.Errorf("failed to create monitoring metric client: %w", err)
	}

	ctx = optionInjector.InjectToCallContext(ctx, googlecloud.Project(projectID))
	filter := fmt.Sprintf(`metric.type="%s" AND resource.type="k8s_container"`, metricsType)
	metricsLabels, err := googlecloud.QueryResourceLabelsFromMetrics(ctx, client, projectID, filter, startTime, endTime, []string{"resource.label.cluster_name", "resource.label.location"})
	if err != nil {
		errorString = err.Error()
	}
	metricsLabels = filterAndTrimPrefixFromClusterNames(metricsLabels, clusterNamePrefix.PrefixFor(k8scommon.ClusterNameUsageK8sCluster))
	if hintString == "" && errorString == "" && len(metricsLabels) == 0 {
		hintString = fmt.Sprintf("No cluster names found between %s and %s. It is highly likely that the time range is incorrect. Please verify the time range, or proceed by manually entering the cluster name.", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
	}

	identities := make([]k8scommon.GoogleCloudClusterIdentity, len(metricsLabels))
	for i, labels := range metricsLabels {
		identities[i] = k8scommon.GoogleCloudClusterIdentity{
			ProjectID:    projectID,
			PrefixPolicy: clusterNamePrefix,
			ClusterName:  labels["cluster_name"],
			Location:     labels["location"],
		}
	}

	return inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity]]{
		DependencyDigest: currentDigest,
		Value: &inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity]{
			Values: identities,
			Error:  errorString,
			Hint:   hintString,
		},
	}, nil
})

// filterAndTrimPrefixFromClusterNames filters cluster names by prefix and trims the prefix from the filtered cluster names.
func filterAndTrimPrefixFromClusterNames(metricsLabels []map[string]string, prefix string) []map[string]string {
	filteredClusters := make([]map[string]string, 0, len(metricsLabels))
	for _, labels := range metricsLabels {
		clusterName := labels["cluster_name"]
		if prefix == "" {
			if !strings.Contains(clusterName, "/") {
				filteredClusters = append(filteredClusters, labels)
			}
		} else if strings.HasPrefix(clusterName, prefix) {
			labels["cluster_name"] = strings.TrimPrefix(clusterName, prefix)
			filteredClusters = append(filteredClusters, labels)
		}
	}
	return filteredClusters
}

// AutocompleteLocationForClusterTask returns the location for the given cluster name.
var AutocompleteLocationForClusterTask = inspectiontaskbase.NewGlobalCachedTask(k8scommon.AutocompleteLocationForClusterTaskID, []coretask.Dependency{
	k8scommon.InputClusterNameTaskID.Ref(), // This task must not depend on ClusterIdentity because this autocomplete will generate the source of it.
	gcpcommon.InputProjectIdTaskID.Ref(),
	gcpcommon.InputStartTimeTaskID.Ref(),
	gcpcommon.InputEndTimeTaskID.Ref(),
	k8scommon.AutocompleteClusterIdentityTaskID.Ref(),
}, func(ctx context.Context, prevValue inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[string]]) (inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[string]], error) {
	projectID := coretask.GetTaskResult(ctx, gcpcommon.InputProjectIdTaskID.Ref())
	clusterName := coretask.GetTaskResult(ctx, k8scommon.InputClusterNameTaskID.Ref())
	startTime := coretask.GetTaskResult(ctx, gcpcommon.InputStartTimeTaskID.Ref())
	endTime := coretask.GetTaskResult(ctx, gcpcommon.InputEndTimeTaskID.Ref())
	clusterIdentities := coretask.GetTaskResult(ctx, k8scommon.AutocompleteClusterIdentityTaskID.Ref())

	currentDigest := fmt.Sprintf("%s-%s-%d-%d", clusterName, projectID, startTime.Unix(), endTime.Unix())
	if currentDigest == prevValue.DependencyDigest {
		return prevValue, nil
	}
	if projectID == "" {
		return inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[string]]{
			Value: &inspectioncore.AutocompleteResult[string]{
				Values: []string{},
				Error:  "",
				Hint:   "Locations will be suggested after the project ID is provided.",
			},
			DependencyDigest: currentDigest,
		}, nil
	}
	if clusterIdentities.Error != "" {
		return inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[string]]{
			Value: &inspectioncore.AutocompleteResult[string]{
				Values: []string{},
				Error:  clusterIdentities.Error,
				Hint:   clusterIdentities.Hint,
			},
			DependencyDigest: currentDigest,
		}, nil
	}
	if clusterName == "" {
		return inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[string]]{
			Value: &inspectioncore.AutocompleteResult[string]{
				Values: []string{},
				Error:  "",
				Hint:   "Locations will be suggested after the cluster name is provided.",
			},
			DependencyDigest: currentDigest,
		}, nil
	}
	result := &inspectioncore.AutocompleteResult[string]{
		Values: []string{},
		Error:  "",
		Hint:   "",
	}

	// Limit the location to the items which has the same cluster name.
	for _, identity := range clusterIdentities.Values {
		if identity.ClusterName == clusterName {
			result.Values = append(result.Values, identity.Location)
		}
	}
	return inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[string]]{
		Value:            result,
		DependencyDigest: currentDigest,
	}, nil
}, coretask.WithSelectionPriority(500))

// clusterScopedAutocompleteConfig defines parameters for querying cluster-scoped autocomplete suggestions from Cloud Monitoring metrics.
type clusterScopedAutocompleteConfig struct {
	resourceType       string
	resourceLabelKey   string
	targetNameSingular string
	targetNamePlural   string
}

// queryClusterScopedAutocompleteMetrics executes a cached metric label query for cluster-scoped autocomplete tasks.
func queryClusterScopedAutocompleteMetrics(
	ctx context.Context,
	prevValue inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[string]],
	metricsType string,
	cfg clusterScopedAutocompleteConfig,
) (inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[string]], error) {
	cluster := coretask.GetTaskResult(ctx, k8scommon.ClusterIdentityTaskID.Ref())
	startTime := coretask.GetTaskResult(ctx, gcpcommon.InputStartTimeTaskID.Ref())
	endTime := coretask.GetTaskResult(ctx, gcpcommon.InputEndTimeTaskID.Ref())
	cf := coretask.GetTaskResult(ctx, gcpcommon.APIClientFactoryTaskID.Ref())
	optionInjector := coretask.GetTaskResult(ctx, gcpcommon.APIClientCallOptionsInjectorTaskID.Ref())

	currentDigest := fmt.Sprintf("%s-%d-%d", cluster.UniqueDigest(), startTime.Unix(), endTime.Unix())
	if currentDigest == prevValue.DependencyDigest {
		return prevValue, nil
	}
	if !cluster.IsComplete() {
		capitalizedPlural := strings.ToUpper(cfg.targetNamePlural[:1]) + cfg.targetNamePlural[1:]
		return inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[string]]{
			Value: &inspectioncore.AutocompleteResult[string]{
				Values: []string{},
				Error:  "",
				Hint:   fmt.Sprintf("%s are suggested after the project ID, cluster name, and location are provided.", capitalizedPlural),
			},
			DependencyDigest: currentDigest,
		}, nil
	}

	errorString := ""
	hintString := ""
	if endTime.Before(time.Now().Add(-time.Hour * 24 * 30 * 24)) {
		hintString = fmt.Sprintf("The end time is more than 24 months ago. Suggested %s may not be complete.", cfg.targetNamePlural)
	}

	client, err := cf.MonitoringMetricClient(ctx, googlecloud.Project(cluster.ProjectID))
	if err != nil {
		return prevValue, fmt.Errorf("failed to create monitoring metric client: %w", err)
	}

	ctx = optionInjector.InjectToCallContext(ctx, googlecloud.Project(cluster.ProjectID))
	filter := fmt.Sprintf(`metric.type="%s" AND resource.type="%s" AND resource.labels.cluster_name="%s" AND resource.labels.location="%s"`, metricsType, cfg.resourceType, cluster.ClusterName, cluster.Location)
	groupByKey := "resource.labels." + cfg.resourceLabelKey
	values, err := googlecloud.QueryDistinctStringLabelValuesFromMetrics(ctx, client, cluster.ProjectID, filter, startTime, endTime, groupByKey, cfg.resourceLabelKey)
	if err != nil {
		errorString = err.Error()
	}
	if hintString == "" && errorString == "" && len(values) == 0 {
		hintString = fmt.Sprintf("No %s found between %s and %s. It is highly likely that the time range is incorrect. Please verify the time range, or proceed by manually entering the %s.", cfg.targetNamePlural, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), cfg.targetNameSingular)
	}
	return inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[string]]{
		DependencyDigest: currentDigest,
		Value: &inspectioncore.AutocompleteResult[string]{
			Values: values,
			Error:  errorString,
			Hint:   hintString,
		},
	}, nil
}

var AutocompleteNamespacesTask = inspectiontaskbase.NewGlobalCachedTask(k8scommon.AutocompleteNamespacesTaskID, []coretask.Dependency{
	k8scommon.ClusterIdentityTaskID.Ref(),
	gcpcommon.InputStartTimeTaskID.Ref(),
	gcpcommon.InputEndTimeTaskID.Ref(),
	gcpcommon.APIClientFactoryTaskID.Ref(),
	gcpcommon.APIClientCallOptionsInjectorTaskID.Ref(),
	k8scommon.AutocompleteMetricsK8sContainerTaskID.Ref(),
}, func(ctx context.Context, prevValue inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[string]]) (inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[string]], error) {
	metricsType := coretask.GetTaskResult(ctx, k8scommon.AutocompleteMetricsK8sContainerTaskID.Ref())
	return queryClusterScopedAutocompleteMetrics(ctx, prevValue, metricsType, clusterScopedAutocompleteConfig{
		resourceType:       "k8s_container",
		resourceLabelKey:   "namespace_name",
		targetNameSingular: "namespace name",
		targetNamePlural:   "namespace names",
	})
})

var AutocompletePodNamesTask = inspectiontaskbase.NewGlobalCachedTask(k8scommon.AutocompletePodNamesTaskID, []coretask.Dependency{
	k8scommon.ClusterIdentityTaskID.Ref(),
	gcpcommon.InputStartTimeTaskID.Ref(),
	gcpcommon.InputEndTimeTaskID.Ref(),
	gcpcommon.APIClientFactoryTaskID.Ref(),
	gcpcommon.APIClientCallOptionsInjectorTaskID.Ref(),
	k8scommon.AutocompleteMetricsK8sContainerTaskID.Ref(),
}, func(ctx context.Context, prevValue inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[string]]) (inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[string]], error) {
	metricsType := coretask.GetTaskResult(ctx, k8scommon.AutocompleteMetricsK8sContainerTaskID.Ref())
	return queryClusterScopedAutocompleteMetrics(ctx, prevValue, metricsType, clusterScopedAutocompleteConfig{
		resourceType:       "k8s_container",
		resourceLabelKey:   "pod_name",
		targetNameSingular: "pod name",
		targetNamePlural:   "pod names",
	})
})

var AutocompleteNodeNamesTask = inspectiontaskbase.NewGlobalCachedTask(k8scommon.AutocompleteNodeNamesTaskID, []coretask.Dependency{
	k8scommon.ClusterIdentityTaskID.Ref(),
	gcpcommon.InputStartTimeTaskID.Ref(),
	gcpcommon.InputEndTimeTaskID.Ref(),
	gcpcommon.APIClientFactoryTaskID.Ref(),
	gcpcommon.APIClientCallOptionsInjectorTaskID.Ref(),
	k8scommon.AutocompleteMetricsK8sNodeTaskID.Ref(),
}, func(ctx context.Context, prevValue inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[string]]) (inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[string]], error) {
	metricsType := coretask.GetTaskResult(ctx, k8scommon.AutocompleteMetricsK8sNodeTaskID.Ref())
	return queryClusterScopedAutocompleteMetrics(ctx, prevValue, metricsType, clusterScopedAutocompleteConfig{
		resourceType:       "k8s_node",
		resourceLabelKey:   "node_name",
		targetNameSingular: "node name",
		targetNamePlural:   "node names",
	})
})
