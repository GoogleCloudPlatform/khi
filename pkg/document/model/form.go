package model

import (
	"github.com/GoogleCloudPlatform/khi/pkg/inspection"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/task/label"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/taskfilter"
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
	forms := taskServer.RootTaskSet.FilteredSubset(label.TaskLabelKeyIsFormTask, taskfilter.HasTrue, false)
	for _, form := range forms.GetAll() {
		usedFeatures, err := getFeaturesRequestingFormTask(taskServer, form)
		if err != nil {
			return nil, err
		}
		usedFeatureElements := []FormUsedFeatureElement{}
		for _, feature := range usedFeatures {
			usedFeatureElements = append(usedFeatureElements, FormUsedFeatureElement{
				ID:   feature.ID().String(),
				Name: feature.Labels().GetOrDefault(inspection_task.LabelKeyFeatureTaskTitle, "").(string),
			})
		}

		result.Forms = append(result.Forms, FormDocumentElement{
			ID:           form.ID().String(),
			Label:        form.Labels().GetOrDefault(label.TaskLabelKeyFormFieldLabel, "").(string),
			Description:  form.Labels().GetOrDefault(label.TaskLabelKeyFormFieldDescription, "").(string),
			UsedFeatures: usedFeatureElements,
		})
	}
	return &result, nil
}

// getFeaturesRequestingFormTask returns the list of feature tasks that depends on the given form task.
func getFeaturesRequestingFormTask(taskServer *inspection.InspectionTaskServer, formTask task.Definition) ([]task.Definition, error) {
	var result []task.Definition
	features := taskServer.RootTaskSet.FilteredSubset(inspection_task.LabelKeyInspectionFeatureFlag, taskfilter.HasTrue, false)
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
