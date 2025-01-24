// Copyright 2024 Google LLC
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

package task

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/form"
	form_metadata "github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/form"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/header"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
	common_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query/queryutil"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

const FormBasePriority = 100000
const PriorityForQueryTimeGroup = FormBasePriority + 50000
const PriorityForResourceIdentifierGroup = FormBasePriority + 40000
const PriorityForK8sResourceFilterGroup = FormBasePriority + 30000

const InputProjectIdTaskID = GCPPrefix + "input/project-id"

var projectIdValidator = regexp.MustCompile(`^\s*[0-9a-z\.:\-]+\s*$`)

var InputProjectIdTask = form.NewInputFormDefinitionBuilder(InputProjectIdTaskID, PriorityForResourceIdentifierGroup+5000, "Project ID").
	WithDescription("A project ID containing the cluster to inspect").
	WithDependencies([]string{}).
	WithValidator(func(ctx context.Context, value string, variables *task.VariableSet) (string, error) {
		if !projectIdValidator.Match([]byte(value)) {
			return "Project ID must match `^*[0-9a-z\\.:\\-]+$`", nil
		}
		return "", nil
	}).
	WithAllowEditFunc(func(ctx context.Context, variables *task.VariableSet) (bool, error) {
		if parameters.Auth.FixedProjectID == nil {
			return true, nil
		}
		return *parameters.Auth.FixedProjectID == "", nil
	}).
	WithDefaultValueFunc(func(ctx context.Context, variables *task.VariableSet, previousValues []string) (string, error) {
		if parameters.Auth.FixedProjectID != nil && *parameters.Auth.FixedProjectID != "" {
			return *parameters.Auth.FixedProjectID, nil
		}
		if len(previousValues) > 0 {
			return previousValues[0], nil
		}
		return "", nil
	}).
	WithConverter(func(ctx context.Context, value string, variables *task.VariableSet) (any, error) {
		return strings.TrimSpace(value), nil
	}).
	Build()

func GetInputProjectIdFromTaskVariable(tv *task.VariableSet) (string, error) {
	return task.GetTypedVariableFromTaskVariable[string](tv, InputProjectIdTaskID, "<INVALID>")
}

const InputClusterNameTaskID = GCPPrefix + "input/cluster-name"

var clusterNameValidator = regexp.MustCompile(`^\s*[0-9a-z\-]+\s*$`)

var InputClusterNameTask = form.NewInputFormDefinitionBuilder(InputClusterNameTaskID, PriorityForResourceIdentifierGroup+4000, "Cluster name").
	WithDependencies([]string{AutocompleteClusterNamesTaskID, ClusterNamePrefixTaskID}).
	WithDefaultValueFunc(func(ctx context.Context, variables *task.VariableSet, previousValues []string) (string, error) {
		clusters, err := GetAutocompleteClusterNamesFromTaskVariable(variables)
		if err != nil {
			return "", err
		}
		// If the previous value is included in the list of cluster names, the name is used as the default value.
		if len(previousValues) > 0 && slices.Index(clusters.ClusterNames, previousValues[0]) > -1 {
			return previousValues[0], nil
		}
		if len(clusters.ClusterNames) == 0 {
			return "", nil
		}
		return clusters.ClusterNames[0], nil
	}).
	WithSuggestionsFunc(func(ctx context.Context, value string, variables *task.VariableSet, previousValues []string) ([]string, error) {
		clusters, err := GetAutocompleteClusterNamesFromTaskVariable(variables)
		if err != nil {
			return []string{}, err
		}
		return common.SortForAutocomplete(value, clusters.ClusterNames), nil
	}).
	WithHintFunc(func(ctx context.Context, value string, convertedValue any, variables *task.VariableSet) (string, form_metadata.FormFieldHintType, error) {
		clusters, err := GetAutocompleteClusterNamesFromTaskVariable(variables)
		if err != nil {
			return "", form_metadata.HintTypeInfo, err
		}
		// on failure of getting the list of clusters
		if clusters.Error != "" {
			return fmt.Sprintf("Failed to obtain the cluster list due to the error '%s'.\n The suggestion list won't popup", clusters.Error), form_metadata.HintTypeWarning, nil
		}
		prefix, err := GetClusterNamePrefixFromTaskVariable(variables)
		if err != nil {
			return "", form_metadata.HintTypeInfo, err
		}
		convertedWithoutPrefix := strings.TrimPrefix(convertedValue.(string), prefix)
		for _, suggestedCluster := range clusters.ClusterNames {
			if suggestedCluster == convertedWithoutPrefix {
				return "", form_metadata.HintTypeInfo, nil
			}
		}
		return fmt.Sprintf("Cluster `%s` was not found in the specified project at this time. It works for the clusters existed in the past but make sure the cluster name is right if you believe the cluster should be there.", value), form_metadata.HintTypeWarning, nil
	}).
	WithValidator(func(ctx context.Context, value string, variables *task.VariableSet) (string, error) {
		if !clusterNameValidator.Match([]byte(value)) {
			return "Cluster name must match `^[0-9a-z:\\-]+$`", nil
		}
		return "", nil
	}).
	WithConverter(func(ctx context.Context, value string, variables *task.VariableSet) (any, error) {
		// Needs to add the cluster name prefix defined by the inspection type.
		prefix, err := GetClusterNamePrefixFromTaskVariable(variables)
		if err != nil {
			return nil, err
		}
		return prefix + strings.TrimSpace(value), nil
	}).
	Build()

