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

package label

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/task"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestQueryTaskLabelOpt(t *testing.T) {
	labelOpt := NewQueryTaskLabelOpt(enum.LogTypeComputeApi, "sample query")
	label := task.NewLabelSet(labelOpt)

	anyQueryTask, exists := typedmap.Get(label, TaskLabelKeyIsQueryTask)
	if !exists {
		t.Errorf("TaskLabel %s is expected to be set, but it is not", TaskLabelKeyIsQueryTask.Key())
	}
	if anyQueryTask != true {
		t.Errorf("TaskLabel %s is expected to be true, but it is %v", TaskLabelKeyIsQueryTask.Key(), anyQueryTask)
	}

	targetLogType, exists := typedmap.Get(label, TaskLabelKeyQueryTaskTargetLogType)
	if !exists {
		t.Errorf("TaskLabel %s is expected to be set, but it is not", TaskLabelKeyQueryTaskTargetLogType.Key())
	}
	if targetLogType != enum.LogTypeComputeApi {
		t.Errorf("TaskLabel %s is expected to be %v, but it is %v", TaskLabelKeyQueryTaskTargetLogType.Key(), enum.LogTypeComputeApi, targetLogType)
	}

	sampleQuery, exists := typedmap.Get(label, TaskLabelKeyQueryTaskSampleQuery)
	if !exists {
		t.Errorf("TaskLabel %s is expected to be set, but it is not", TaskLabelKeyQueryTaskSampleQuery.Key())
	}
	if sampleQuery != "sample query" {
		t.Errorf("TaskLabel %s is expected to be sample query, but it is %v", TaskLabelKeyQueryTaskSampleQuery.Key(), sampleQuery)
	}
}
