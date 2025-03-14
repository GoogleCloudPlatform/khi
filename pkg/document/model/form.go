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
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/task/label"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

// FormDocumentModel represents the model for generating document docs/en/reference/form.md.
type FormDocumentModel struct {
	// Forms is a list of form elements for the document.
	Forms []FormDocumentElement
}

// FormDocumentElement represents a single form element in the documentation.
type FormDocumentElement struct {
	// ID is the unique identifier of the form.
	ID string
	// Label is the display label for the form.
	Label string
	// Description provides a description of the form.
	Description string
	// UsedFeatures lists the features requesting this form parameter in their dependency.
	UsedFeatures []FormUsedFeatureElement
}

// FormUsedFeatureElement represents a feature used by a form.
type FormUsedFeatureElement struct {
	// ID is the unique identifier of the feature.
	ID string
	// Name is the human-readable name of the feature.
	Name string
}

// GetFormDocumentModel returns the document model for forms.
func GetFormDocumentModel(taskServer *inspection.InspectionTaskServer) (*FormDocumentModel, error) {
	result := FormDocumentModel{}
	forms := task.Subset(taskServer.RootTaskSet, filter.NewEnabledFilter(label.TaskLabelKeyIsFormTask, false))
	for _, form := range forms.GetAll() {
		usedFeatures, err := getFeaturesRequestingFormTask(taskServer, form)
		if err != nil {
			return nil, err
		}
		usedFeatureElements := []FormUsedFeatureElement{}
		for _, feature := range usedFeatures {
			usedFeatureElements = append(usedFeatureElements, FormUsedFeatureElement{
				ID:   feature.ID().String(),
				Name: typedmap.GetOrDefault(feature.Labels(), inspection_task.LabelKeyFeatureTaskTitle, ""),
			})
		}

		result.Forms = append(result.Forms, FormDocumentElement{
			ID:           form.ID().String(),
			Label:        typedmap.GetOrDefault(form.Labels(), label.TaskLabelKeyFormFieldLabel, ""),
			Description:  typedmap.GetOrDefault(form.Labels(), label.TaskLabelKeyFormFieldDescription, ""),
			UsedFeatures: usedFeatureElements,
		})
	}
	return &result, nil
}

// getFeaturesRequestingFormTask returns the list of feature tasks that depends on the given form task.
func getFeaturesRequestingFormTask(taskServer *inspection.InspectionTaskServer, formTask task.Definition) ([]task.Definition, error) {
	var result []task.Definition
	features := task.Subset(taskServer.RootTaskSet, filter.NewEnabledFilter(inspection_task.LabelKeyInspectionFeatureFlag, false))
	for _, feature := range features.GetAll() {
		hasDependency, err := task.HasDependency(taskServer.RootTaskSet, feature, formTask)
		if err != nil {
			return nil, err
		}
		if hasDependency {
			result = append(result, feature)
		}

	}
	return result, nil
}
