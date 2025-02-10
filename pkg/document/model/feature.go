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
	Forms   []FeatureDependentFormElement
}

type FeatureDependentQueryElement struct {
	ID               string
	LogType          enum.LogType
	LogTypeLabel     string
	LogTypeColorCode string
	SampleQuery      string
}

type FeatureDependentFormElement struct {
	ID          string
	Label       string
	Description string
}

func GetFeatureDocumentModel(taskServer *inspection.InspectionTaskServer) (*FeatureDocumentModel, error) {
	result := FeatureDocumentModel{}
	features := taskServer.RootTaskSet.FilteredSubset(inspection_task.LabelKeyInspectionFeatureFlag, taskfilter.HasTrue, false)
	for _, feature := range features.GetAll() {
		queryElements := []FeatureDependentQueryElement{}

		// Get query related tasks required by this feature.
		queryTasks, err := getDependentQueryTasks(taskServer, feature)
		if err != nil {
			return nil, err
		}
		for _, queryTask := range queryTasks {
			logTypeKey := enum.LogType(queryTask.Labels().GetOrDefault(label.TaskLabelKeyQueryTaskTargetLogType, enum.LogTypeUnknown).(enum.LogType))
			logType := enum.LogTypes[logTypeKey]
			queryElements = append(queryElements, FeatureDependentQueryElement{
				ID:               queryTask.ID().String(),
				LogType:          logTypeKey,
				LogTypeLabel:     logType.Label,
				LogTypeColorCode: strings.TrimLeft(logType.LabelBackgroundColor, "#"),
				SampleQuery:      strings.TrimRight(queryTask.Labels().GetOrDefault(label.TaskLabelKeyQueryTaskSampleQuery, "").(string), "\n"),
			})
		}

		formElements := []FeatureDependentFormElement{}
		formTasks, err := getDependentFormTasks(taskServer, feature)
		if err != nil {
			return nil, err
		}
		for _, formTask := range formTasks {
			formElements = append(formElements, FeatureDependentFormElement{
				ID:          formTask.ID().String(),
				Label:       formTask.Labels().GetOrDefault(label.TaskLabelKeyFormFieldLabel, "").(string),
				Description: formTask.Labels().GetOrDefault(label.TaskLabelKeyFormFieldDescription, "").(string),
			})
		}

		result.Features = append(result.Features, FeatureDocumentElement{
			ID:          feature.Labels().GetOrDefault(inspection_task.LabelKeyFeatureDocumentAnchorID, "").(string),
			Name:        feature.Labels().GetOrDefault(inspection_task.LabelKeyFeatureTaskTitle, "").(string),
			Description: feature.Labels().GetOrDefault(inspection_task.LabelKeyFeatureTaskDescription, "").(string),
			Queries:     queryElements,
			Forms:       formElements,
		})

	}
	return &result, nil
}

func getDependentQueryTasks(taskServer *inspection.InspectionTaskServer, featureTask task.Definition) ([]task.Definition, error) {
	resolveSource, err := task.NewSet([]task.Definition{featureTask})
	if err != nil {
		return nil, err
	}
	resolved, err := resolveSource.ResolveTask(taskServer.RootTaskSet)
	if err != nil {
		return nil, err
	}
	return resolved.FilteredSubset(label.TaskLabelKeyIsQueryTask, taskfilter.HasTrue, false).GetAll(), nil
}

func getDependentFormTasks(taskServer *inspection.InspectionTaskServer, featureTask task.Definition) ([]task.Definition, error) {
	resolveSource, err := task.NewSet([]task.Definition{featureTask})
	if err != nil {
		return nil, err
	}
	resolved, err := resolveSource.ResolveTask(taskServer.RootTaskSet)
	if err != nil {
		return nil, err
	}
	return resolved.FilteredSubset(label.TaskLabelKeyIsFormTask, taskfilter.HasTrue, false).GetAll(), nil
}
