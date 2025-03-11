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

package oss

import (
	"math"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/source/oss/constant"
)

var OSSKubernetesLogFilesInspectionType = inspection.InspectionType{
	Id:          constant.OSSInspectionTypeID,
	Name:        "OSS Kubernetes Log Files",
	Description: "Parsers for logs uploaded as the raw log files.",
	Icon:        "assets/icons/anthos.png", // TODO: change! place holder
	Priority:    math.MaxInt - 1000,
}
