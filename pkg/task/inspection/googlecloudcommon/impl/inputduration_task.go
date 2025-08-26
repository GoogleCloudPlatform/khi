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

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// InputDurationTask defines a form task to input the duration for log queries.
var InputDurationTask = formtask.NewTextFormTaskBuilder(googlecloudcommon_contract.InputDurationTaskID, googlecloudcommon_contract.PriorityForQueryTimeGroup+4000, "Duration").
	WithDependencies([]taskid.UntypedTaskReference{
		inspectioncore_contract.InspectionTimeTaskID.Ref(),
		googlecloudcommon_contract.InputEndTimeTaskID.Ref(),
		inspectioncore_contract.TimeZoneShiftInputTaskID.Ref(),
	}).
	WithDescription("The duration of time range to gather logs. Supported time units are `h`,`m` or `s`. (Example: `3h30m`)").
	WithDefaultValueFunc(func(ctx context.Context, previousValues []string) (string, error) {
		if len(previousValues) > 0 {
			return previousValues[0], nil
		} else {
			return "1h", nil
		}
	}).
	WithHintFunc(func(ctx context.Context, value string, convertedValue any) (string, inspectionmetadata.ParameterHintType, error) {
		inspectionTime := coretask.GetTaskResult(ctx, inspectioncore_contract.InspectionTimeTaskID.Ref())
		endTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputEndTimeTaskID.Ref())
		timezoneShift := coretask.GetTaskResult(ctx, inspectioncore_contract.TimeZoneShiftInputTaskID.Ref())

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
		return hintString, inspectionmetadata.Info, nil
	}).
	WithSuggestionsConstant([]string{"1m", "10m", "1h", "3h", "12h", "24h"}).
	WithValidator(func(ctx context.Context, value string) (string, error) {
		d, err := time.ParseDuration(value)
		if err != nil {
			return err.Error(), nil
		}
		if d <= 0 {
			return "duration must be positive", nil
		}
		return "", nil
	}).
	WithConverter(func(ctx context.Context, value string) (time.Duration, error) {
		d, err := time.ParseDuration(value)
		if err != nil {
			return 0, err
		}
		return d, nil
	}).
	Build()

func toTimeDurationWithTimezone(startTime time.Time, endTime time.Time, timezone *time.Location, withTimezone bool) string {
	timeFormat := "2006-01-02T15:04:05"
	if withTimezone {
		timeFormat = time.RFC3339
	}
	startTimeStr := startTime.In(timezone).Format(timeFormat)
	endTimeStr := endTime.In(timezone).Format(timeFormat)
	return fmt.Sprintf("%s ~ %s", startTimeStr, endTimeStr)
}
