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

package googlecloudloggkeapiaudit_impl

import (
	"context"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudloggkeapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudloggkeapiaudit/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// emptyInitialResourceStateProvider reports no initial state. It is used when no inventory is available.
type emptyInitialResourceStateProvider struct{}

var _ googlecloudloggkeapiaudit_contract.InitialResourceStateProvider = (*emptyInitialResourceStateProvider)(nil)

// ClusterInitialState implements googlecloudloggkeapiaudit_contract.InitialResourceStateProvider.
func (p *emptyInitialResourceStateProvider) ClusterInitialState(clusterName string) (*googlecloudloggkeapiaudit_contract.InitialResourceState, bool) {
	return nil, false
}

// NodePoolInitialState implements googlecloudloggkeapiaudit_contract.InitialResourceStateProvider.
func (p *emptyInitialResourceStateProvider) NodePoolInitialState(clusterName, nodePoolName string) (*googlecloudloggkeapiaudit_contract.InitialResourceState, bool) {
	return nil, false
}

// EmptyInitialResourceStateProviderTask is the fallback provider task when CAI is not enabled.
var EmptyInitialResourceStateProviderTask = inspectiontaskbase.NewInspectionTask(
	taskid.NewImplementationID(googlecloudloggkeapiaudit_contract.InitialResourceStateProviderRef, "empty"),
	nil,
	func(ctx context.Context, taskMode inspectioncore.InspectionTaskModeType) (googlecloudloggkeapiaudit_contract.InitialResourceStateProvider, error) {
		return &emptyInitialResourceStateProvider{}, nil
	},
)
