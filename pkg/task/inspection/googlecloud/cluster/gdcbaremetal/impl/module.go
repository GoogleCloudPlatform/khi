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

package gdcbaremetal_impl

import (
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/cluster/gdcbaremetal"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// Module declares the GDCV for Baremetal inspection type and the cluster name prefix task for it.
var Module = coreinspection.Module{
	Name: "googlecloud/cluster/gdcbaremetal",
	Scope: coreinspection.Scope{
		inspectioncore.InspectionTypeLabelKeyEnvironment:  "googlecloud",
		inspectioncore.InspectionTypeLabelKeyBasePlatform: "kubernetes",
		gcpcommon.InspectionTypeLabelKeyClusterType:       "gdc",
		gcpcommon.InspectionTypeLabelKeyClusterSubType:    "baremetal",
	},
	InspectionTypes: []coreinspection.InspectionType{gdcbaremetal.GDCVForBaremetalInspectionType},
	Tasks: []coretask.UntypedTask{
		gdcvForBaremetalClusterNamePrefixTask,
	},
}
