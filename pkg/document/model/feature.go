package model

import (
	"slices"
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

	IndirectQueryDependency  []FeatureIndirectDependentQueryElement
	TargetQueryDependency    FeatureDependentTargetQueryElement
	Forms                    []FeatureDependentFormElement
	OutputTimelines          []FeatureOutputTimelineElement
	AvailableInspectionTypes []FeatureAvailableInspectionType
}

type FeatureIndirectDependentQueryElement struct {
	ID               string
	LogTypeLabel     string
	LogTypeColorCode string
}

type FeatureDependentTargetQueryElement struct {
	ID               string
	LogTypeLabel     string
	LogTypeColorCode string
	SampleQuery      string
}

type FeatureDependentFormElement struct {
	ID          string
	Label       string
	Description string
}

type FeatureOutputTimelineElement struct {
	RelationshipID        string
	RelationshipColorCode string
	LongName              string
	Label                 string
	Description           string
}

type FeatureAvailableInspectionType struct {
	ID   string
	Name string
}

func GetFeatureDocumentModel(taskServer *inspection.InspectionTaskServer) (*FeatureDocumentModel, error) {
	result := FeatureDocumentModel{}
	features := taskServer.RootTaskSet.FilteredSubset(inspection_task.LabelKeyInspectionFeatureFlag, taskfilter.HasTrue, false)
	for _, feature := range features.GetAll() {
		indirectQueryDependencyElement := []FeatureIndirectDependentQueryElement{}
		targetQueryDependencyElement := FeatureDependentTargetQueryElement{}
		targetLogTypeKey := feature.Labels().GetOrDefault(inspection_task.LabelKeyFeatureTaskTargetLogType, enum.LogTypeUnknown).(enum.LogType)

		// Get query related tasks in the dependency of this feature.
		queryTasksInDependency, err := getDependentQueryTasks(taskServer, feature)
		if err != nil {
			return nil, err
		}
		for _, queryTask := range queryTasksInDependency {
			logTypeKey := enum.LogType(queryTask.Labels().GetOrDefault(label.TaskLabelKeyQueryTaskTargetLogType, enum.LogTypeUnknown).(enum.LogType))
			if targetLogTypeKey != logTypeKey {
				logType := enum.LogTypes[logTypeKey]
				indirectQueryDependencyElement = append(indirectQueryDependencyElement, FeatureIndirectDependentQueryElement{
					ID:               queryTask.ID().String(),
					LogTypeLabel:     logType.Label,
					LogTypeColorCode: strings.TrimLeft(logType.LabelBackgroundColor, "#"),
				})
			} else {
				targetQueryDependencyElement = FeatureDependentTargetQueryElement{
					ID:               queryTask.ID().String(),
					LogTypeLabel:     enum.LogTypes[targetLogTypeKey].Label,
					LogTypeColorCode: strings.TrimLeft(enum.LogTypes[targetLogTypeKey].LabelBackgroundColor, "#"),
					SampleQuery:      queryTask.Labels().GetOrDefault(label.TaskLabelKeyQueryTaskSampleQuery, "").(string),
				}
			}
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

		outputTimelines := []FeatureOutputTimelineElement{}
		for i := 0; i < enum.EnumParentRelationshipLength; i++ {
			relationshipKey := enum.ParentRelationship(i)
			relationship := enum.ParentRelationships[relationshipKey]

			isRelated := false
			for _, event := range relationship.GeneratableEvents {
				if event.SourceLogType == targetLogTypeKey {
					isRelated = true
					break
				}
			}
			for _, revision := range relationship.GeneratableRevisions {
				if revision.SourceLogType == targetLogTypeKey {
					isRelated = true
					break
				}
			}
			for _, alias := range relationship.GeneratableAliasTimelineInfo {
				if alias.SourceLogType == targetLogTypeKey {
					isRelated = true
					break
				}
			}
			if isRelated {
				outputTimelines = append(outputTimelines, FeatureOutputTimelineElement{
					RelationshipID:        relationship.EnumKeyName,
					RelationshipColorCode: strings.TrimLeft(relationship.LabelBackgroundColor, "#"),
					LongName:              relationship.LongName,
					Label:                 relationship.Label,
					Description:           relationship.Description,
				})
			}
		}

		result.Features = append(result.Features, FeatureDocumentElement{
			ID:                       feature.Labels().GetOrDefault(inspection_task.LabelKeyFeatureDocumentAnchorID, "").(string),
			Name:                     feature.Labels().GetOrDefault(inspection_task.LabelKeyFeatureTaskTitle, "").(string),
			Description:              feature.Labels().GetOrDefault(inspection_task.LabelKeyFeatureTaskDescription, "").(string),
			IndirectQueryDependency:  indirectQueryDependencyElement,
			TargetQueryDependency:    targetQueryDependencyElement,
			Forms:                    formElements,
			OutputTimelines:          outputTimelines,
			AvailableInspectionTypes: getAvailableInspectionTypes(taskServer, feature),
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

// getAvailableInspectionTypes returns the list of information about inspection type that supports this feature.
func getAvailableInspectionTypes(taskServer *inspection.InspectionTaskServer, featureTask task.Definition) []FeatureAvailableInspectionType {
	result := []FeatureAvailableInspectionType{}
	inspectionTypes := taskServer.GetAllInspectionTypes()
	for _, inspectionType := range inspectionTypes {
		labelsAny, found := featureTask.Labels().Get(inspection_task.LabelKeyInspectionTypes)
		labels := []string{inspectionType.Id}
		if found {
			labels = labelsAny.([]string)
		}
		if slices.Contains(labels, inspectionType.Id) {
			result = append(result, FeatureAvailableInspectionType{
				ID:   inspectionType.Id,
				Name: inspectionType.Name,
			})
		}
	}
	return result
}
