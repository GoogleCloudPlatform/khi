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

package log

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structurev2"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

type TestFieldSet struct {
	TestField string
}

// FieldSetKind implements FieldReader.
func (t *TestFieldSet) Kind() string {
	return "test"
}

var _ FieldSet = (*TestFieldSet)(nil)

type TestFieldSetReader struct {
}

var _ FieldSetReader = (*TestFieldSetReader)(nil)

func (t *TestFieldSetReader) FieldSetKind() string {
	return (&TestFieldSet{}).Kind()
}

func (t *TestFieldSetReader) Read(reader *structurev2.NodeReader) (FieldSet, error) {
	testField, err := reader.ReadString("test_field")
	if err != nil {
		return nil, err
	}
	return &TestFieldSet{
		TestField: testField,
	}, nil
}

func TestGetField(t *testing.T) {
	yamlNode, err := structurev2.FromYAML(`test_field: foo`)
	if err != nil {
		t.Fatal(err.Error())
	}
	nodeReader := structurev2.NewNodeReader(yamlNode)

	l := NewLog(nodeReader)
	l.SetFieldSetReader(&TestFieldSetReader{})

	f, err := GetFieldSet(l, &TestFieldSet{})
	if err != nil {
		t.Errorf("GetField() error = %v", err)
	}
	if f.TestField != "foo" {
		t.Errorf("GetField() = %v, want %v", f.TestField, "foo")
	}

	f2, err := GetFieldSet(l, &TestFieldSet{})
	if err != nil {
		t.Errorf("GetField() error = %v", err)
	}
	if f != f2 {
		t.Errorf("GetField() must return same references on multiple calls, but returned different references")
	}
}