func GetInputClusterNameFromTaskVariable(tv *task.VariableSet) (string, error) {
	return task.GetTypedVariableFromTaskVariable[string](tv, InputClusterNameTaskID, "<INVALID>")
}

const InputDurationTaskID = GCPPrefix + "input/duration"

var InputDurationTask = form.NewInputFormDefinitionBuilder(InputDurationTaskID, PriorityForQueryTimeGroup+4000, "Duration").
	WithDependencies([]string{
		common_task.InspectionTimeTaskID,
		InputEndTimeTaskID,
		TimeZoneShiftInputTaskID,
	}).
	WithDefaultValueFunc(func(ctx context.Context, variables *task.VariableSet, previousValues []string) (string, error) {
		if len(previousValues) > 0 {
			return previousValues[0], nil
		} else {
			return "1h", nil
		}
	}).
	WithHintFunc(func(ctx context.Context, value string, convertedValue any, variables *task.VariableSet) (string, form_metadata.FormFieldHintType, error) {
		inspectionTime, err := common_task.GetInspectionTimeFromTaskVariable(variables)
		if err != nil {
			return "", form_metadata.HintTypeInfo, err
		}
		endTime, err := GetInputEndTimeFromTaskVariable(variables)
		if err != nil {
			return "", form_metadata.HintTypeInfo, err
		}
		timezoneShift, err := GetTimezoneShiftInput(variables)
		if err != nil {
			return "", form_metadata.HintTypeInfo, err
		}
		duration := convertedValue.(time.Duration)
		startTime := endTime.Add(-duration)
		startToNow := inspectionTime.Sub(startTime)
		hintString := ""
		if startToNow > time.Hour*24*30 {
			hintString += "Specified time range starts from over than 30 days ago, maybe some logs are missing and the generated result could be incomplete.\n"
		}
		if duration > time.Hour*3 {
			hintString += "This duration can be too long for big clusters and lead OOM. Please retry with shorter duration when your machine crashed.\n"
		}
		hintString += fmt.Sprintf("Query range:\n%s\n", toTimeDurationWithTimezone(startTime, endTime, timezoneShift, true))
		hintString += fmt.Sprintf("(UTC: %s)\n", toTimeDurationWithTimezone(startTime, endTime, time.UTC, false))
		hintString += fmt.Sprintf("(PDT: %s)", toTimeDurationWithTimezone(startTime, endTime, time.FixedZone("PDT", -7*3600), false))
		return hintString, form_metadata.HintTypeInfo, nil
	}).
	WithSuggestionsConstant([]string{"1m", "10m", "1h", "3h", "12h", "24h"}).
	WithValidator(func(ctx context.Context, value string, variables *task.VariableSet) (string, error) {
		d, err := time.ParseDuration(value)
		if err != nil {
			return err.Error(), nil
		}
		if d <= 0 {
			return "duration must be positive", nil
		}
		return "", nil
	}).
	WithConverter(func(ctx context.Context, value string, variables *task.VariableSet) (any, error) {
		d, err := time.ParseDuration(value)
		if err != nil {
			return nil, err
		}
		return d, nil
	}).
	Build()

func GetInputDurationFromTaskVariable(tv *task.VariableSet) (time.Duration, error) {
	return task.GetTypedVariableFromTaskVariable[time.Duration](tv, InputDurationTaskID, 0)
}

const InputEndTimeTaskID = GCPPrefix + "input/end-time"

