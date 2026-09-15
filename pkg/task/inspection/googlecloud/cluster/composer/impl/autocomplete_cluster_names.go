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

// AutocompleteComposerClusterNamesTask is an implementation for k8scommon.AutocompleteClusterNamesTaskID
// the task returns GKE cluster name where the provided Composer environment is running.
var AutocompleteComposerClusterNamesTask = inspectiontaskbase.NewGlobalCachedTask(composercluster.AutocompleteComposerClusterNamesTaskID, []coretask.Dependency{
	composercluster.ComposerEnvironmentClusterFinderTaskID.Ref(),
	gcpcommon.InputProjectIdTaskID.Ref(),
	gcpcommon.InputLocationsTaskID.Ref(),
	composercluster.InputComposerEnvironmentNameTaskID.Ref(),
	composercluster.AutocompleteComposerEnvironmentIdentityTaskID.Ref(),
	gcpcommon.InputStartTimeTaskID.Ref(),
	gcpcommon.InputEndTimeTaskID.Ref(),
}, func(ctx context.Context, prevValue inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity]]) (inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity]], error) {

	projectID := coretask.GetTaskResult(ctx, gcpcommon.InputProjectIdTaskID.Ref())
	environment := coretask.GetTaskResult(ctx, composercluster.InputComposerEnvironmentNameTaskID.Ref())
	location := coretask.GetTaskResult(ctx, gcpcommon.InputLocationsTaskID.Ref())
	startTime := coretask.GetTaskResult(ctx, gcpcommon.InputStartTimeTaskID.Ref())
	endTime := coretask.GetTaskResult(ctx, gcpcommon.InputEndTimeTaskID.Ref())

	dependencyDigest := fmt.Sprintf("%s-%s-%s-%d-%d", projectID, environment, location, startTime.Unix(), endTime.Unix())

	// when the user is inputing these information, abort
	isWIP := projectID == "" || environment == ""
	if isWIP {
		return inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity]]{
			DependencyDigest: dependencyDigest,
			Value: &inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity]{
				Values: []k8scommon.GoogleCloudClusterIdentity{},
				Error:  "Project ID or Composer environment name is empty",
			},
		}, nil
	}

	if location == "" {
		return inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity]]{
			DependencyDigest: dependencyDigest,
			Value: &inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity]{
				Values: []k8scommon.GoogleCloudClusterIdentity{},
				Error:  "",
				Hint:   "Cluster names are suggested after the location is provided.",
			},
		}, nil
	}

	if environment != "" && dependencyDigest == prevValue.DependencyDigest {
		return prevValue, nil
	}

	clusterFinder := coretask.GetTaskResult(ctx, composercluster.ComposerEnvironmentClusterFinderTaskID.Ref())
	clusterNames, err := clusterFinder.GetGKEClusterNames(ctx, projectID, location, environment, startTime, endTime)
	if err != nil {
		if errors.Is(err, composercluster.ErrEnvironmentClusterNotFound) {
			return inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity]]{
				DependencyDigest: dependencyDigest,
				Value: &inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity]{
					Values: []k8scommon.GoogleCloudClusterIdentity{},
					Error: `Not found. It works for the clusters existed in the past but make sure the cluster name is right if you believe the cluster should be there.
Note: Composer 3 is not running on your GKE cluster. Please remove all Kubernetes/GKE queries from the previous section.`,
				},
			}, nil
		}
		return inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity]]{
			DependencyDigest: dependencyDigest,
			Value: &inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity]{
				Values: []k8scommon.GoogleCloudClusterIdentity{},
				Error:  "Failed to fetch the list GKE cluster. Please confirm if the Project ID is correct, or retry later",
			},
		}, nil
	}

	identities := make([]k8scommon.GoogleCloudClusterIdentity, len(clusterNames))
	for i, clusterName := range clusterNames {
		identities[i] = k8scommon.GoogleCloudClusterIdentity{
			ClusterName: clusterName,
			ProjectID:   projectID,
			Location:    location,
		}
	}

	return inspectiontaskbase.CacheableTaskResult[*inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity]]{
		DependencyDigest: dependencyDigest,
		Value: &inspectioncore.AutocompleteResult[k8scommon.GoogleCloudClusterIdentity]{
			Values: identities,
		},
	}, nil
},
	coretask.WithSelectionPriority(1000), // Setting higher priority compared to the default autocomplete cluster name finder to override it. Composer cluster finder is currently overriding the common autocomplete cluster name finder using Cloud Monitoring to compare the environment label name.
)
