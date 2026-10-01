// Copyright 2026 Google LLC
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
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	composercluster "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/cluster/composer"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// Module declares the Cloud Composer inspection type and the tasks for discovering and selecting Composer clusters.
var Module = coreinspection.Module{
	Name: "googlecloud/cluster/composer",
	Scope: coreinspection.Scope{
		inspectioncore.InspectionTypeLabelKeyEnvironment:  "googlecloud",
		inspectioncore.InspectionTypeLabelKeyBasePlatform: "kubernetes",
		gcpcommon.InspectionTypeLabelKeyClusterType:       "gke",
		gcpcommon.InspectionTypeLabelKeyProduct:           "composer",
	},
	InspectionTypes: []coreinspection.InspectionType{composercluster.ComposerInspectionType},
	Tasks: []coretask.UntypedTask{
		clusterIdentityAliasTask,
		composerEnvironmentClusterFinderTask,
		autocompleteComposerClusterNamesTask,
		autocompleteComposerEnvironmentIdentityTask,
		autocompleteLocationForComposerEnvironmentTask,
		inputComposerEnvironmentNameTask,
	},
}
