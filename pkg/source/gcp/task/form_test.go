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
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common"
	form_task_test "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask/test"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/query/queryutil"
	googlecloudcommon_impl "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/impl"
	"github.com/google/go-cmp/cmp/cmpopts"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestClusterNameInput(t *testing.T) {
	wantDescription := "The cluster name to gather logs."
	testClusterNamePrefix := tasktest.StubTaskFromReferenceID(ClusterNamePrefixTaskID, "", nil)
	mockClusterNamesTask1 := tasktest.StubTaskFromReferenceID(AutocompleteClusterNamesTaskID, &AutocompleteClusterNameList{
		ClusterNames: []string{"foo-cluster", "bar-cluster"},
		Error:        "",
	}, nil)
	form_task_test.TestTextForms(t, "cluster name", InputClusterNameTask, []*form_task_test.TextFormTestCase{
		{
			Name:          "with valid cluster name",
			Input:         "foo-cluster",
			ExpectedValue: "foo-cluster",
			Dependencies:  []coretask.UntypedTask{mockClusterNamesTask1, testClusterNamePrefix},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					ID:          GCPPrefix + "input/cluster-name",
					Type:        "Text",
					Label:       "Cluster name",
					HintType:    inspectionmetadata.None,
					Description: wantDescription,
				},
				Suggestions: []string{"foo-cluster", "bar-cluster"},
				Default:     "foo-cluster",
			},
		},
		{
			Name:          "spaces around cluster name must be trimmed",
			Input:         "  foo-cluster   ",
			ExpectedValue: "foo-cluster",
			Dependencies:  []coretask.UntypedTask{mockClusterNamesTask1, testClusterNamePrefix},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					ID:          GCPPrefix + "input/cluster-name",
					Type:        "Text",
					Label:       "Cluster name",
					Description: wantDescription,
					HintType:    inspectionmetadata.None,
				},
				Suggestions: []string{"foo-cluster", "bar-cluster"},
				Default:     "foo-cluster",
			},
		},
		{
			Name:          "invalid cluster name",
			Input:         "An invalid cluster name",
			ExpectedValue: "foo-cluster",
			Dependencies:  []coretask.UntypedTask{mockClusterNamesTask1, testClusterNamePrefix},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					ID:          GCPPrefix + "input/cluster-name",
					Type:        "Text",
					Label:       "Cluster name",
					Description: wantDescription,
					HintType:    inspectionmetadata.Error,
					Hint:        "Cluster name must match `^[0-9a-z:\\-]+$`",
				},
				Suggestions: common.SortForAutocomplete("An invalid cluster name", []string{"foo-cluster", "bar-cluster"}),
				Default:     "foo-cluster",
			},
		},
		{
			Name:          "non existing cluster should show a hint",
			Input:         "nonexisting-cluster",
			ExpectedValue: "nonexisting-cluster",
			Dependencies:  []coretask.UntypedTask{mockClusterNamesTask1, testClusterNamePrefix},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					ID:          GCPPrefix + "input/cluster-name",
					Type:        "Text",
					Label:       "Cluster name",
					Description: wantDescription,
					Hint:        "Cluster `nonexisting-cluster` was not found in the specified project at this time. It works for the clusters existed in the past but make sure the cluster name is right if you believe the cluster should be there.",
					HintType:    inspectionmetadata.Warning,
				},
				Suggestions: []string{"foo-cluster", "bar-cluster"},
				Default:     "foo-cluster",
			},
		},
	})
}

