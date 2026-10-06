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

package taskrecord

import (
	"os"
	"path/filepath"
	"testing"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
	"github.com/google/go-cmp/cmp"
)

func TestResolveTaskTypeFromTask(t *testing.T) {
	testUpstreamID := taskid.NewDefaultImplementationID[[]*log.Log]("test/upstream")
	testDownstreamID := taskid.NewDefaultImplementationID[[]string]("test/downstream")

	testCases := []struct {
		name     string
		task     coretask.UntypedTask
		wantType string
		wantOk   bool
	}{
		{
			name:     "nil task returns false",
			task:     nil,
			wantType: "",
			wantOk:   false,
		},
		{
			name:     "task returning []*log.Log",
			task:     coretask.DefineConstant[[]*log.Log](testUpstreamID, nil),
			wantType: "[]*log.Log",
			wantOk:   true,
		},
		{
			name:     "task returning []string",
			task:     coretask.DefineConstant[[]string](testDownstreamID, nil),
			wantType: "[]string",
			wantOk:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotType, gotOk := ResolveTaskTypeFromTask(tc.task)
			if diff := cmp.Diff(tc.wantOk, gotOk); diff != "" {
				t.Errorf("ResolveTaskTypeFromTask() ok mismatch (-want +got):\n%s", diff)
			}
			if tc.wantOk {
				if diff := cmp.Diff(tc.wantType, gotType.String()); diff != "" {
					t.Errorf("ResolveTaskTypeFromTask() type string mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestResolveTaskTypeFromTaskSet(t *testing.T) {
	testUpstreamID := taskid.NewDefaultImplementationID[[]*log.Log]("test/upstream")
	testDownstreamID := taskid.NewDefaultImplementationID[[]string]("test/downstream")
	testScopedID := taskid.NewDefaultImplementationID[map[string]int]("test/scoped")

	ioConfig, err := inspectioncore.NewIOConfigForTest()
	if err != nil {
		t.Fatalf("failed to create ioConfig: %v", err)
	}
	server, err := coreinspection.NewServer(ioConfig)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	err = server.AddModules(coreinspection.Module{
		Name: "test",
		Tasks: []coretask.UntypedTask{
			coretask.DefineConstant[[]*log.Log](testUpstreamID, nil),
			coretask.DefineConstant[[]string](testDownstreamID, nil),
		},
		SubModules: []coreinspection.Module{
			{
				Scope: coreinspection.Scope{"env": "test"},
				Tasks: []coretask.UntypedTask{
					coretask.DefineConstant[map[string]int](testScopedID, nil),
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to add module: %v", err)
	}

	testCases := []struct {
		name     string
		taskRef  taskid.UntypedTaskReference
		wantType string
		wantOk   bool
	}{
		{
			name:     "finds log slice task type",
			taskRef:  testUpstreamID.Ref(),
			wantType: "[]*log.Log",
			wantOk:   true,
		},
		{
			name:     "finds string slice task type",
			taskRef:  testDownstreamID.Ref(),
			wantType: "[]string",
			wantOk:   true,
		},
		{
			name:     "finds task type when wrapped by Module.Scope",
			taskRef:  testScopedID.Ref(),
			wantType: "map[string]int",
			wantOk:   true,
		},
		{
			name:     "unknown task reference returns false",
			taskRef:  taskid.NewTaskReference[int]("test/unknown"),
			wantType: "",
			wantOk:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotType, gotOk := ResolveTaskTypeFromTaskSet(server.RootTaskSet, tc.taskRef)
			if diff := cmp.Diff(tc.wantOk, gotOk); diff != "" {
				t.Errorf("ResolveTaskTypeFromTaskSet() ok mismatch (-want +got):\n%s", diff)
			}
			if tc.wantOk {
				if diff := cmp.Diff(tc.wantType, gotType.String()); diff != "" {
					t.Errorf("ResolveTaskTypeFromTaskSet() type string mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestLoadRecordedTaskResult(t *testing.T) {
	tempDir := t.TempDir()
	taskRef := taskid.NewTaskReference[[]string]("test/sample-task")

	fileName := sanitizeTaskReferenceForFileName(taskRef.ReferenceIDString()) + ".json"
	filePath := filepath.Join(tempDir, fileName)
	if err := os.WriteFile(filePath, []byte(`["hello","world"]`), 0644); err != nil {
		t.Fatalf("failed to write test fixture: %v", err)
	}

	testCases := []struct {
		name       string
		fixtureDir string
		taskRef    taskid.UntypedTaskReference
		want       []string
		wantErr    bool
	}{
		{
			name:       "successfully loads fixture",
			fixtureDir: tempDir,
			taskRef:    taskRef,
			want:       []string{"hello", "world"},
			wantErr:    false,
		},
		{
			name:       "non-existent file returns error",
			fixtureDir: tempDir,
			taskRef:    taskid.NewTaskReference[[]string]("test/non-existent"),
			want:       nil,
			wantErr:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := LoadRecordedTaskResult[[]string](tc.fixtureDir, tc.taskRef)
			if (err != nil) != tc.wantErr {
				t.Fatalf("LoadRecordedTaskResult() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr {
				if diff := cmp.Diff(tc.want, got); diff != "" {
					t.Errorf("LoadRecordedTaskResult() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
