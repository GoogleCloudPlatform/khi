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
)

func TestNewAliasTask(t *testing.T) {
	ctx := context.Background()
	taskDependentValues := typedmap.NewTypedMap()
	sourceTaskId := taskid.NewDefaultImplementationID[string]("source-task")
	aliasTaskId := taskid.NewDefaultImplementationID[string]("alias-task")

	typedmap.Set(taskDependentValues, typedmap.NewTypedKey[string](sourceTaskId.ReferenceIDString()), "source-value")
	ctx = khictx.WithValue(ctx, core_contract.TaskResultMapContextKey, taskDependentValues)
	ctx = khictx.WithValue(ctx, core_contract.TaskImplementationIDContextKey, taskid.UntypedTaskImplementationID(aliasTaskId))

	aliasTask := NewAliasTask(aliasTaskId, sourceTaskId.Ref())

	deps := aliasTask.Dependencies()
	if len(deps) != 1 || deps[0].String() != sourceTaskId.ReferenceIDString() {
		t.Errorf("Unexpected dependencies: %v", deps)
	}

	res, err := aliasTask.Run(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if res != "source-value" {
		t.Errorf("Expected 'source-value', but got %q", res)
	}
}
