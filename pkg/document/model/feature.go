package model

import (
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/task/label"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/taskfilter"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

type FeatureDocumentModel struct {
	Features []FeatureDocumentElement
}

type FeatureDocumentElement struct {
	ID          string
	Name        string
	Description string

	Queries []FeatureDependentQueryElement
}

type FeatureDependentQueryElement struct {
	ID               string
	LogType          enum.LogType
	LogTypeLabel     string
	LogTypeColorCode string
	SampleQuery      string
}

func GetFeatureDocumentModel(taskServer *inspection.InspectionTaskServer) (*FeatureDocumentModel, error) {
	result := FeatureDocumentModel{}
	features := taskServer.RootTaskSet.FilteredSubset(inspection_task.LabelKeyInspectionFeatureFlag, taskfilter.HasTrue, false)
	for _, feature := range features.GetAll() {
		queryElements := []FeatureDependentQueryElement{}

		// Get query related tasks required by this feature.
		resolveSource, err := task.NewSet([]task.Definition{feature})
		if err != nil {
			return nil, err
		}
		resolved, err := resolveSource.ResolveTask(taskServer.RootTaskSet)
		if err != nil {
			return nil, err
		}
		queryTasks := resolved.FilteredSubset(label.TaskLabelKeyIsQueryTask, taskfilter.HasTrue, false).GetAll()
		for _, queryTask := range queryTasks {
			logTypeKey := enum.LogType(queryTask.Labels().GetOrDefault(label.TaskLabelKeyQueryTaskTargetLogType, enum.LogTypeUnknown).(enum.LogType))
			logType := enum.LogTypes[logTypeKey]
			queryElements = append(queryElements, FeatureDependentQueryElement{
				ID:               queryTask.ID().String(),
				LogType:          logTypeKey,
				LogTypeLabel:     logType.Label,
				LogTypeColorCode: strings.TrimLeft(logType.LabelBackgroundColor, "#"),
				SampleQuery:      queryTask.Labels().GetOrDefault(label.TaskLabelKeyQueryTaskSampleQuery, "").(string),
			})
		}

		result.Features = append(result.Features, FeatureDocumentElement{
			ID:          feature.ID().String(),
			Name:        feature.Labels().GetOrDefault(inspection_task.LabelKeyFeatureTaskTitle, "").(string),
			Description: feature.Labels().GetOrDefault(inspection_task.LabelKeyFeatureTaskDescription, "").(string),
			Queries:     queryElements,
		})

	}
	return &result, nil
}
