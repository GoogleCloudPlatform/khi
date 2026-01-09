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

package googlecloudk8scommon_impl

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
)

// AutocompleteMetricsK8sContainerTask is the task to provide the default metrics type to collect the cluster names.
// The resource type "k8s_container" must be available on the returned metrics type.
// This task is overriden in GKE clusters.
var AutocompleteMetricsK8sContainerTask = coretask.NewTask(googlecloudk8scommon_contract.AutocompleteMetricsK8sContainerTaskID, []taskid.UntypedTaskReference{}, func(ctx context.Context) (string, error) {
	// logging.googleapis.com/log_entry_count is better from the perspective of KHI's purpose, but use container metrics for longer retention period(24 months).
	return "kubernetes.io/anthos/up", nil
})

var AutocompleteMetricsK8sNodeTask = coretask.NewTask(googlecloudk8scommon_contract.AutocompleteMetricsK8sNodeTaskID, []taskid.UntypedTaskReference{}, func(ctx context.Context) (string, error) {
	return "kubernetes.io/anthos/up", nil
})

var AutocompleteClusterNamesTask = inspectiontaskbase.NewCachedTask(googlecloudk8scommon_contract.AutocompleteClusterNamesTaskID, []taskid.UntypedTaskReference{
	googlecloudk8scommon_contract.ClusterNamePrefixTaskRef,
	googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
	googlecloudcommon_contract.InputStartTimeTaskID.Ref(),
	googlecloudcommon_contract.InputEndTimeTaskID.Ref(),
	googlecloudk8scommon_contract.AutocompleteMetricsK8sContainerTaskID.Ref(),
	googlecloudcommon_contract.APIClientFactoryTaskID.Ref(),
	googlecloudcommon_contract.APIClientCallOptionsInjectorTaskID.Ref(),
}, func(ctx context.Context, prevValue inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteResult]) (inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteResult], error) {
	clusterNamePrefix := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.ClusterNamePrefixTaskRef)
	projectID := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
	startTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputStartTimeTaskID.Ref())
	endTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputEndTimeTaskID.Ref())
	metricsType := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.AutocompleteMetricsK8sContainerTaskID.Ref())
	cf := coretask.GetTaskResult(ctx, googlecloudcommon_contract.APIClientFactoryTaskID.Ref())
	optionInjector := coretask.GetTaskResult(ctx, googlecloudcommon_contract.APIClientCallOptionsInjectorTaskID.Ref())

	currentDigest := fmt.Sprintf("%s-%s-%d-%d", clusterNamePrefix, projectID, startTime.Unix(), endTime.Unix())
	if projectID != "" && currentDigest == prevValue.DependencyDigest {
		return prevValue, nil
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
	defer client.Close()

	ctx = optionInjector.InjectToCallContext(ctx, googlecloud.Project(projectID))
	filter := fmt.Sprintf(`metric.type="%s" AND resource.type="k8s_container"`, metricsType)
	clusterNames, err := googlecloud.QueryDistinctLabelValuesFromMetrics(ctx, client, projectID, filter, startTime, endTime, "resource.label.cluster_name", "cluster_name")
	if err != nil {
		errorString = err.Error()
	}
	clusterNames = filterAndTrimPrefixFromClusterNames(clusterNames, clusterNamePrefix)
	if hintString == "" && errorString == "" && len(clusterNames) == 0 {
		hintString = fmt.Sprintf("No cluster names found between %s and %s. It is highly likely that the time range is incorrect. Please verify the time range, or proceed by manually entering the cluster name.", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
	}
	return inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteResult]{
		DependencyDigest: currentDigest,
		Value: &googlecloudk8scommon_contract.AutocompleteResult{
			Values: clusterNames,
			Error:  errorString,
			Hint:   hintString,
		},
	}, nil
})

// filterAndTrimPrefixFromClusterNames filters cluster names by prefix and trims the prefix from the filtered cluster names.
func filterAndTrimPrefixFromClusterNames(clusterNames []string, prefix string) []string {
	filteredClusters := make([]string, 0, len(clusterNames))
	for _, clusterName := range clusterNames {
		if prefix == "" {
			if !strings.Contains(clusterName, "/") {
				filteredClusters = append(filteredClusters, clusterName)
			}
		} else if strings.HasPrefix(clusterName, prefix) {
			filteredClusters = append(filteredClusters, strings.TrimPrefix(clusterName, prefix))
		}
	}
	return filteredClusters
}

