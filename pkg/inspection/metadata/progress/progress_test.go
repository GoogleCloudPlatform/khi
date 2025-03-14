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

package progress

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestGetTaskProgress(t *testing.T) {
	progress := NewProgress()
	tp, err := progress.GetTaskProgress("foo")
	if err != nil {
		t.Errorf("unexpected error %s", err)
	}

	expected := &TaskProgress{
		Id:         "foo",
		Percentage: 0,
		Message:    "",
		Label:      "foo",
	}

	if diff := cmp.Diff(expected, tp); diff != "" {
		t.Errorf("generated task progress is not containing the expected state\n%s", diff)
	}

	tp2, err := progress.GetTaskProgress("foo")
	if err != nil {
		t.Errorf("unexpected error %s", err)
	}

	if tp != tp2 {
		t.Errorf("GetTaskProgress should return the same reference for the existing task")
	}
}

func TestResolveTasks(t *testing.T) {
	progress := NewProgress()
	progress.SetTotalTaskCount(2)
	progress.GetTaskProgress("foo")
	progress.GetTaskProgress("bar")
	progress.ResolveTask("foo")

	if diff := cmp.Diff(&Progress{
		Phase:         "RUNNING",
		TotalProgress: &TaskProgress{Id: "Total", Label: "Total", Message: "1 of 2 tasks complete", Percentage: 0.5},
		TaskProgresses: []*TaskProgress{
			{
				Id:    "bar",
				Label: "bar",
			},
		},
	}, progress, cmpopts.IgnoreUnexported(Progress{})); diff != "" {
		t.Errorf("The result status is not in the expected status\n%s", diff)
	}
}

func TestDoneClearTasks(t *testing.T) {
	progress := NewProgress()
	progress.SetTotalTaskCount(2)
	progress.GetTaskProgress("foo")
	progress.GetTaskProgress("bar")
	progress.Done()

	if diff := cmp.Diff(&Progress{
		Phase:          "DONE",
		TotalProgress:  &TaskProgress{Id: "Total", Label: "Total", Message: "2 of 2 tasks complete", Percentage: 1},
		TaskProgresses: []*TaskProgress{},
	}, progress, cmpopts.IgnoreUnexported(Progress{})); diff != "" {
		t.Errorf("The result status is not in the expected status\n%s", diff)
	}
}

func TestCancelClearTasks(t *testing.T) {
	progress := NewProgress()
	progress.SetTotalTaskCount(2)
	progress.GetTaskProgress("foo")
	progress.GetTaskProgress("bar")
	progress.Cancel()

	if diff := cmp.Diff(&Progress{
		Phase:          "CANCELLED",
		TaskProgresses: []*TaskProgress{},
		TotalProgress:  &TaskProgress{Id: "Total", Label: "Total", Message: "0 of 2 tasks complete"},
	}, progress, cmpopts.IgnoreUnexported(Progress{})); diff != "" {
		t.Errorf("The result status is not in the expected status\n%s", diff)
	}
}
