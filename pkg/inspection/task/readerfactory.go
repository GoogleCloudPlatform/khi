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

package task

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/log/structure"
	"github.com/GoogleCloudPlatform/khi/pkg/log/structure/structuredatastore"
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	common_task "github.com/GoogleCloudPlatform/khi/pkg/task"
)

// ReaderFactoryGeneratorTask generates the instance of Reader factory to be used in later task.
const ReaderFactoryGeneratorTaskID = InspectionTaskPrefix + "reader-factory-generator"

var ReaderFactoryGeneratorTask = task.NewProcessorTask(ReaderFactoryGeneratorTaskID, []string{}, func(ctx context.Context, taskMode int, v *task.VariableSet) (any, error) {
	return structure.NewReaderFactory(structuredatastore.NewLRUStructureDataStoreFactory()), nil
})

func GetReaderFactoryFromTaskVariable(v *task.VariableSet) (*structure.ReaderFactory, error) {
	return common_task.GetTypedVariableFromTaskVariable[*structure.ReaderFactory](v, ReaderFactoryGeneratorTaskID, nil)
}
