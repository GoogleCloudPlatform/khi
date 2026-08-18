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

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// completedTaskRunner implements coretask.TaskRunner for pre-computed / imported results.
type completedTaskRunner struct {
	result *typedmap.ReadonlyTypedMap
}

var _ coretask.TaskRunner = (*completedTaskRunner)(nil)

// Run does nothing as the task has already been completed.
func (c *completedTaskRunner) Run(ctx context.Context) error {
	return nil
}

// Wait returns a closed channel immediately since the task has already finished.
func (c *completedTaskRunner) Wait() <-chan interface{} {
	ch := make(chan interface{})
	close(ch)
	return ch
}

// Result returns the pre-computed task results.
func (c *completedTaskRunner) Result() (*typedmap.ReadonlyTypedMap, error) {
	return c.result, nil
}

// Tasks returns an empty task slice because no dynamic tasks were executed.
func (c *completedTaskRunner) Tasks() []coretask.UntypedTask {
	return nil
}

// AddInterceptor is a no-op for completed tasks.
func (c *completedTaskRunner) AddInterceptor(interceptor coretask.Interceptor) {}

// NewImportedInspectionRunner creates an InspectionTaskRunner initialized in completed state with imported data.
func NewImportedInspectionRunner(server *InspectionTaskServer, ioConfig *inspectioncore_contract.IOConfig, id string, store inspectioncore_contract.Store, metadata *typedmap.ReadonlyTypedMap, options ...RunContextOption) *InspectionTaskRunner {
	runner := NewInspectionRunner(server, ioConfig, id, options...)
	close(runner.runComplete)

	resultMap := typedmap.NewTypedMap()
	typedmap.Set(resultMap, typedmap.NewTypedKey[inspectioncore_contract.Store](inspectioncore_contract.SerializerTaskID.ReferenceIDString()), store)
	runner.runner = &completedTaskRunner{
		result: resultMap.AsReadonly(),
	}
	runner.metadata = metadata
	return runner
}
