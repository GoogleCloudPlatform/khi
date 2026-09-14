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

package commonlogk8saudit_impl

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// emptyInitialResourceStateProvider reports no initial state at all. It backs the environments that have
// no resource inventory to read the pre inspection state from.
type emptyInitialResourceStateProvider struct{}

var _ commonlogk8saudit_contract.InitialResourceStateProvider = (*emptyInitialResourceStateProvider)(nil)

// InitialResourceState implements commonlogk8saudit_contract.InitialResourceStateProvider.
func (p *emptyInitialResourceStateProvider) InitialResourceState(*commonlogk8saudit_contract.ResourceIdentity) (*structured.NodeReader, bool) {
	return nil, false
}

// EmptyInitialResourceStateProviderTask is the fallback provider used where no resource inventory exists.
// An environment with an inventory overrides it with a higher task selection priority.
var EmptyInitialResourceStateProviderTask = inspectiontaskbase.NewInspectionTask(
	taskid.NewImplementationID(commonlogk8saudit_contract.InitialResourceStateProviderRef, "empty"),
	nil,
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (commonlogk8saudit_contract.InitialResourceStateProvider, error) {
		return &emptyInitialResourceStateProvider{}, nil
	},
)
