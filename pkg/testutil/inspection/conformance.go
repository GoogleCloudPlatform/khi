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

package inspection_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

// ConformanceEveryInspectionTasksAreResolvable verify the InspectionTaskServer initialzied with the given preparation method must be resolvable by each tasks.
func ConformanceEveryInspectionTasksAreResolvable(t *testing.T, label string, preps []inspection.PrepareInspectionServerFunc) {
	testServer, err := inspection.NewServer()
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	for _, prep := range preps {
		err := prep(testServer)
		if err != nil {
			t.Errorf("unexpected error %v. failed to complete the preparation step", err)
		}
	}

	for _, targetTask := range testServer.GetAllRegisteredTasks() {
		t.Run(fmt.Sprintf("%s-only-contains-%s-must-be-resolvable", label, targetTask.UntypedID().String()), func(t *testing.T) {
			availableSet, err := task.NewTaskSet(testServer.GetAllRegisteredTasks())
			if err != nil {
				t.Errorf("unexpected error %v", err)
			}
			originalSet, err := task.NewTaskSet([]task.UntypedTask{targetTask})
			if err != nil {
				t.Errorf("unexpected error %v", err)
			}

			rs, err := originalSet.ResolveTask(availableSet)
			if err != nil {
				t.Errorf("given graph with a single task %s couldn't be resolved.\n unexpected error %v", targetTask.UntypedID().String(), err)
			}
			graphViz, err := rs.DumpGraphviz()
			if err != nil {
				t.Errorf("unexpected error\n%v", err)
			}
			fmt.Printf("graphviz:\n%s\n%s\n", targetTask.UntypedID().String(), graphViz)
		})
	}
}

func ConformanceTestForInspectionTypes(t *testing.T, preps []inspection.PrepareInspectionServerFunc) {
	testServer, err := inspection.NewServer()
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	for _, prep := range preps {
		err := prep(testServer)
		if err != nil {
			t.Errorf("unexpected error %v. failed to complete the preparation step", err)
		}
	}

	for _, inspectionType := range testServer.GetAllInspectionTypes() {
		t.Run(fmt.Sprintf("%s-contains-at-least-one-feature", inspectionType.Name), func(t *testing.T) {
			taskId, err := testServer.CreateInspection(inspectionType.Id)
			if err != nil {
				t.Errorf("unexpected error\n%v", err)
			}
			features, err := testServer.GetTask(taskId).FeatureList()
			if err != nil {
				t.Errorf("unexpected error\n%v", err)
			}
			if len(features) == 0 {
				t.Errorf("feature=`%s` had no feature", inspectionType.Name)
			}
			result := ""
			for _, feature := range features {
				result += fmt.Sprintf("* %s", feature.Label)
			}
			fmt.Printf("Feature=%s\n%s\n", inspectionType.Id, result)
		})

		// icons must be in relative path for frontend to read it when the base path was rewritten
		t.Run(fmt.Sprintf("%s-icon-must-be-relative-path", inspectionType.Name), func(t *testing.T) {
			if strings.HasPrefix(inspectionType.Icon, "/") {
				t.Errorf("icon path must be relative path, got %s", inspectionType.Icon)
			}
		})
	}
}
