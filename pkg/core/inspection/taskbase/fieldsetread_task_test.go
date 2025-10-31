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

package inspectiontaskbase

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
)

type testFieldSetFoo struct {
	Foo string
}

// Kind implements log.FieldSet.
func (t *testFieldSetFoo) Kind() string {
	return "test-foo"
}

var _ log.FieldSet = (*testFieldSetFoo)(nil)

type testFieldSetFooReader struct {
}

// FieldSetKind implements log.FieldSetReader.
func (t *testFieldSetFooReader) FieldSetKind() string {
	return "test-foo"
}

// Read implements log.FieldSetReader.
func (t *testFieldSetFooReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	foo, err := reader.ReadString("foo")
	if err != nil {
		return nil, err
	}
	return &testFieldSetFoo{
		Foo: foo,
	}, nil
}

var _ log.FieldSetReader = (*testFieldSetFooReader)(nil)

type testFieldSetBar struct {
	Bar string
}

// Kind implements log.FieldSet.
func (t *testFieldSetBar) Kind() string {
	return "test-bar"
}

var _ log.FieldSet = (*testFieldSetBar)(nil)

type testFieldSetBarReader struct {
}

// FieldSetKind implements log.FieldSetReader.
func (t *testFieldSetBarReader) FieldSetKind() string {
	return "test-bar"
}

// Read implements log.FieldSetReader.
func (t *testFieldSetBarReader) Read(reader *structured.NodeReader) (log.FieldSet, error) {
	bar, err := reader.ReadString("bar")
	if err != nil {
		return nil, err
	}
	return &testFieldSetBar{
		Bar: bar,
	}, nil
}

var _ log.FieldSetReader = (*testFieldSetBarReader)(nil)

func TestNewFieldSetReadTask(t *testing.T) {
	testCases := []struct {
		name     string
		taskMode inspectioncore_contract.InspectionTaskModeType
		logYAMLs []string
		readers  []log.FieldSetReader
		wantFoo  []*testFieldSetFoo
		wantBar  []*testFieldSetBar
	}{
		{
			name:     "TaskModeRun: should read and set fieldsets",
			taskMode: inspectioncore_contract.TaskModeRun,
			logYAMLs: []string{
				`foo: "hello"`,
				`bar: "world"`,
				`foo: "hello"
bar: "world"`,
			},
			readers: []log.FieldSetReader{&testFieldSetFooReader{}, &testFieldSetBarReader{}},
			wantFoo: []*testFieldSetFoo{
				{Foo: "hello"},
				nil,
				{Foo: "hello"},
			},
			wantBar: []*testFieldSetBar{
				nil,
				{Bar: "world"},
				{Bar: "world"},
			},
		},
		{
			name:     "TaskModeDryRun: should not read any fieldsets",
			taskMode: inspectioncore_contract.TaskModeDryRun,
			logYAMLs: []string{
				`foo: "hello"`,
			},
			readers: []log.FieldSetReader{&testFieldSetFooReader{}},
			wantFoo: []*testFieldSetFoo{nil},
			wantBar: []*testFieldSetBar{nil},
		},
		{
			name:     "TaskModeRun: should read and set fieldsets for logs over the concurrency count",
			taskMode: inspectioncore_contract.TaskModeRun,
			logYAMLs: func() []string {
				logs := make([]string, 20)
				for i := 0; i < 20; i++ {
					logs[i] = `foo: "value"`
				}
				return logs
			}(),
			readers: []log.FieldSetReader{&testFieldSetFooReader{}},
			wantFoo: func() []*testFieldSetFoo {
				foos := make([]*testFieldSetFoo, 20)
				for i := 0; i < 20; i++ {
					foos[i] = &testFieldSetFoo{Foo: "value"}
				}
				return foos
			}(),
			wantBar: make([]*testFieldSetBar, 20), // all nil
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logs := []*log.Log{}
			for _, logYaml := range tc.logYAMLs {
				l, err := log.NewLogFromYAMLString(logYaml)
				if err != nil {
					t.Fatal(err.Error())
				}
				logs = append(logs, l)
			}

			testSourceTaskID := taskid.NewDefaultImplementationID[[]*log.Log]("source")
			testTaskID := taskid.NewDefaultImplementationID[[]*log.Log]("dest")
			fieldSetReadTask := NewFieldSetReadTask(testTaskID, testSourceTaskID.Ref(), tc.readers)

			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(context.Background())
			_, _, err := inspectiontest.RunInspectionTask(ctx, fieldSetReadTask, tc.taskMode, map[string]any{}, tasktest.NewTaskDependencyValuePair(testSourceTaskID.Ref(), logs))
			if err != nil {
				t.Fatalf("RunInspectionTask returned an unexpected error: %v", err)
			}

			for i, l := range logs {
				foo, _ := log.GetFieldSet(l, &testFieldSetFoo{})
				if tc.wantFoo[i] == nil {
					if foo != nil {
						t.Errorf("log[%d]: foo fieldset: want nil, got non-nil", i)
					}
				} else if diff := cmp.Diff(tc.wantFoo[i], foo); diff != "" {
					t.Errorf("log[%d]: foo fieldset mismatch (-want +got):\n%s", i, diff)
				}

				bar, _ := log.GetFieldSet(l, &testFieldSetBar{})
				if tc.wantBar[i] == nil {
					if bar != nil {
						t.Errorf("log[%d]: bar fieldset: want nil, got non-nil", i)
					}
				} else if diff := cmp.Diff(tc.wantBar[i], bar); diff != "" {
					t.Errorf("log[%d]: bar fieldset mismatch (-want +got):\n%s", i, diff)
				}
			}
		})
	}
}
