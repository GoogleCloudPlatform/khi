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

package k8scontrolplane_impl

import (
	"context"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/gcpqueryutil"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scontrolplane"
)

const priorityForControlPlaneGroup = gcpcommon.FormBasePriority + 30000

var inputControlPlaneComponentNameAliasMap map[string][]string = map[string][]string{}

// inputControlPlaneComponentNameFilterTask is a form task for filtering control plane component names.
var inputControlPlaneComponentNameFilterTask = formtask.DefineSetForm(
	k8scontrolplane.InputControlPlaneComponentNameFilterTaskID,
	priorityForControlPlaneGroup+1000,
	"Control plane component names",
	"Control plane component names to query(e.g. apiserver, controller-manager...etc)",
	func(b *coretask.Binder) formtask.SetFormSpec[*gcpqueryutil.SetFilterParseResult] {
		return formtask.SetFormSpec[*gcpqueryutil.SetFilterParseResult]{
			DefaultValue:     formtask.PreviousOrConstantDefaultValue([]string{"@any", "-apiserver"}),
			AllowCustomValue: formtask.ConstantBool(true),
			AllowAddAll:      formtask.ConstantBool(false),
			AllowRemoveAll:   formtask.ConstantBool(false),
			Options: formtask.ConstantOptions(
				inspectionmetadata.SetParameterFormFieldOptionItem{ID: "@any", Description: "[Alias]Matches any component name"},
				inspectionmetadata.SetParameterFormFieldOptionItem{ID: "apiserver", Description: "Matches logs from kube-apiserver"},
				inspectionmetadata.SetParameterFormFieldOptionItem{ID: "controller-manager", Description: "Matches logs from kube-controller-manager"},
				inspectionmetadata.SetParameterFormFieldOptionItem{ID: "scheduler", Description: "Matches logs from kube-scheduler"},
				inspectionmetadata.SetParameterFormFieldOptionItem{ID: "hpa-controller", Description: "Matches logs from horizontal pod autoscaler"},
			),
			Validator: func(ctx context.Context, value []string) (string, error) {
				strFilter := strings.Join(value, " ")
				result, err := gcpqueryutil.ParseSetFilter(strFilter, inputControlPlaneComponentNameAliasMap, true, true, true)
				if err != nil {
					return "", err
				}
				return result.ValidationError, nil
			},
			Converter: func(ctx context.Context, value []string) (*gcpqueryutil.SetFilterParseResult, error) {
				strFilter := strings.Join(value, " ")
				result, err := gcpqueryutil.ParseSetFilter(strFilter, inputControlPlaneComponentNameAliasMap, true, true, true)
				if err != nil {
					return nil, err
				}
				return result, nil
			},
		}
	},
)