var InputEndTimeTask = form.NewInputFormDefinitionBuilder(InputEndTimeTaskID, PriorityForQueryTimeGroup+5000, "End time").
	WithDependencies([]string{
		common_task.InspectionTimeTaskID,
		TimeZoneShiftInputTaskID,
	}).
	WithDescription(`The endtime of query. Please input it in the format of RFC3339
(example: 2006-01-02T15:04:05-07:00)`).
	WithSuggestionsFunc(func(ctx context.Context, value string, variables *task.VariableSet, previousValues []string) ([]string, error) {
		return previousValues, nil
	}).
	WithDefaultValueFunc(func(ctx context.Context, variables *task.VariableSet, previousValues []string) (string, error) {
		if len(previousValues) > 0 {
			return previousValues[0], nil
		}
		inspectionTime, err := common_task.GetInspectionTimeFromTaskVariable(variables)
		if err != nil {
			return "", err
		}
		timezoneShift, err := GetTimezoneShiftInput(variables)
		if err != nil {
			return "", err
		}
		return inspectionTime.In(timezoneShift).Format(time.RFC3339), nil
	}).
	WithHintFunc(func(ctx context.Context, value string, convertedValue any, variables *task.VariableSet) (string, form_metadata.FormFieldHintType, error) {
		inspectionTime, err := common_task.GetInspectionTimeFromTaskVariable(variables)
		if err != nil {
			return "", form_metadata.HintTypeInfo, err
		}
		specifiedTime := convertedValue.(time.Time)
		if inspectionTime.Sub(specifiedTime) < 0 {
			return fmt.Sprintf("Specified time `%s` is pointing the future. Please make sure if you specified the right value", value), form_metadata.HintTypeWarning, nil
		}
		return "", form_metadata.HintTypeInfo, nil
	}).
	WithValidator(func(ctx context.Context, value string, variables *task.VariableSet) (string, error) {
		_, err := common.ParseTime(value)
		if err != nil {
			return "invalid time format. Please specify in the format of `2006-01-02T15:04:05-07:00`(RFC3339)", nil
		}
		return "", nil
	}).
	WithConverter(func(ctx context.Context, value string, variables *task.VariableSet) (any, error) {
		return common.ParseTime(value)
	}).
	Build()

func GetInputEndTimeFromTaskVariable(tv *task.VariableSet) (time.Time, error) {
	return task.GetTypedVariableFromTaskVariable[time.Time](tv, InputEndTimeTaskID, time.Time{})
}

const InputStartTimeTaskID = GCPPrefix + "input/start-time"

var InputStartTimeTask = common_task.NewInspectionProcessor(InputStartTimeTaskID, []string{
	InputEndTimeTaskID,
	InputDurationTaskID,
}, func(ctx context.Context, taskMode int, v *task.VariableSet, progress *progress.TaskProgress) (any, error) {
	endTime, err := GetInputEndTimeFromTaskVariable(v)
	if err != nil {
		return nil, err
	}
	duration, err := GetInputDurationFromTaskVariable(v)
	if err != nil {
		return nil, err
	}
	startTime := endTime.Add(-duration)
	// Add starttime and endtime on the header metadata
	metadataSet, err := common_task.GetMetadataSetFromVariable(v)
	if err != nil {
		return nil, err
	}
	header := metadataSet.LoadOrStore(header.HeaderMetadataKey, &header.HeaderMetadataFactory{}).(*header.Header)
	header.StartTimeUnixSeconds = startTime.Unix()
	header.EndTimeUnixSeconds = endTime.Unix()
	return startTime, nil
})

func GetInputStartTimeFromTaskVariable(tv *task.VariableSet) (time.Time, error) {
	return task.GetTypedVariableFromTaskVariable[time.Time](tv, InputStartTimeTaskID, time.Time{})
}

const InputKindFilterTaskID = GCPPrefix + "input/kinds"

var inputKindNameAliasMap queryutil.SetFilterAliasToItemsMap = map[string][]string{
	"default": strings.Split("pods replicasets daemonsets nodes deployments namespaces statefulsets services servicenetworkendpointgroups ingresses poddisruptionbudgets jobs cronjobs endpointslices persistentvolumes persistentvolumeclaims storageclasses horizontalpodautoscalers verticalpodautoscalers multidimpodautoscalers", " "),
}
var InputKindFilterTask = form.NewInputFormDefinitionBuilder(InputKindFilterTaskID, PriorityForK8sResourceFilterGroup+5000, "Kind").
	WithDefaultValueConstant("@default", true).
	WithValidator(func(ctx context.Context, value string, variables *task.VariableSet) (string, error) {
		if value == "" {
			return "kind filter can't be empty", nil
		}
		result, err := queryutil.ParseSetFilter(value, inputKindNameAliasMap, true, true, true)
		if err != nil {
			return "", err
		}
		return result.ValidationError, nil
	}).
	WithConverter(func(ctx context.Context, value string, variables *task.VariableSet) (any, error) {
		result, err := queryutil.ParseSetFilter(value, inputKindNameAliasMap, true, true, true)
		if err != nil {
			return "", err
		}
		return result, nil
	}).
	Build()

