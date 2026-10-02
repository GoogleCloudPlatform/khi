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

// autocompleteComposerEnvironmentIdentityTask is the task that autocompletes composer environment identities.
var autocompleteComposerEnvironmentIdentityTask = inspectiontaskbase.DefineCachedTask(
	composercluster.AutocompleteComposerEnvironmentIdentityTaskID,
	func(b *coretask.Binder) inspectiontaskbase.CachedTaskSpec[*inspectioncore.AutocompleteResult[composercluster.ComposerEnvironmentIdentity]] {
		projectID := coretask.Use(b, gcpcommon.InputProjectIdTaskID.Ref())
		startTime := coretask.Use(b, gcpcommon.InputStartTimeTaskID.Ref())
		endTime := coretask.Use(b, gcpcommon.InputEndTimeTaskID.Ref())
		cf := coretask.Use(b, gcpcommon.APIClientFactoryTaskID.Ref())
		optionInjector := coretask.Use(b, gcpcommon.APIClientCallOptionsInjectorTaskID.Ref())

		return inspectiontaskbase.CachedTaskSpec[*inspectioncore.AutocompleteResult[composercluster.ComposerEnvironmentIdentity]]{
			Scope: inspectiontaskbase.CacheScopeGlobal,
			InputDigest: func(ctx context.Context) string {
				return fmt.Sprintf("%s-%d-%d", projectID.Get(ctx), startTime.Get(ctx).Unix(), endTime.Get(ctx).Unix())
			},
			Compute: func(ctx context.Context) (*inspectioncore.AutocompleteResult[composercluster.ComposerEnvironmentIdentity], error) {
				pid := projectID.Get(ctx)
				if pid == "" {
					return &inspectioncore.AutocompleteResult[composercluster.ComposerEnvironmentIdentity]{
						Values: []composercluster.ComposerEnvironmentIdentity{},
						Error:  "",
						Hint:   "Composer environments are suggested after the project ID is provided.",
					}, nil
				}

				errorString := ""
				hintString := ""
				st := startTime.Get(ctx)
				et := endTime.Get(ctx)
				if et.Before(time.Now().Add(-time.Hour * 24 * 30 * 24)) {
					hintString = "The end time is more than 24 months ago. Suggested environment names may not be complete."
				}

				client, err := cf.Get(ctx).MonitoringMetricClient(ctx, googlecloud.Project(pid))
				if err != nil {
					return nil, fmt.Errorf("failed to create monitoring metric client: %w", err)
				}

				callCtx := optionInjector.Get(ctx).InjectToCallContext(ctx, googlecloud.Project(pid))
				filter := `metric.type="composer.googleapis.com/environment/healthy" AND resource.type="cloud_composer_environment"`
				metricsLabels, err := googlecloud.QueryResourceLabelsFromMetrics(callCtx, client, pid, filter, st, et, []string{"resource.label.environment_name", "resource.label.location"})
				if err != nil {
					errorString = err.Error()
				}

				if hintString == "" && errorString == "" && len(metricsLabels) == 0 {
					hintString = fmt.Sprintf("No Composer environments found between %s and %s. It is highly likely that the time range is incorrect. Please verify the time range, or proceed by manually entering the environment name.", st.Format(time.RFC3339), et.Format(time.RFC3339))
				}

				identities := make([]composercluster.ComposerEnvironmentIdentity, 0, len(metricsLabels))
				for _, labels := range metricsLabels {
					envName := labels["environment_name"]
					location := labels["location"]
					if envName != "" && location != "" {
						identities = append(identities, composercluster.ComposerEnvironmentIdentity{
							ProjectID:       pid,
							Location:        location,
							EnvironmentName: envName,
						})
					}
				}

				return &inspectioncore.AutocompleteResult[composercluster.ComposerEnvironmentIdentity]{
					Values: identities,
					Error:  errorString,
					Hint:   hintString,
				}, nil
			},
		}
	},
)

// autocompleteLocationForComposerEnvironmentTask suggests the locations of the selected Composer environment.
var autocompleteLocationForComposerEnvironmentTask = inspectiontaskbase.DefineCachedTask(
	composercluster.AutocompleteLocationForComposerEnvironmentTaskID,
	func(b *coretask.Binder) inspectiontaskbase.CachedTaskSpec[*inspectioncore.AutocompleteResult[string]] {
		identities := coretask.Use(b, composercluster.AutocompleteComposerEnvironmentIdentityTaskID.Ref())
		projectID := coretask.Use(b, gcpcommon.InputProjectIdTaskID.Ref())
		environmentName := coretask.Use(b, composercluster.InputComposerEnvironmentNameTaskID.Ref())
		startTime := coretask.Use(b, gcpcommon.InputStartTimeTaskID.Ref())
		endTime := coretask.Use(b, gcpcommon.InputEndTimeTaskID.Ref())

		return inspectiontaskbase.CachedTaskSpec[*inspectioncore.AutocompleteResult[string]]{
			Scope: inspectiontaskbase.CacheScopeGlobal,
			InputDigest: func(ctx context.Context) string {
				return fmt.Sprintf("%s-%s-%d-%d", projectID.Get(ctx), environmentName.Get(ctx), startTime.Get(ctx).Unix(), endTime.Get(ctx).Unix())
			},
			Compute: func(ctx context.Context) (*inspectioncore.AutocompleteResult[string], error) {
				pid := projectID.Get(ctx)
				if pid == "" {
					return &inspectioncore.AutocompleteResult[string]{
						Values: []string{},
						Error:  "",
						Hint:   "Locations are suggested after the project ID is provided.",
					}, nil
				}

				envName := environmentName.Get(ctx)
				if envName == "" {
					return &inspectioncore.AutocompleteResult[string]{
						Values: []string{},
						Error:  "",
						Hint:   "Locations are suggested after the environment name is provided.",
					}, nil
				}

				envIdentities := identities.Get(ctx)
				if envIdentities.Error != "" {
					return &inspectioncore.AutocompleteResult[string]{
						Values: []string{},
						Error:  envIdentities.Error,
						Hint:   envIdentities.Hint,
					}, nil
				}

				locationsMap := make(map[string]struct{})
				for _, identity := range envIdentities.Values {
					if identity.EnvironmentName == envName {
						locationsMap[identity.Location] = struct{}{}
					}
				}

				locations := make([]string, 0, len(locationsMap))
				for location := range locationsMap {
					locations = append(locations, location)
				}

				return &inspectioncore.AutocompleteResult[string]{
					Values: locations,
					Error:  "",
					Hint:   envIdentities.Hint,
				}, nil
			},
		}
	},
	coretask.WithSelectionPriority(1000),
)