func TestInputKindName(t *testing.T) {
	expectedDescription := "The kinds of resources to gather logs. `@default` is a alias of set of kinds that frequently queried. Specify `@any` to query every kinds of resources"
	expectedLabel := "Kind"
	form_task_test.TestTextForms(t, "kind", InputKindFilterTask, []*form_task_test.TextFormTestCase{
		{
			Input: "",
			ExpectedValue: &queryutil.SetFilterParseResult{
				Additives:       inputKindNameAliasMap["default"],
				Subtractives:    []string{},
				ValidationError: "",
				SubtractMode:    false,
			},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Label:       expectedLabel,
					Description: expectedDescription,
					HintType:    inspectionmetadata.Error,
					Hint:        "kind filter can't be empty",
				},
				Readonly: false,
				Default:  "@default",
			},
		},
		{
			Input: "pods replicasets",
			ExpectedValue: &queryutil.SetFilterParseResult{
				Additives:       []string{"pods", "replicasets"},
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
				Default:  "@default",
			},
		},
		{
			Input: "@invalid_alias",
			ExpectedValue: &queryutil.SetFilterParseResult{
				Additives:       inputKindNameAliasMap["default"],
				Subtractives:    []string{},
				ValidationError: "",
				SubtractMode:    false,
			}, ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Label:       expectedLabel,
					Description: expectedDescription,
					Hint:        "alias `invalid_alias` was not found",
					HintType:    inspectionmetadata.Error,
				},
				Default:  "@default",
				Readonly: false,
			},
		},
	}, cmpopts.SortSlices(func(a string, b string) bool {
		return strings.Compare(a, b) > 0
	}))
}

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

func TestNodeNameFiltertask(t *testing.T) {
	wantLabelName := "Node names"
	wantDescription := "A space-separated list of node name substrings used to collect node-related logs. If left blank, KHI gathers logs from all nodes in the cluster."
	form_task_test.TestTextForms(t, "node-name", InputNodeNameFilterTask, []*form_task_test.TextFormTestCase{
		{
			Name:          "With an empty input",
			Input:         "",
			ExpectedValue: []string{},
			Dependencies:  []coretask.UntypedTask{},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Label:       wantLabelName,
					Description: wantDescription,
					HintType:    inspectionmetadata.None,
				},
				Readonly: false,
			},
		},
		{
			Name:          "With a single node name substring",
			Input:         "node-name-1",
			ExpectedValue: []string{"node-name-1"},
			Dependencies:  []coretask.UntypedTask{},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Label:       wantLabelName,
					Description: wantDescription,
					HintType:    inspectionmetadata.None,
				},
			},
		},
		{
			Name:          "With multiple node name substrings",
			Input:         "node-name-1 node-name-2 node-name-3",
			ExpectedValue: []string{"node-name-1", "node-name-2", "node-name-3"},
			Dependencies:  []coretask.UntypedTask{},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Label:       wantLabelName,
					Description: wantDescription,
					HintType:    inspectionmetadata.None,
				},
			},
		},
		{
			Name:          "With invalid node name substring",
			Input:         "node-name-1 invalid=node=name node-name-3",
			ExpectedValue: []string{},
			Dependencies:  []coretask.UntypedTask{},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Label:       wantLabelName,
					Description: wantDescription,
					Hint:        "substring `invalid=node=name` is not valid as a substring of node name",
					HintType:    inspectionmetadata.Error,
				},
			},
		},
		{
			Name:          "With spaces around node name substring",
			Input:         "  node-name-1  node-name-2  ",
			ExpectedValue: []string{"node-name-1", "node-name-2"},
			Dependencies:  []coretask.UntypedTask{},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					Label:       wantLabelName,
					Description: wantDescription,
					HintType:    inspectionmetadata.None,
				},
			},
		},
	})
}

func TestLocationInput(t *testing.T) {
	form_task_test.TestTextForms(t, "gcp-location", InputLocationsTask, []*form_task_test.TextFormTestCase{
		{
			Name:          "With valid location",
			Input:         "asia-northeast1",
			ExpectedValue: "asia-northeast1",
			Dependencies:  []coretask.UntypedTask{AutocompleteLocationTask, googlecloudcommon_impl.InputProjectIdTask},
			ExpectedFormField: inspectionmetadata.TextParameterFormField{
				ParameterFormFieldBase: inspectionmetadata.ParameterFormFieldBase{
					ID:          GCPPrefix + "input/location",
					Type:        "Text",
					Label:       "Location",
					Description: "The location(region) to specify the resource exist(s|ed)",
					HintType:    inspectionmetadata.None,
				},
				Suggestions: []string{},
				Readonly:    false,
			},
		},
	})
}
