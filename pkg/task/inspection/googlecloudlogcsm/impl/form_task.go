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

package googlecloudlogcsm_impl

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khierrors"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogcsm_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcsm/contract"
)

const priorityForCSMGroup = googlecloudcommon_contract.FormBasePriority + 10000

var inputCSMAliasMap gcpqueryutil.SetFilterAliasToItemsMap = map[string][]string{}

var InputCSMResponseFlagsTask = formtask.NewSetFormTaskBuilder(googlecloudlogcsm_contract.InputCSMResponseFlagsTaskID, priorityForCSMGroup+1000, "Envoy response flags").
	WithDefaultValueConstant([]string{"@any", "-OK"}, true).
	WithAllowAddAll(false).
	WithAllowRemoveAll(false).
	WithAllowCustomValue(true).
	WithDescription("Response flags used for filtering CSM access logs. Note '-' in response flags is corresponded to 'OK' in this form.").
	WithOptionsFunc(func(ctx context.Context, previousValues []string) ([]inspectionmetadata.SetParameterFormFieldOptionItem, error) {
		result := []inspectionmetadata.SetParameterFormFieldOptionItem{
			{ID: "@any", Description: "[Alias] Matches any response flag"},
		}
		ids := make([]string, 0, len(googlecloudlogcsm_contract.HumanReadableErrorMessage))
		for flag := range googlecloudlogcsm_contract.HumanReadableErrorMessage {
			ids = append(ids, string(flag))
		}
		sort.Strings(ids)
		for _, id := range ids {
			message := googlecloudlogcsm_contract.HumanReadableErrorMessage[googlecloudlogcsm_contract.ResponseFlag(id)]
			if id == "-" {
				id = "OK"
				message = "It's '-' in the response flag field because '-' means substracting operator in this form."
			}
			result = append(result, inspectionmetadata.SetParameterFormFieldOptionItem{ID: id, Description: message})
		}
		return result, nil
	}).
	WithValidator(func(ctx context.Context, value []string) (string, error) {
		strFilter := strings.Join(value, " ")
		result, err := gcpqueryutil.ParseSetFilter(strFilter, inputCSMAliasMap, true, true, true)
		if err != nil {
			return "", err
		}
		if result.ValidationError == "" {
			err = verifyResponseFlags(convertInputOnlyResponseFlagToActualFlag(result.Additives))
			if err != nil {
				return err.Error(), nil
			}
			err = verifyResponseFlags(convertInputOnlyResponseFlagToActualFlag(result.Subtractives))
			if err != nil {
				return err.Error(), nil
			}
		}
		return result.ValidationError, nil
	}).
	WithConverter(func(ctx context.Context, value []string) (*gcpqueryutil.SetFilterParseResult, error) {
		strFilter := strings.Join(value, " ")
		result, err := gcpqueryutil.ParseSetFilter(strFilter, inputCSMAliasMap, true, true, true)
		if err != nil {
			return nil, err
		}
		result.Additives = convertInputOnlyResponseFlagToActualFlag(result.Additives)
		result.Subtractives = convertInputOnlyResponseFlagToActualFlag(result.Subtractives)
		return result, nil
	}).
	Build()

// convertInputOnlyResponseFlagToActualFlag replaces "OK" included in the given flag array to "-" and all other lower cased flags to upper case.
func convertInputOnlyResponseFlagToActualFlag(flags []string) []string {
	result := make([]string, 0, len(flags))
	for _, flag := range flags {
		if flag == "ok" {
			result = append(result, string(googlecloudlogcsm_contract.ResponseFlagNoError))
		} else {
			result = append(result, strings.ToUpper(flag))
		}
	}
	return result
}

func verifyResponseFlags(flags []string) error {
	for _, flag := range flags {
		if _, found := googlecloudlogcsm_contract.HumanReadableErrorMessage[googlecloudlogcsm_contract.ResponseFlag(flag)]; !found {
			return fmt.Errorf("unknown response flag: %q: %w", flag, khierrors.ErrInvalidInput)
		}
	}
	return nil
}
