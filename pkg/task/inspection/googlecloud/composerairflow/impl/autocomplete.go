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

package composerairflow_impl

import (
	"context"
	"fmt"
	"sort"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	composercluster "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/cluster/composer"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/composerairflow"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

var AutocompleteComposerComponentsTask = inspectiontaskbase.NewGlobalCachedTask(composerairflow.AutocompleteComposerComponentsTaskID, []coretask.Dependency{
	composercluster.ClusterIdentityTaskID.Ref(),
	gcpcommon.InputStartTimeTaskID.Ref(),
	gcpcommon.InputEndTimeTaskID.Ref(),
	composercluster.InputComposerEnvironmentNameTaskID.Ref(),
	gcpcommon.APIClientFactoryTaskID.Ref(),
	gcpcommon.APIClientCallOptionsInjectorTaskID.Ref(),
}, func(ctx context.Context, prevValue inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[string]]) (inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[string]], error) {
	clusterIdentity := coretask.GetTaskResult(ctx, composercluster.ClusterIdentityTaskID.Ref())
	projectID := clusterIdentity.ProjectID
	location := clusterIdentity.Location

	startTime := coretask.GetTaskResult(ctx, gcpcommon.InputStartTimeTaskID.Ref())
	endTime := coretask.GetTaskResult(ctx, gcpcommon.InputEndTimeTaskID.Ref())
	environmentName := coretask.GetTaskResult(ctx, composercluster.InputComposerEnvironmentNameTaskID.Ref())
	cf := coretask.GetTaskResult(ctx, gcpcommon.APIClientFactoryTaskID.Ref())
	optionInjector := coretask.GetTaskResult(ctx, gcpcommon.APIClientCallOptionsInjectorTaskID.Ref())

	currentDigest := fmt.Sprintf("%s-%s-%s-%s-%d-%d", projectID, location, environmentName, "logging.googleapis.com/log_entry_count", startTime.Unix(), endTime.Unix())
	if currentDigest == prevValue.DependencyDigest {
		return prevValue, nil
	}

	if projectID == "" || environmentName == "" || location == "" {
		return inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[string]]{
			Value: &inspectioncore.AutocompleteResult[string]{
				Values: []string{},
				Hint:   "Components are suggested after the project ID, location, and environment name are provided.",
			},
			DependencyDigest: currentDigest,
		}, nil
	}

	client, err := cf.MonitoringMetricClient(ctx, googlecloud.Project(projectID))
	if err != nil {
		return prevValue, fmt.Errorf("failed to create monitoring metric client: %w", err)
	}

	ctx = optionInjector.InjectToCallContext(ctx, googlecloud.Project(projectID))

	filter := fmt.Sprintf(`resource.type = "cloud_composer_environment" AND metric.type = "logging.googleapis.com/log_entry_count" AND resource.labels.environment_name = "%s" AND resource.labels.location = "%s"`, environmentName, location)

	errorString := ""
	hintString := ""
	metricsLabels, err := googlecloud.QueryResourceLabelsFromMetrics(ctx, client, projectID, filter, startTime, endTime, []string{"metric.label.log"})
	if err != nil {
		errorString = err.Error()
	}

	componentsMap := make(map[string]struct{})
	for _, labels := range metricsLabels {
		if logName, ok := labels["log"]; ok && logName != "" {
			componentsMap[logName] = struct{}{}
		}
	}

	components := make([]string, 0, len(componentsMap))
	for comp := range componentsMap {
		components = append(components, comp)
	}
	sort.Strings(components)

	if hintString == "" && errorString == "" && len(components) == 0 {
		hintString = "No components found for the specified environment and time range."
	}

	return inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[string]]{
		DependencyDigest: currentDigest,
		Value: &inspectioncore.AutocompleteResult[string]{
			Values: components,
			Error:  errorString,
			Hint:   hintString,
		},
	}, nil
})