func GetInputKindNameFromTaskVariable(tv *task.VariableSet) (*queryutil.SetFilterParseResult, error) {
	return task.GetTypedVariableFromTaskVariable[*queryutil.SetFilterParseResult](tv, InputKindFilterTaskID, nil)
}

const InputNamespaceFilterTaskID = GCPPrefix + "input/namespaces"

var inputNamespacesAliasMap queryutil.SetFilterAliasToItemsMap = map[string][]string{
	"all_cluster_scoped": {"#cluster-scoped"},
	"all_namespaced":     {"#namespaced"},
}
var InputNamespaceFilterTask = form.NewInputFormDefinitionBuilder(InputNamespaceFilterTaskID, PriorityForK8sResourceFilterGroup+4000, "Namespaces").
	WithDefaultValueConstant("@all_cluster_scoped @all_namespaced", true).
	WithValidator(func(ctx context.Context, value string, variables *task.VariableSet) (string, error) {
		if value == "" {
			return "namespace filter can't be empty", nil
		}
		result, err := queryutil.ParseSetFilter(value, inputNamespacesAliasMap, false, false, true)
		if err != nil {
			return "", err
		}
		return result.ValidationError, nil
	}).
	WithConverter(func(ctx context.Context, value string, variables *task.VariableSet) (any, error) {
		result, err := queryutil.ParseSetFilter(value, inputNamespacesAliasMap, false, false, true)
		if err != nil {
			return "", err
		}
		return result, nil
	}).
	Build()

func GetInputNamespaceFilterFromTaskVariable(tv *task.VariableSet) (*queryutil.SetFilterParseResult, error) {
	return task.GetTypedVariableFromTaskVariable[*queryutil.SetFilterParseResult](tv, InputNamespaceFilterTaskID, nil)
}

const InputNodeNameFilterTaskID = GCPPrefix + "input/node-name-filter"

var nodeNameSubstringValidator = regexp.MustCompile("^[-a-z0-9]*$")

// getNodeNameSubstringsFromRawInput splits input by spaces and returns result in array.
// This removes surround spaces and removes empty string.
func getNodeNameSubstringsFromRawInput(value string) []string {
	result := []string{}
	nodeNameSubstrings := strings.Split(value, " ")
	for _, v := range nodeNameSubstrings {
		nodeNameSubstring := strings.TrimSpace(v)
		if nodeNameSubstring != "" {
			result = append(result, nodeNameSubstring)
		}
	}
	return result
}

// InputNodeNameFilterTask is a task to collect list of substrings of node names. This input value is used in querying k8s_node or serialport logs.
var InputNodeNameFilterTask = form.NewInputFormDefinitionBuilder(InputNodeNameFilterTaskID, PriorityForK8sResourceFilterGroup+3000, "Node names").
	WithDefaultValueConstant("", true).
	WithDescription("A space-separated list of node name substrings used to collect node-related logs. If left blank, KHI gathers logs from all nodes in the cluster.").
	WithValidator(func(ctx context.Context, value string, variables *task.VariableSet) (string, error) {
		nodeNameSubstrings := getNodeNameSubstringsFromRawInput(value)
		for _, name := range nodeNameSubstrings {
			if !nodeNameSubstringValidator.Match([]byte(name)) {
				return fmt.Sprintf("substring `%s` is not valid as a substring of node name", name), nil
			}
		}
		return "", nil
	}).WithConverter(func(ctx context.Context, value string, variables *task.VariableSet) (any, error) {
	return getNodeNameSubstringsFromRawInput(value), nil
}).Build()

func GetNodeNameFilterFromTaskVaraible(tv *task.VariableSet) ([]string, error) {
	return task.GetTypedVariableFromTaskVariable[[]string](tv, InputNodeNameFilterTaskID, nil)
}

const InputLocationsTaskID = GCPPrefix + "input/location"

var InputLocationsTask = form.NewInputFormDefinitionBuilder(InputLocationsTaskID, PriorityForResourceIdentifierGroup+4500, "Location").WithDescription(
	"A location(regions) containing the environments to inspect",
).Build()

func GetInputLocationsFromTaskVariable(tv *task.VariableSet) (string, error) {
	return task.GetTypedVariableFromTaskVariable[string](tv, InputLocationsTaskID, "")
}

func toTimeDurationWithTimezone(startTime time.Time, endTime time.Time, timezone *time.Location, withTimezone bool) string {
	timeFormat := "2006-01-02T15:04:05"
	if withTimezone {
		timeFormat = time.RFC3339
	}
	startTimeStr := startTime.In(timezone).Format(timeFormat)
	endTimeStr := endTime.In(timezone).Format(timeFormat)
	return fmt.Sprintf("%s ~ %s", startTimeStr, endTimeStr)
}
