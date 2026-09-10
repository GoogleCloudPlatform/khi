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

package coretask

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	core_contract "github.com/GoogleCloudPlatform/khi/pkg/task/core/contract"
	"github.com/google/go-cmp/cmp"
)

func TestNewAliasTask(t *testing.T) {
	tests := []struct {
		name        string
		sourceValue string
	}{
		{
			name:        "alias task successfully forwards dependency result",
			sourceValue: "source-value",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			taskDependentValues := typedmap.NewTypedMap()
			sourceTaskID := taskid.NewDefaultImplementationID[string]("source-task")
			aliasTaskID := taskid.NewDefaultImplementationID[string]("alias-task")

			aliasTask := NewAliasTask(aliasTaskID, sourceTaskID.Ref())

			typedmap.Set(taskDependentValues, typedmap.NewTypedKey[string](sourceTaskID.ReferenceIDString()), tc.sourceValue)
			ctx = khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, taskDependentValues)
			ctx = khictx.WithValue(ctx, core_contract.TaskImplementationIDContextKey, taskid.UntypedTaskImplementationID(aliasTaskID))
			ctx = khictx.WithValue(ctx, core_contract.TaskDependenciesContextKey, aliasTask.Dependencies())

			deps := aliasTask.Dependencies()
			if len(deps) != 1 {
				t.Fatalf("unexpected dependency count: %d", len(deps))
			}
			ptp, ok := deps[0].(taskid.PointToPointDescriptor)
			if !ok || ptp.ReferenceID() != sourceTaskID.ReferenceIDString() {
				t.Errorf("unexpected dependencies: %v", deps)
			}

			res, err := aliasTask.Run(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.sourceValue, res); diff != "" {
				t.Errorf("alias task result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
