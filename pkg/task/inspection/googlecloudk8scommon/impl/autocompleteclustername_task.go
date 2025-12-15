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

// AutocompleteClusterNamesMetricsTypeTask is the task to provide the default metrics type to collect the cluster names.
// The resource type "k8s_container" must be available on the returned metrics type.
// This task is overriden in GKE clusters.
var AutocompleteClusterNamesMetricsTypeTask = coretask.NewTask(googlecloudk8scommon_contract.AutocompleteClusterNamesMetricsTypeTaskID, []taskid.UntypedTaskReference{}, func(ctx context.Context) (string, error) {
	// logging.googleapis.com/log_entry_count is better from the perspective of KHI's purpose, but use container metrics for longer retention period(24 months).
	return "kubernetes.io/anthos/container/uptime", nil
})

var AutocompleteClusterNamesTask = inspectiontaskbase.NewCachedTask(googlecloudk8scommon_contract.AutocompleteClusterNamesTaskID, []taskid.UntypedTaskReference{
	googlecloudk8scommon_contract.ClusterNamePrefixTaskID,
	googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
	googlecloudcommon_contract.InputStartTimeTaskID.Ref(),
	googlecloudcommon_contract.InputEndTimeTaskID.Ref(),
	googlecloudk8scommon_contract.AutocompleteClusterNamesMetricsTypeTaskID.Ref(),
	googlecloudcommon_contract.APIClientFactoryTaskID.Ref(),
	googlecloudcommon_contract.APIClientCallOptionsInjectorTaskID.Ref(),
}, func(ctx context.Context, prevValue inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]) (inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList], error) {
	clusterNamePrefix := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.ClusterNamePrefixTaskID)
	projectID := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
	startTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputStartTimeTaskID.Ref())
	endTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputEndTimeTaskID.Ref())
	metricsType := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.AutocompleteClusterNamesMetricsTypeTaskID.Ref())
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
	if clusterNamePrefix != "" {
		filteredClusters := make([]string, 0, len(clusterNames))
		for _, clusterName := range clusterNames {
			if strings.HasPrefix(clusterName, clusterNamePrefix) {
				filteredClusters = append(filteredClusters, clusterName)
			}
		}
		clusterNames = filteredClusters
	}
	if hintString == "" && errorString == "" && len(clusterNames) == 0 {
		hintString = fmt.Sprintf("No cluster names found between %s and %s. It is highly likely that the time range is incorrect. Please verify the time range, or proceed by manually entering the cluster name.", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
	}
	return inspectiontaskbase.CacheableTaskResult[*googlecloudk8scommon_contract.AutocompleteClusterNameList]{
		DependencyDigest: currentDigest,
		Value: &googlecloudk8scommon_contract.AutocompleteClusterNameList{
			ClusterNames: clusterNames,
			Error:        errorString,
			Hint:         hintString,
		},
	}, nil
})