var AutocompleteNamespacesTask = inspectiontaskbase.NewCachedTask(googlecloudk8scommon_contract.AutocompleteNamespacesTaskID, []taskid.UntypedTaskReference{
	googlecloudk8scommon_contract.InputClusterNameTaskID.Ref(),
	googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
	googlecloudcommon_contract.InputStartTimeTaskID.Ref(),
	googlecloudcommon_contract.InputEndTimeTaskID.Ref(),
	googlecloudcommon_contract.APIClientFactoryTaskID.Ref(),
	googlecloudcommon_contract.APIClientCallOptionsInjectorTaskID.Ref(),
	googlecloudk8scommon_contract.AutocompleteMetricsK8sContainerTaskID.Ref(),
}, func(ctx context.Context, prevValue inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteResult]) (inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteResult], error) {
	projectID := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
	startTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputStartTimeTaskID.Ref())
	endTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputEndTimeTaskID.Ref())
	clusterName := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.InputClusterNameTaskID.Ref())
	metricsType := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.AutocompleteMetricsK8sContainerTaskID.Ref())
	cf := coretask.GetTaskResult(ctx, googlecloudcommon_contract.APIClientFactoryTaskID.Ref())
	optionInjector := coretask.GetTaskResult(ctx, googlecloudcommon_contract.APIClientCallOptionsInjectorTaskID.Ref())

	currentDigest := fmt.Sprintf("%s-%s-%d-%d", clusterName, projectID, startTime.Unix(), endTime.Unix())
	if projectID != "" && currentDigest == prevValue.DependencyDigest {
		return prevValue, nil
	}

	errorString := ""
	hintString := ""
	if endTime.Before(time.Now().Add(-time.Hour * 24 * 30 * 24)) {
		hintString = "The end time is more than 24 months ago. Suggested namespace names may not be complete."
	}

	client, err := cf.MonitoringMetricClient(ctx, googlecloud.Project(projectID))
	if err != nil {
		return prevValue, fmt.Errorf("failed to create monitoring metric client: %w", err)
	}
	defer client.Close()

	ctx = optionInjector.InjectToCallContext(ctx, googlecloud.Project(projectID))
	filter := fmt.Sprintf(`metric.type="%s" AND resource.type="k8s_container" AND resource.label.cluster_name="%s"`, metricsType, clusterName)
	namespaces, err := googlecloud.QueryDistinctLabelValuesFromMetrics(ctx, client, projectID, filter, startTime, endTime, "resource.label.namespace_name", "namespace_name")
	if err != nil {
		errorString = err.Error()
	}
	if hintString == "" && errorString == "" && len(namespaces) == 0 {
		hintString = fmt.Sprintf("No namespace names found between %s and %s. It is highly likely that the time range is incorrect. Please verify the time range, or proceed by manually entering the namespace name.", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
	}
	return inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteResult]{
		DependencyDigest: currentDigest,
		Value: &googlecloudk8scommon_contract.AutocompleteResult{
			Values: namespaces,
			Error:  errorString,
			Hint:   hintString,
		},
	}, nil
})

var AutocompletePodNamesTask = inspectiontaskbase.NewCachedTask(googlecloudk8scommon_contract.AutocompletePodNamesTaskID, []taskid.UntypedTaskReference{
	googlecloudk8scommon_contract.InputClusterNameTaskID.Ref(),
	googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
	googlecloudcommon_contract.InputStartTimeTaskID.Ref(),
	googlecloudcommon_contract.InputEndTimeTaskID.Ref(),
	googlecloudcommon_contract.APIClientFactoryTaskID.Ref(),
	googlecloudcommon_contract.APIClientCallOptionsInjectorTaskID.Ref(),
	googlecloudk8scommon_contract.AutocompleteMetricsK8sContainerTaskID.Ref(),
}, func(ctx context.Context, prevValue inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteResult]) (inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteResult], error) {
	projectID := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
	startTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputStartTimeTaskID.Ref())
	endTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputEndTimeTaskID.Ref())
	clusterName := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.InputClusterNameTaskID.Ref())
	metricsType := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.AutocompleteMetricsK8sContainerTaskID.Ref())
	cf := coretask.GetTaskResult(ctx, googlecloudcommon_contract.APIClientFactoryTaskID.Ref())
	optionInjector := coretask.GetTaskResult(ctx, googlecloudcommon_contract.APIClientCallOptionsInjectorTaskID.Ref())

	currentDigest := fmt.Sprintf("%s-%s-%d-%d", clusterName, projectID, startTime.Unix(), endTime.Unix())
	if projectID != "" && currentDigest == prevValue.DependencyDigest {
		return prevValue, nil
	}

	errorString := ""
	hintString := ""
	if endTime.Before(time.Now().Add(-time.Hour * 24 * 30 * 24)) {
		hintString = "The end time is more than 24 months ago. Suggested pod names may not be complete."
	}

	client, err := cf.MonitoringMetricClient(ctx, googlecloud.Project(projectID))
	if err != nil {
		return prevValue, fmt.Errorf("failed to create monitoring metric client: %w", err)
	}
	defer client.Close()

	ctx = optionInjector.InjectToCallContext(ctx, googlecloud.Project(projectID))
	filter := fmt.Sprintf(`metric.type="%s" AND resource.type="k8s_container" AND resource.label.cluster_name="%s"`, metricsType, clusterName)
	podNames, err := googlecloud.QueryDistinctLabelValuesFromMetrics(ctx, client, projectID, filter, startTime, endTime, "resource.label.pod_name", "pod_name")
	if err != nil {
		errorString = err.Error()
	}
	if hintString == "" && errorString == "" && len(podNames) == 0 {
		hintString = fmt.Sprintf("No pod names found between %s and %s. It is highly likely that the time range is incorrect. Please verify the time range, or proceed by manually entering the pod name.", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
	}
	return inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteResult]{
		DependencyDigest: currentDigest,
		Value: &googlecloudk8scommon_contract.AutocompleteResult{
			Values: podNames,
			Error:  errorString,
			Hint:   hintString,
		},
	}, nil
})

