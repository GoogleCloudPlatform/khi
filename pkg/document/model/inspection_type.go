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

package model

import (
	"github.com/GoogleCloudPlatform/khi/pkg/common/filter"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

// InspectionTypeDocumentModel is a model type for generating document docs/en/reference/inspection-type.md
type InspectionTypeDocumentModel struct {
	// InspectionTypes are the list of InspectionType defind in KHI.
	InspectionTypes []InspectionTypeDocumentElement
}

// InspectionTypeDocumentElement is a model type for a InspectionType used in InspectionTypeDocumentModel.
type InspectionTypeDocumentElement struct {
	// ID is the unique name of the InspectionType.
	ID string
	// Name is the human readable name of the InspectionType.
	Name string
	// SupportedFeatures is the list of the feature tasks usable for this InspectionType.
	SupportedFeatures []InspectionTypeDocumentElementFeature
}

// InspectionTypeDocumentElementFeature is a model type for a feature task used for generatng the list of supported features of a InspectionType.
type InspectionTypeDocumentElementFeature struct {
	// ID is the unique name of the feature task.
	ID string
	// Name is the human readable name of the feature task.
	Name string
	// Description is the string exlains the feature task.
	Description string
}

// GetInspectionTypeDocumentModel returns the document model for inspection types from task server.
func GetInspectionTypeDocumentModel(taskServer *inspection.InspectionTaskServer) InspectionTypeDocumentModel {
	result := InspectionTypeDocumentModel{}
	inspectionTypes := taskServer.GetAllInspectionTypes()
	for _, inspectionType := range inspectionTypes {
		// Get the list of feature tasks supporting the inspection type.
		tasksInInspectionType := task.Subset(taskServer.RootTaskSet, filter.NewContainsElementFilter(inspection_task.LabelKeyInspectionTypes, inspectionType.Id, true))
		featureTasks := task.Subset(tasksInInspectionType, filter.NewEnabledFilter(inspection_task.LabelKeyInspectionFeatureFlag, false)).GetAll()

		features := []InspectionTypeDocumentElementFeature{}
		for _, task := range featureTasks {
			features = append(features, InspectionTypeDocumentElementFeature{
				ID:          task.ID().String(),
				Name:        typedmap.GetOrDefault(task.Labels(), inspection_task.LabelKeyFeatureTaskTitle, ""),
				Description: typedmap.GetOrDefault(task.Labels(), inspection_task.LabelKeyFeatureTaskDescription, ""),
			})
		}

		result.InspectionTypes = append(result.InspectionTypes, InspectionTypeDocumentElement{
			ID:                inspectionType.Id,
			Name:              inspectionType.Name,
			SupportedFeatures: features,
		})
	}
	return result
}
