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

package googlecloudcommon_impl

import (
	"context"
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// InputEndTimeTask defines a form task to input the end time for log queries.
var InputEndTimeTask = formtask.NewTextFormTaskBuilder(googlecloudcommon_contract.InputEndTimeTaskID, googlecloudcommon_contract.PriorityForQueryTimeGroup+5000, "End time").
	WithDependencies([]taskid.UntypedTaskReference{
		inspectioncore_contract.TimeZoneShiftInputTaskID.Ref(),
	}).
	WithDescription(`The endtime of query. Please input it in the format of RFC3339
(example: 2006-01-02T15:04:05-07:00)`).
	WithSuggestionsFunc(func(ctx context.Context, value string, previousValues []string) ([]string, error) {
		return previousValues, nil
	}).
	WithDefaultValueFunc(func(ctx context.Context, previousValues []string) (string, error) {
		if len(previousValues) > 0 {
			return previousValues[0], nil
		}
		creationTime := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionCreationTime)
		timezoneShift := coretask.GetTaskResult(ctx, inspectioncore_contract.TimeZoneShiftInputTaskID.Ref())

		return creationTime.In(timezoneShift).Format(time.RFC3339), nil
	}).
	WithHintFunc(func(ctx context.Context, value string, convertedValue any) (string, inspectionmetadata.ParameterHintType, error) {
		creationTime := khictx.MustGetValue(ctx, inspectioncore_contract.InspectionCreationTime)

		specifiedTime := convertedValue.(time.Time)
		if creationTime.Sub(specifiedTime) < 0 {
			return fmt.Sprintf("Specified time `%s` is pointing the future. Please make sure if you specified the right value", value), inspectionmetadata.Warning, nil
		}
		return "", inspectionmetadata.Info, nil
	}).
	WithValidator(func(ctx context.Context, value string) (string, error) {
		_, err := common.ParseTime(value)
		if err != nil {
			return "invalid time format. Please specify in the format of `2006-01-02T15:04:05-07:00`(RFC3339)", nil
		}
		return "", nil
	}).
	WithConverter(func(ctx context.Context, value string) (time.Time, error) {
		return common.ParseTime(value)
	}).
	Build()
