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

package csm_impl

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khierrors"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logutil"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/csm"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
)

const priorityForCSMGroup = gcpcommon.FormBasePriority + 10000

// inputFleetProjectIDTask is the form task for the ID of the project that hosts the Fleet and stores CSM control plane logs.
var inputFleetProjectIDTask = formtask.DefineTextForm(
	csm.InputFleetProjectIDTaskID,
	priorityForCSMGroup+500,
	"Fleet project ID",
	"The project ID where the Fleet is hosted and CSM control plane logs are stored. Default is the cluster's project ID.",
	func(b *coretask.Binder) formtask.TextFormSpec[string] {
		clusterIdentity := coretask.Use(b, csm.ClusterIdentityTaskID.Ref())
		return formtask.TextFormSpec[string]{
			DefaultValue: func(ctx context.Context, previousValues []string) (string, error) {
				if len(previousValues) > 0 {
					return previousValues[0], nil
				}
				cluster := clusterIdentity.Get(ctx)
				return cluster.ProjectID, nil
			},
		}
	},
)

var inputCSMAliasMap gcpqueryutil.SetFilterAliasToItemsMap = map[string][]string{}

// inputCSMResponseFlagsTask is the form task for the Envoy response flags that filter CSM traffic logs.
var inputCSMResponseFlagsTask = formtask.DefineSetForm(
	csm.InputCSMResponseFlagsTaskID,
	priorityForCSMGroup+1000,
	"Envoy response flags",
	"Response flags used for filtering CSM traffic logs. Note '-' in response flags is corresponded to 'OK' in this form.",
	func(b *coretask.Binder) formtask.SetFormSpec[*gcpqueryutil.SetFilterParseResult] {
		return formtask.SetFormSpec[*gcpqueryutil.SetFilterParseResult]{
			DefaultValue:     formtask.PreviousOrConstantDefaultValue([]string{"@any", "-OK"}),
			AllowAddAll:      formtask.ConstantBool(false),
			AllowRemoveAll:   formtask.ConstantBool(false),
			AllowCustomValue: formtask.ConstantBool(true),
			Options: func(ctx context.Context, previousValues []string) ([]inspectionmetadata.SetParameterFormFieldOptionItem, error) {
				result := []inspectionmetadata.SetParameterFormFieldOptionItem{
					{ID: "@any", Description: "[Alias] Matches any response flag"},
				}
				ids := make([]string, 0, len(logutil.EnvoyResponseFlagDescriptions))
				for flag := range logutil.EnvoyResponseFlagDescriptions {
					ids = append(ids, string(flag))
				}
				sort.Strings(ids)
				for _, id := range ids {
					message := logutil.EnvoyResponseFlagDescriptions[logutil.EnvoyResponseFlag(id)]
					if id == "-" {
						id = "OK"
						message = "It's '-' in the response flag field because '-' means subtracting operator in this form."
					}
					result = append(result, inspectionmetadata.SetParameterFormFieldOptionItem{ID: id, Description: message})
				}
				return result, nil
			},
			Validator: func(ctx context.Context, value []string) (string, error) {
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
			},
			Converter: func(ctx context.Context, value []string) (*gcpqueryutil.SetFilterParseResult, error) {
				strFilter := strings.Join(value, " ")
				result, err := gcpqueryutil.ParseSetFilter(strFilter, inputCSMAliasMap, true, true, true)
				if err != nil {
					return nil, err
				}
				result.Additives = convertInputOnlyResponseFlagToActualFlag(result.Additives)
				result.Subtractives = convertInputOnlyResponseFlagToActualFlag(result.Subtractives)
				return result, nil
			},
		}
	},
)

// convertInputOnlyResponseFlagToActualFlag replaces "OK" included in the given flag array to "-" and all other lower cased flags to upper case.
func convertInputOnlyResponseFlagToActualFlag(flags []string) []string {
	result := make([]string, 0, len(flags))
	for _, flag := range flags {
		if flag == "ok" {
			result = append(result, string(logutil.EnvoyResponseFlagNoError))
		} else {
			result = append(result, strings.ToUpper(flag))
		}
	}
	return result
}

func verifyResponseFlags(flags []string) error {
	for _, flag := range flags {
		if _, found := logutil.EnvoyResponseFlagDescriptions[logutil.EnvoyResponseFlag(flag)]; !found {
			return fmt.Errorf("unknown response flag: %q: %w", flag, khierrors.ErrInvalidInput)
		}
	}
	return nil
}
