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

package googlecloudk8scommon_impl

import (
	"strings"
	"testing"

	form_task_test "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask/test"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query/queryutil"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestInputNamespaces(t *testing.T) {
	expectedDescription := "The namespace of resources to gather logs. Specify `@all_cluster_scoped` to gather logs for all non-namespaced resources. Specify `@all_namespaced` to gather logs for all namespaced resources."
	expectedLabel := "Namespaces"
	form_task_test.TestTextForms(t, "namespaces", InputNamespaceFilterTask, []*form_task_test.TextFormTestCase{
		{
			Input: "",
			ExpectedValue: &queryutil.SetFilterParseResult{
				Additives: []string{
					"#namespaced",
					"#cluster-scoped",
				},
				Subtractives:    []string{},
				ValidationError: "",
				SubtractMode:    false,
			},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Label:       expectedLabel,
					Description: expectedDescription,
					HintType:    inspectionmetadata.Error,
					Hint:        "namespace filter can't be empty",
				},
				Readonly: false,
				Default:  "@all_cluster_scoped @all_namespaced",
			},
		},
		{
			Input: "kube-system default",
			ExpectedValue: &queryutil.SetFilterParseResult{
				Additives:       []string{"kube-system", "default"},
				Subtractives:    []string{},
				ValidationError: "",
				SubtractMode:    false,
			},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Label:       expectedLabel,
					Description: expectedDescription,
					HintType:    inspectionmetadata.None,
				},
				Readonly: false,
				Default:  "@all_cluster_scoped @all_namespaced",
			},
		},
		{
			Input: "@all_cluster_scoped @all_namespaced",
			ExpectedValue: &queryutil.SetFilterParseResult{
				Additives:       []string{"#namespaced", "#cluster-scoped"},
				Subtractives:    []string{},
				ValidationError: "",
				SubtractMode:    false,
			}, ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Label:       expectedLabel,
					Description: expectedDescription,
					HintType:    inspectionmetadata.None,
				},
				Readonly: false,
				Default:  "@all_cluster_scoped @all_namespaced",
			},
		},
	}, cmpopts.SortSlices(func(a string, b string) bool {
		return strings.Compare(a, b) > 0
	}))
}
