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
	"errors"
	"fmt"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	composercluster "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/cluster/composer"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"

	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// autocompleteComposerClusterNamesTask is an implementation for k8scommon.AutocompleteClusterNamesTaskID.
// The task returns GKE cluster name where the provided Composer environment is running.
var autocompleteComposerClusterNamesTask = inspectiontaskbase.DefineCachedTask(
	composercluster.AutocompleteComposerClusterNamesTaskID,
	func(b *coretask.Binder) inspectiontaskbase.CachedTaskSpec[*inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity]] {
		clusterFinder := coretask.Use(b, composercluster.ComposerEnvironmentClusterFinderTaskID.Ref())
		projectID := coretask.Use(b, gcpcommon.InputProjectIdTaskID.Ref())
		location := coretask.Use(b, gcpcommon.InputLocationsTaskID.Ref())
		environment := coretask.Use(b, composercluster.InputComposerEnvironmentNameTaskID.Ref())
		startTime := coretask.Use(b, gcpcommon.InputStartTimeTaskID.Ref())
		endTime := coretask.Use(b, gcpcommon.InputEndTimeTaskID.Ref())

		return inspectiontaskbase.CachedTaskSpec[*inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity]]{
			Scope: inspectiontaskbase.CacheScopeGlobal,
			InputDigest: func(ctx context.Context) string {
				return fmt.Sprintf("%s-%s-%s-%d-%d", projectID.Get(ctx), environment.Get(ctx), location.Get(ctx), startTime.Get(ctx).Unix(), endTime.Get(ctx).Unix())
			},
			Compute: func(ctx context.Context) (*inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity], error) {
				pid := projectID.Get(ctx)
				env := environment.Get(ctx)
				loc := location.Get(ctx)
				st := startTime.Get(ctx)
				et := endTime.Get(ctx)

				// when the user is inputing these information, abort
				isWIP := pid == "" || env == ""
				if isWIP {
					return &inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity]{
						Values: []k8scommon.GoogleCloudClusterIdentity{},
						Error:  "Project ID or Composer environment name is empty",
					}, nil
				}

				if loc == "" {
					return &inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity]{
						Values: []k8scommon.GoogleCloudClusterIdentity{},
						Error:  "",
						Hint:   "Cluster names are suggested after the location is provided.",
					}, nil
				}

				finder := clusterFinder.Get(ctx)
				clusterNames, err := finder.GetGKEClusterNames(ctx, pid, loc, env, st, et)
				if err != nil {
					if errors.Is(err, composercluster.ErrEnvironmentClusterNotFound) {
						return &inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity]{
							Values: []k8scommon.GoogleCloudClusterIdentity{},
							Error: `Not found. It works for the clusters existed in the past but make sure the cluster name is right if you believe the cluster should be there.
Note: Composer 3 is not running on your GKE cluster. Please remove all Kubernetes/GKE queries from the previous section.`,
						}, nil
					}
					return &inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity]{
						Values: []k8scommon.GoogleCloudClusterIdentity{},
						Error:  "Failed to fetch the list GKE cluster. Please confirm if the Project ID is correct, or retry later",
					}, nil
				}

				identities := make([]k8scommon.GoogleCloudClusterIdentity, len(clusterNames))
				for i, clusterName := range clusterNames {
					identities[i] = k8scommon.GoogleCloudClusterIdentity{
						ClusterName: clusterName,
						ProjectID:   pid,
						Location:    loc,
					}
				}

				return &inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity]{
					Values: identities,
				}, nil
			},
		}
	},
	coretask.WithSelectionPriority(1000), // Setting higher priority compared to the default autocomplete cluster name finder to override it. Composer cluster finder is currently overriding the common autocomplete cluster name finder using Cloud Monitoring to compare the environment label name.
)
