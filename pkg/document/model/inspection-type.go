package model

import (
	"github.com/GoogleCloudPlatform/khi/pkg/inspection"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/taskfilter"
)

// InspectionTypeDocumentModel is a document model type for generating docs/en/inspection-type.md
type InspectionTypeDocumentModel struct {
	InspectionTypes []InspectionTypeDocumentElement
}

type InspectionTypeDocumentElement struct {
	ID   string
	Name string

	SupportedFeatures []InspectionTypeDocumentElementFeature
}

type InspectionTypeDocumentElementFeature struct {
	ID          string
	Name        string
	Description string
}

// GetInspectionTypeDocumentModel returns the document model from task server.
func GetInspectionTypeDocumentModel(taskServer *inspection.InspectionTaskServer) InspectionTypeDocumentModel {
	result := InspectionTypeDocumentModel{}
	inspectionTypes := taskServer.GetAllInspectionTypes()
	for _, inspectionType := range inspectionTypes {
		tasks := taskServer.RootTaskSet.
			FilteredSubset(inspection_task.LabelKeyInspectionTypes, taskfilter.ContainsElement(inspectionType.Id), true).
			FilteredSubset(inspection_task.LabelKeyInspectionFeatureFlag, taskfilter.HasTrue, false).
			GetAll()
		features := []InspectionTypeDocumentElementFeature{}
		for _, task := range tasks {
			features = append(features, InspectionTypeDocumentElementFeature{
				ID:          task.ID().String(),
				Name:        task.Labels().GetOrDefault(inspection_task.LabelKeyFeatureTaskTitle, "").(string),
				Description: task.Labels().GetOrDefault(inspection_task.LabelKeyFeatureTaskDescription, "").(string),
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
