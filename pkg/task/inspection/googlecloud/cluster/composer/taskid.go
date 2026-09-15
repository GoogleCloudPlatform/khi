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

package composercluster

import (
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// GoogleCloudComposerTaskIDPrefix is the prefix for all task ids related to google cloud composer.
var GoogleCloudComposerTaskIDPrefix = "cloud.google.com/composer/"

// ClusterIdentityTaskID is the task id for aliasing the cluster identity.
var ClusterIdentityTaskID = taskid.NewDefaultImplementationID[k8scommon.GoogleCloudClusterIdentity](GoogleCloudComposerTaskIDPrefix + "cluster-identity")

// AutocompleteComposerClusterNamesTaskID is the task id for the task that autocompletes GKE cluster names created by Cloud Composer.
var AutocompleteComposerClusterNamesTaskID = taskid.NewImplementationID(k8scommon.AutocompleteClusterIdentityTaskID.Ref(), "composer")

// ComposerClusterNamePrefixTaskID is the task id for the task that returns the GKE cluster name prefix used by Cloud Composer.
var ComposerClusterNamePrefixTaskID = taskid.NewImplementationID(k8scommon.ClusterNamePrefixTaskRef, "composer")

// AutocompleteComposerEnvironmentNamesTaskID is the task id for the task that autocompletes composer environment names.
var AutocompleteComposerEnvironmentNamesTaskID taskid.TaskImplementationID[[]string] = taskid.NewDefaultImplementationID[[]string](GoogleCloudComposerTaskIDPrefix + "autocomplete/composer-environment-names")

// AutocompleteComposerEnvironmentIdentityTaskID is the task id for the task that autocompletes composer environment identities.
var AutocompleteComposerEnvironmentIdentityTaskID = taskid.NewDefaultImplementationID[*inspectioncore.AutocompleteResult[ComposerEnvironmentIdentity]](GoogleCloudComposerTaskIDPrefix + "autocomplete/composer-environment-identities")

// InputComposerEnvironmentNameTaskID is the task id for the task that inputs composer environment name.
var InputComposerEnvironmentNameTaskID taskid.TaskImplementationID[string] = taskid.NewDefaultImplementationID[string](GoogleCloudComposerTaskIDPrefix + "input/composer/environment_name")

// ComposerEnvironmentClusterFinderTaskID is the task id for injecting ComposerEnvironmentClusterFinder instance.
var ComposerEnvironmentClusterFinderTaskID = taskid.NewDefaultImplementationID[ComposerEnvironmentClusterFinder](GoogleCloudComposerTaskIDPrefix + "composer-environment-cluster-finder")

// AutocompleteLocationForComposerEnvironmentTaskID is the task id for the task that autocompletes GKE cluster location from Composer environments.
var AutocompleteLocationForComposerEnvironmentTaskID = taskid.NewImplementationID(gcpcommon.AutocompleteLocationTaskID.Ref(), "composer")
