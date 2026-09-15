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

package composercluster_impl

import (
	"context"
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	composercluster "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/cluster/composer"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// AutocompleteComposerEnvironmentIdentityTask is the task that autocompletes composer environment identities.
var AutocompleteComposerEnvironmentIdentityTask = inspectiontaskbase.NewGlobalCachedTask(composercluster.AutocompleteComposerEnvironmentIdentityTaskID, []coretask.Dependency{
	gcpcommon.InputProjectIdTaskID.Ref(),
	gcpcommon.InputStartTimeTaskID.Ref(),
	gcpcommon.InputEndTimeTaskID.Ref(),
	gcpcommon.APIClientFactoryTaskID.Ref(),
	gcpcommon.APIClientCallOptionsInjectorTaskID.Ref(),
}, func(ctx context.Context, prevValue inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[composercluster.ComposerEnvironmentIdentity]]) (inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[composercluster.ComposerEnvironmentIdentity]], error) {
	projectID := coretask.GetTaskResult(ctx, gcpcommon.InputProjectIdTaskID.Ref())
	startTime := coretask.GetTaskResult(ctx, gcpcommon.InputStartTimeTaskID.Ref())
	endTime := coretask.GetTaskResult(ctx, gcpcommon.InputEndTimeTaskID.Ref())
	cf := coretask.GetTaskResult(ctx, gcpcommon.APIClientFactoryTaskID.Ref())
	optionInjector := coretask.GetTaskResult(ctx, gcpcommon.APIClientCallOptionsInjectorTaskID.Ref())

	currentDigest := fmt.Sprintf("%s-%d-%d", projectID, startTime.Unix(), endTime.Unix())
	if currentDigest == prevValue.DependencyDigest {
		return prevValue, nil
	}
	if projectID == "" {
		return inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[composercluster.ComposerEnvironmentIdentity]]{
			Value: &inspectioncore.AutocompleteResult[composercluster.ComposerEnvironmentIdentity]{
				Values: []composercluster.ComposerEnvironmentIdentity{},
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

	ctx = optionInjector.InjectToCallContext(ctx, googlecloud.Project(projectID))
	filter := `metric.type="composer.googleapis.com/environment/healthy" AND resource.type="cloud_composer_environment"`
	metricsLabels, err := googlecloud.QueryResourceLabelsFromMetrics(ctx, client, projectID, filter, startTime, endTime, []string{"resource.label.environment_name", "resource.label.location"})
	if err != nil {
		errorString = err.Error()
	}

	if hintString == "" && errorString == "" && len(metricsLabels) == 0 {
		hintString = fmt.Sprintf("No Composer environments found between %s and %s. It is highly likely that the time range is incorrect. Please verify the time range, or proceed by manually entering the environment name.", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
	}

	identities := make([]composercluster.ComposerEnvironmentIdentity, 0, len(metricsLabels))
	for _, labels := range metricsLabels {
		envName := labels["environment_name"]
		location := labels["location"]
		if envName != "" && location != "" {
			identities = append(identities, composercluster.ComposerEnvironmentIdentity{
				ProjectID:       projectID,
				Location:        location,
				EnvironmentName: envName,
			})
		}
	}

	return inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[composercluster.ComposerEnvironmentIdentity]]{
		DependencyDigest: currentDigest,
		Value: &inspectioncore.AutocompleteResult[composercluster.ComposerEnvironmentIdentity]{
			Values: identities,
			Error:  errorString,
			Hint:   hintString,
		},
	}, nil
})

var AutocompleteLocationForComposerEnvironmentTask = inspectiontaskbase.NewGlobalCachedTask(composercluster.AutocompleteLocationForComposerEnvironmentTaskID, []coretask.Dependency{
	composercluster.AutocompleteComposerEnvironmentIdentityTaskID.Ref(),
	gcpcommon.InputProjectIdTaskID.Ref(),
	composercluster.InputComposerEnvironmentNameTaskID.Ref(),
	gcpcommon.InputStartTimeTaskID.Ref(),
	gcpcommon.InputEndTimeTaskID.Ref(),
}, func(ctx context.Context, prevValue inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[string]]) (inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[string]], error) {
	projectID := coretask.GetTaskResult(ctx, gcpcommon.InputProjectIdTaskID.Ref())
	environmentName := coretask.GetTaskResult(ctx, composercluster.InputComposerEnvironmentNameTaskID.Ref())
	startTime := coretask.GetTaskResult(ctx, gcpcommon.InputStartTimeTaskID.Ref())
	endTime := coretask.GetTaskResult(ctx, gcpcommon.InputEndTimeTaskID.Ref())
	identities := coretask.GetTaskResult(ctx, composercluster.AutocompleteComposerEnvironmentIdentityTaskID.Ref())

	currentDigest := fmt.Sprintf("%s-%s-%d-%d", projectID, environmentName, startTime.Unix(), endTime.Unix())
	if currentDigest == prevValue.DependencyDigest {
		return prevValue, nil
	}

	if projectID == "" {
		return inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[string]]{
			Value: &inspectioncore.AutocompleteResult[string]{
				Values: []string{},
				Error:  "",
				Hint:   "Locations are suggested after the project ID is provided.",
			},
			DependencyDigest: currentDigest,
		}, nil
	}

	if environmentName == "" {
		return inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[string]]{
			Value: &inspectioncore.AutocompleteResult[string]{
				Values: []string{},
				Error:  "",
				Hint:   "Locations are suggested after the environment name is provided.",
			},
			DependencyDigest: currentDigest,
		}, nil
	}

	if identities.Error != "" {
		return inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[string]]{
			Value: &inspectioncore.AutocompleteResult[string]{
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

	return inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[string]]{
		Value: &inspectioncore.AutocompleteResult[string]{
			Values: locations,
			Error:  "",
			Hint:   identities.Hint,
		},
		DependencyDigest: currentDigest,
	}, nil
},
	coretask.WithSelectionPriority(1000),
)
