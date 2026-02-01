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

package googlecloudclustercomposer_impl

import (
	"context"
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// AutocompleteComposerEnvironmentIdentityTask is the task that autocompletes composer environment identities.
var AutocompleteComposerEnvironmentIdentityTask = inspectiontaskbase.NewCachedTask(googlecloudclustercomposer_contract.AutocompleteComposerEnvironmentIdentityTaskID, []taskid.UntypedTaskReference{
	googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
	googlecloudcommon_contract.InputStartTimeTaskID.Ref(),
	googlecloudcommon_contract.InputEndTimeTaskID.Ref(),
	googlecloudcommon_contract.APIClientFactoryTaskID.Ref(),
	googlecloudcommon_contract.APIClientCallOptionsInjectorTaskID.Ref(),
}, func(ctx context.Context, prevValue inspectiontaskbase.CacheableTaskResult[*inspectioncore_contract.AutocompleteResult[googlecloudclustercomposer_contract.ComposerEnvironmentIdentity]]) (inspectiontaskbase.CacheableTaskResult[*inspectioncore_contract.AutocompleteResult[googlecloudclustercomposer_contract.ComposerEnvironmentIdentity]], error) {
	projectID := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
	startTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputStartTimeTaskID.Ref())
	endTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputEndTimeTaskID.Ref())
	cf := coretask.GetTaskResult(ctx, googlecloudcommon_contract.APIClientFactoryTaskID.Ref())
	optionInjector := coretask.GetTaskResult(ctx, googlecloudcommon_contract.APIClientCallOptionsInjectorTaskID.Ref())

	currentDigest := fmt.Sprintf("%s-%d-%d", projectID, startTime.Unix(), endTime.Unix())
	if currentDigest == prevValue.DependencyDigest {
		return prevValue, nil
	}
	if projectID == "" {
		return inspectiontaskbase.CacheableTaskResult[*inspectioncore_contract.AutocompleteResult[googlecloudclustercomposer_contract.ComposerEnvironmentIdentity]]{
			Value: &inspectioncore_contract.AutocompleteResult[googlecloudclustercomposer_contract.ComposerEnvironmentIdentity]{
				Values: []googlecloudclustercomposer_contract.ComposerEnvironmentIdentity{},
				Error:  "",
				Hint:   "Composer environments are suggested after the project ID is provided.",
			},
			DependencyDigest: currentDigest,
		}, nil
	}

	errorString := ""
	hintString := ""
	if endTime.Before(time.Now().Add(-time.Hour * 24 * 30 * 24)) {
		hintString = "The end time is more than 24 months ago. Suggested environment names may not be complete."
	}

	client, err := cf.MonitoringMetricClient(ctx, googlecloud.Project(projectID))
	if err != nil {
		return prevValue, fmt.Errorf("failed to create monitoring metric client: %w", err)
	}
	defer client.Close()

	ctx = optionInjector.InjectToCallContext(ctx, googlecloud.Project(projectID))
	// Using "composer.googleapis.com/environment/healthy" as a generic metric to find environments.
	filter := `metric.type="composer.googleapis.com/environment/healthy" AND resource.type="cloud_composer_environment"`
	// Use QueryResourceLabelsFromMetrics to get environment_name and location.
	metricsLabels, err := googlecloud.QueryResourceLabelsFromMetrics(ctx, client, projectID, filter, startTime, endTime, []string{"resource.label.environment_name", "resource.label.location"})
	if err != nil {
		errorString = err.Error()
	}

	if hintString == "" && errorString == "" && len(metricsLabels) == 0 {
		hintString = fmt.Sprintf("No Composer environments found between %s and %s. It is highly likely that the time range is incorrect. Please verify the time range, or proceed by manually entering the environment name.", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
	}

	identities := make([]googlecloudclustercomposer_contract.ComposerEnvironmentIdentity, 0, len(metricsLabels))
	for _, labels := range metricsLabels {
		envName := labels["environment_name"]
		location := labels["location"]
		if envName != "" && location != "" {
			identities = append(identities, googlecloudclustercomposer_contract.ComposerEnvironmentIdentity{
				ProjectID:       projectID,
				Location:        location,
				EnvironmentName: envName,
			})
		}
	}

	return inspectiontaskbase.CacheableTaskResult[*inspectioncore_contract.AutocompleteResult[googlecloudclustercomposer_contract.ComposerEnvironmentIdentity]]{
		DependencyDigest: currentDigest,
		Value: &inspectioncore_contract.AutocompleteResult[googlecloudclustercomposer_contract.ComposerEnvironmentIdentity]{
			Values: identities,
			Error:  errorString,
			Hint:   hintString,
		},
	}, nil
})

