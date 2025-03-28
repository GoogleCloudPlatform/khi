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
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

// ReaderFactoryGeneratorTask generates the instance of Reader factory to be used in later task.
var ReaderFactoryGeneratorTaskID = taskid.NewDefaultImplementationID[*structure.ReaderFactory](InspectionTaskPrefix + "reader-factory-generator")

var ReaderFactoryGeneratorTask = task.NewTask(ReaderFactoryGeneratorTaskID, []taskid.UntypedTaskReference{}, func(ctx context.Context) (*structure.ReaderFactory, error) {
	return structure.NewReaderFactory(structuredatastore.NewLRUStructureDataStoreFactory()), nil
})
