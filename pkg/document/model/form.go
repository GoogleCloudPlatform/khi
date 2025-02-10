package model

import (
	"github.com/GoogleCloudPlatform/khi/pkg/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/task/label"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/taskfilter"
)

type FormDocumentModel struct {
	Forms []FormDocumentElement
}

type FormDocumentElement struct {
	ID          string
	Label       string
	Description string
}

func GetFormDocumentModel(taskServer *inspection.InspectionTaskServer) (*FormDocumentModel, error) {
	result := FormDocumentModel{}
	forms := taskServer.RootTaskSet.FilteredSubset(label.TaskLabelKeyIsFormTask, taskfilter.HasTrue, false)
	for _, form := range forms.GetAll() {
		result.Forms = append(result.Forms, FormDocumentElement{
			ID:          form.ID().String(),
			Label:       form.Labels().GetOrDefault(label.TaskLabelKeyFormFieldLabel, "").(string),
			Description: form.Labels().GetOrDefault(label.TaskLabelKeyFormFieldDescription, "").(string),
		})
	}
	return &result, nil
}