var AutocompleteLocationForComposerEnvironmentTask = inspectiontaskbase.NewCachedTask(googlecloudclustercomposer_contract.AutocompleteLocationForComposerEnvironmentTaskID, []taskid.UntypedTaskReference{
	googlecloudclustercomposer_contract.AutocompleteComposerEnvironmentIdentityTaskID.Ref(),
	googlecloudcommon_contract.InputProjectIdTaskID.Ref(),
	googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref(),
	googlecloudcommon_contract.InputStartTimeTaskID.Ref(),
	googlecloudcommon_contract.InputEndTimeTaskID.Ref(),
}, func(ctx context.Context, prevValue inspectiontaskbase.CacheableTaskResult[*inspectioncore_contract.AutocompleteResult[string]]) (inspectiontaskbase.CacheableTaskResult[*inspectioncore_contract.AutocompleteResult[string]], error) {
	projectID := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputProjectIdTaskID.Ref())
	environmentName := coretask.GetTaskResult(ctx, googlecloudclustercomposer_contract.InputComposerEnvironmentNameTaskID.Ref())
	startTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputStartTimeTaskID.Ref())
	endTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputEndTimeTaskID.Ref())
	identities := coretask.GetTaskResult(ctx, googlecloudclustercomposer_contract.AutocompleteComposerEnvironmentIdentityTaskID.Ref())

	currentDigest := fmt.Sprintf("%s-%s-%d-%d", projectID, environmentName, startTime.Unix(), endTime.Unix())
	if currentDigest == prevValue.DependencyDigest {
		return prevValue, nil
	}

	if projectID == "" {
		return inspectiontaskbase.CacheableTaskResult[*inspectioncore_contract.AutocompleteResult[string]]{
			Value: &inspectioncore_contract.AutocompleteResult[string]{
				Values: []string{},
				Error:  "",
				Hint:   "Locations are suggested after the project ID is provided.",
			},
			DependencyDigest: currentDigest,
		}, nil
	}

	if environmentName == "" {
		return inspectiontaskbase.CacheableTaskResult[*inspectioncore_contract.AutocompleteResult[string]]{
			Value: &inspectioncore_contract.AutocompleteResult[string]{
				Values: []string{},
				Error:  "",
				Hint:   "Locations are suggested after the environment name is provided.",
			},
			DependencyDigest: currentDigest,
		}, nil
	}

	if identities.Error != "" {
		return inspectiontaskbase.CacheableTaskResult[*inspectioncore_contract.AutocompleteResult[string]]{
			Value: &inspectioncore_contract.AutocompleteResult[string]{
				Values: []string{},
				Error:  identities.Error,
				Hint:   identities.Hint,
			},
			DependencyDigest: currentDigest,
		}, nil
	}

	locationsMap := make(map[string]struct{})
	for _, identity := range identities.Values {
		if identity.EnvironmentName == environmentName {
			locationsMap[identity.Location] = struct{}{}
		}
	}

	locations := make([]string, 0, len(locationsMap))
	for location := range locationsMap {
		locations = append(locations, location)
	}

	return inspectiontaskbase.CacheableTaskResult[*inspectioncore_contract.AutocompleteResult[string]]{
		Value: &inspectioncore_contract.AutocompleteResult[string]{
			Values: locations,
			Error:  "",
			Hint:   identities.Hint,
		},
		DependencyDigest: currentDigest,
	}, nil
}, inspectioncore_contract.InspectionTypeLabel(googlecloudclustercomposer_contract.InspectionTypeId),
	coretask.WithSelectionPriority(1000),
)
