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

package inspectionmetadata

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func fieldWithIdAndPriorityForTest(id string, priority int) TextParameterFormField {
	return TextParameterFormField{
		ParameterFormFieldBase: ParameterFormFieldBase{
			ID:       id,
			Priority: priority,
		},
	}
}

func TestFormFieldSetShouldSortOnAddingNewField(t *testing.T) {
	fsActual := NewFormFieldSetMetadata()
	fsActual.SetField(fieldWithIdAndPriorityForTest("foo", 1))
	fsActual.SetField(fieldWithIdAndPriorityForTest("bar", 3))
	fsActual.SetField(fieldWithIdAndPriorityForTest("qux", 2))
	fsActual.SetField(fieldWithIdAndPriorityForTest("aqux", 2))

	fsExpected := &FormFieldSetMetadata{
		fields: []ParameterFormField{
			fieldWithIdAndPriorityForTest("bar", 3),
			fieldWithIdAndPriorityForTest("aqux", 2),
			fieldWithIdAndPriorityForTest("qux", 2),
			fieldWithIdAndPriorityForTest("foo", 1),
		},
	}

	if diff := cmp.Diff(fsActual, fsExpected, cmp.AllowUnexported(FormFieldSetMetadata{}), cmpopts.IgnoreFields(FormFieldSetMetadata{}, "fieldsLock")); diff != "" {
		t.Errorf("FieldSet has fields in unexpected shape\n%v", diff)
	}
}

func TestGetParameterFormFieldBase(t *testing.T) {
	testCases := []struct {
		name  string
		input ParameterFormField
		want  ParameterFormFieldBase
	}{
		{
			name: "group field",
			input: GroupParameterFormField{
				ParameterFormFieldBase: ParameterFormFieldBase{
					ID:    "group-1",
					Label: "Group 1",
					Type:  Group,
				},
			},
			want: ParameterFormFieldBase{
				ID:    "group-1",
				Label: "Group 1",
				Type:  Group,
			},
		},
		{
			name: "text field",
			input: TextParameterFormField{
				ParameterFormFieldBase: ParameterFormFieldBase{
					ID:    "text-1",
					Label: "Text 1",
					Type:  Text,
				},
			},
			want: ParameterFormFieldBase{
				ID:    "text-1",
				Label: "Text 1",
				Type:  Text,
			},
		},
		{
			name: "set field",
			input: SetParameterFormField{
				ParameterFormFieldBase: ParameterFormFieldBase{
					ID:    "set-1",
					Label: "Set 1",
					Type:  Set,
				},
			},
			want: ParameterFormFieldBase{
				ID:    "set-1",
				Label: "Set 1",
				Type:  Set,
			},
		},
		{
			name: "file field",
			input: FileParameterFormField{
				ParameterFormFieldBase: ParameterFormFieldBase{
					ID:    "file-1",
					Label: "File 1",
					Type:  File,
				},
			},
			want: ParameterFormFieldBase{
				ID:    "file-1",
				Label: "File 1",
				Type:  File,
			},
		},
		{
			name: "checkbox field",
			input: CheckboxParameterFormField{
				ParameterFormFieldBase: ParameterFormFieldBase{
					ID:    "checkbox-1",
					Label: "Checkbox 1",
					Type:  Checkbox,
				},
				Default:  true,
				Readonly: false,
			},
			want: ParameterFormFieldBase{
				ID:    "checkbox-1",
				Label: "Checkbox 1",
				Type:  Checkbox,
			},
		},
		{
			name:  "unknown field",
			input: struct{}{},
			want:  ParameterFormFieldBase{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := GetParameterFormFieldBase(tc.input)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GetParameterFormFieldBase() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
