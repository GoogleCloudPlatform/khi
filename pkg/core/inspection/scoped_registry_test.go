// Copyright 2026 Google LLC
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

package coreinspection

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
)

func TestScopedRegistry(t *testing.T) {
	tests := []struct {
		name         string
		scopeOptions []coretask.LabelOpt
		wantSelector inspectioncore_contract.LabelSelector
	}{
		{
			name: "should inherit platform:gke label from scoped registry",
			scopeOptions: []coretask.LabelOpt{
				inspectioncore_contract.InspectionTypeLabelSelector(map[string]string{"platform": "gke"}),
			},
			wantSelector: inspectioncore_contract.LabelSelector{"platform": "gke"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ioConfig := &inspectioncore_contract.IOConfig{}
			server, _ := NewServer(ioConfig)

			scoped := NewScopedRegistry(server, tt.scopeOptions...)

			task := coretask.NewTask(
				taskid.NewDefaultImplementationID[any]("scoped-task"),
				nil,
				func(ctx context.Context) (any, error) { return nil, nil },
			)

			if err := scoped.AddTask(task); err != nil {
				t.Fatalf("AddTask failed: %v", err)
			}

			registeredTask, err := server.RootTaskSet.Get("scoped-task#default")
			if err != nil {
				t.Fatalf("Failed to find registered task: %v", err)
			}

			labels := registeredTask.Labels()
			gotSelector, ok := typedmap.Get(labels, inspectioncore_contract.LabelKeyInspectionTypeLabelSelector)
			if !ok {
				t.Fatalf("LabelKeyInspectionTypeLabelSelector not found")
			}

			if diff := cmp.Diff(tt.wantSelector, gotSelector); diff != "" {
				t.Errorf("LabelSelector mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
