package model

import (
	"github.com/GoogleCloudPlatform/khi/pkg/inspection"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/taskfilter"
)

// InspectionTypeDocumentModel is a model type for generating document docs/en/reference/inspection-type.md
type InspectionTypeDocumentModel struct {
	// InspectionTypes are the list of InspectionType defind in KHI.
	InspectionTypes []InspectionTypeDocumentElement
}

// InspectionTypeDocumentElement is a model for a InspectionType used in InspectionTypeDocumentModel.
type InspectionTypeDocumentElement struct {
	// ID is the unique name of the InspectionType.
	ID string
	// Name is the human readable name of the InspectionType.
	Name string
	// SupportedFeatures is the list of the feature tasks usable for this InspectionType.
	SupportedFeatures []InspectionTypeDocumentElementFeature
}

// InspectionTypeDocumentElementFeature is a model for a feature task used for generatng the list of supported features of a InspectionType.
type InspectionTypeDocumentElementFeature struct {
	// ID is the unique name of the feature task.
	ID string
	// Name is the human readable name of the feature task.
	Name string
	// Description is the string exlains the feature task.
	Description string
}

// GetInspectionTypeDocumentModel returns the document model from task server.
func GetInspectionTypeDocumentModel(taskServer *inspection.InspectionTaskServer) InspectionTypeDocumentModel {
	result := InspectionTypeDocumentModel{}
	inspectionTypes := taskServer.GetAllInspectionTypes()
	for _, inspectionType := range inspectionTypes {
		// Get the list of feature tasks supporting the inspection type.
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