var AutocompleteNodeNamesTask = inspectiontaskbase.NewCachedTask(googlecloudk8scommon_contract.AutocompleteNodeNamesTaskID, []taskid.UntypedTaskReference{
	googlecloudk8scommon_contract.InputClusterNameTaskID.Ref(),
	googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
	googlecloudcommon_contract.InputStartTimeTaskID.Ref(),
	googlecloudcommon_contract.InputEndTimeTaskID.Ref(),
	googlecloudcommon_contract.APIClientFactoryTaskID.Ref(),
	googlecloudcommon_contract.APIClientCallOptionsInjectorTaskID.Ref(),
	googlecloudk8scommon_contract.AutocompleteMetricsK8sNodeTaskID.Ref(),
}, func(ctx context.Context, prevValue inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteResult]) (inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteResult], error) {
	projectID := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
	startTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputStartTimeTaskID.Ref())
	endTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputEndTimeTaskID.Ref())
	clusterName := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.InputClusterNameTaskID.Ref())
	metricsType := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.AutocompleteMetricsK8sNodeTaskID.Ref())
	cf := coretask.GetTaskResult(ctx, googlecloudcommon_contract.APIClientFactoryTaskID.Ref())
	optionInjector := coretask.GetTaskResult(ctx, googlecloudcommon_contract.APIClientCallOptionsInjectorTaskID.Ref())

	currentDigest := fmt.Sprintf("%s-%s-%d-%d", clusterName, projectID, startTime.Unix(), endTime.Unix())
	if projectID != "" && currentDigest == prevValue.DependencyDigest {
		return prevValue, nil
	}

	errorString := ""
	hintString := ""
	if endTime.Before(time.Now().Add(-time.Hour * 24 * 30 * 24)) {
		hintString = "The end time is more than 24 months ago. Suggested namespace names may not be complete."
	}

	client, err := cf.MonitoringMetricClient(ctx, googlecloud.Project(projectID))
	if err != nil {
		return prevValue, fmt.Errorf("failed to create monitoring metric client: %w", err)
	}
	defer client.Close()

	ctx = optionInjector.InjectToCallContext(ctx, googlecloud.Project(projectID))
	filter := fmt.Sprintf(`metric.type="%s" AND resource.type="k8s_node" AND resource.label.cluster_name="%s"`, metricsType, clusterName)
	nodes, err := googlecloud.QueryDistinctLabelValuesFromMetrics(ctx, client, projectID, filter, startTime, endTime, "resource.label.node_name", "node_name")
	if err != nil {
		errorString = err.Error()
	}
	if hintString == "" && errorString == "" && len(nodes) == 0 {
		hintString = fmt.Sprintf("No node names found between %s and %s. It is highly likely that the time range is incorrect. Please verify the time range, or proceed by manually entering the node name.", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
	}
	return inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteResult]{
		DependencyDigest: currentDigest,
		Value: &googlecloudk8scommon_contract.AutocompleteResult{
			Values: nodes,
			Error:  errorString,
			Hint:   hintString,
		},
	}, nil
})
