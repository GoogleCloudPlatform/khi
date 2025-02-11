package model

import (
	"github.com/GoogleCloudPlatform/khi/pkg/inspection"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/task/label"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/taskfilter"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

type FormDocumentModel struct {
	Forms []FormDocumentElement
}

type FormDocumentElement struct {
	ID          string
	Label       string
	Description string

	UsedFeatures []FormUsedFeatureElement
}

type FormUsedFeatureElement struct {
	ID   string
	Name string
}

func GetFormDocumentModel(taskServer *inspection.InspectionTaskServer) (*FormDocumentModel, error) {
	result := FormDocumentModel{}
	forms := taskServer.RootTaskSet.FilteredSubset(label.TaskLabelKeyIsFormTask, taskfilter.HasTrue, false)
	for _, form := range forms.GetAll() {
		usedFeatures, err := getUsedFeatures(taskServer, form)
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

func getUsedFeatures(taskServer *inspection.InspectionTaskServer, formTask task.Definition) ([]task.Definition, error) {
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
