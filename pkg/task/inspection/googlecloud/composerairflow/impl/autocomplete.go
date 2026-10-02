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

// autocompleteComposerComponentsTask suggests available Composer components from Cloud Monitoring metrics.
var autocompleteComposerComponentsTask = inspectiontaskbase.DefineCachedTask(
	composerairflow.AutocompleteComposerComponentsTaskID,
	func(b *coretask.Binder) inspectiontaskbase.CachedTaskSpec[*inspectioncore.AutocompleteResult[string]] {
		clusterIdentity := coretask.Use(b, composercluster.ClusterIdentityTaskID.Ref())
		startTime := coretask.Use(b, gcpcommon.InputStartTimeTaskID.Ref())
		endTime := coretask.Use(b, gcpcommon.InputEndTimeTaskID.Ref())
		environmentName := coretask.Use(b, composercluster.InputComposerEnvironmentNameTaskID.Ref())
		cf := coretask.Use(b, gcpcommon.APIClientFactoryTaskID.Ref())
		optionInjector := coretask.Use(b, gcpcommon.APIClientCallOptionsInjectorTaskID.Ref())

		return inspectiontaskbase.CachedTaskSpec[*inspectioncore.AutocompleteResult[string]]{
			Scope: inspectiontaskbase.CacheScopeGlobal,
			InputDigest: func(ctx context.Context) string {
				ci := clusterIdentity.Get(ctx)
				st := startTime.Get(ctx)
				et := endTime.Get(ctx)
				return fmt.Sprintf("%s-%s-%s-%s-%d-%d", ci.ProjectID, ci.Location, environmentName.Get(ctx), "logging.googleapis.com/log_entry_count", st.Unix(), et.Unix())
			},
			Compute: func(ctx context.Context) (*inspectioncore.AutocompleteResult[string], error) {
				ci := clusterIdentity.Get(ctx)
				projectID := ci.ProjectID
				location := ci.Location
				envName := environmentName.Get(ctx)

				if projectID == "" || envName == "" || location == "" {
					return &inspectioncore.AutocompleteResult[string]{
						Values: []string{},
						Hint:   "Components are suggested after the project ID, location, and environment name are provided.",
					}, nil
				}

				client, err := cf.Get(ctx).MonitoringMetricClient(ctx, googlecloud.Project(projectID))
				if err != nil {
					return nil, fmt.Errorf("failed to create monitoring metric client: %w", err)
				}

				callCtx := optionInjector.Get(ctx).InjectToCallContext(ctx, googlecloud.Project(projectID))

				st := startTime.Get(ctx)
				et := endTime.Get(ctx)
				filter := fmt.Sprintf(`resource.type = "cloud_composer_environment" AND metric.type = "logging.googleapis.com/log_entry_count" AND resource.labels.environment_name = "%s" AND resource.labels.location = "%s"`, envName, location)

				errorString := ""
				hintString := ""
				metricsLabels, err := googlecloud.QueryResourceLabelsFromMetrics(callCtx, client, projectID, filter, st, et, []string{"metric.label.log"})
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

				return &inspectioncore.AutocompleteResult[string]{
					Values: components,
					Error:  errorString,
					Hint:   hintString,
				}, nil
			},
		}
	},
)
