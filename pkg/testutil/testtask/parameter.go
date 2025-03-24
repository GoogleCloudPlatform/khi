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

package testtask

import (
	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

// TestRunTaskParameterOpt is type used for the Functional Option Pattern on RunSingleTask
type TestRunTaskParameterOpt interface {
	AddParam(params map[string]any)
}

type priorTaskResultOpt struct {
	refID     string
	parameter any
}

// AddParam implements RunSingleTaskParameterOpt.
func (p *priorTaskResultOpt) AddParam(params map[string]any) {
	params[p.refID] = p.parameter
}

var _ TestRunTaskParameterOpt = (*priorTaskResultOpt)(nil)

// PriorTaskResult returns RunSingleTaskParameterOpt to fill a parameter of a task result with given value.
func PriorTaskResult[T any](task task.Definition[T], parameter T) TestRunTaskParameterOpt {
	return &priorTaskResultOpt{
		refID:     task.UntypedID().String(),
		parameter: parameter,
	}
}

// PriorTaskResultFromID returns RunSingleTaskParameterOpt to fill a parameter of a task result with given value.
func PriorTaskResultFromID[T any](id taskid.TaskImplementationID[T], parameter T) TestRunTaskParameterOpt {
	return &priorTaskResultOpt{
		refID:     id.ReferenceIDString(),
		parameter: parameter,
	}
}
