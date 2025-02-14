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

package form

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func fieldWithIdAndPriorityForTest(id string, priority int) FormField {
	return FormField{
		Id:       id,
		Priority: priority,
	}
}

func TestFormFieldSetShouldSortOnAddingNewField(t *testing.T) {
	fsActual := (&FormFieldSetMetadataFactory{}).Instanciate().(*FormFieldSet)
	fsActual.SetField(fieldWithIdAndPriorityForTest("foo", 1))
	fsActual.SetField(fieldWithIdAndPriorityForTest("bar", 3))
	fsActual.SetField(fieldWithIdAndPriorityForTest("qux", 2))

	fsExpected := &FormFieldSet{
		fields: []FormField{
			fieldWithIdAndPriorityForTest("bar", 3),
			fieldWithIdAndPriorityForTest("qux", 2),
			fieldWithIdAndPriorityForTest("foo", 1),
		},
	}

	if diff := cmp.Diff(fsActual, fsExpected, cmp.AllowUnexported(FormFieldSet{}), cmpopts.IgnoreFields(FormFieldSet{}, "fieldsLock")); diff != "" {
		t.Errorf("FieldSet has fields in unexpected shape\n%v", diff)
	}
}
