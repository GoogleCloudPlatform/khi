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

package googlecloudcaik8s_impl

import (
	"context"
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	googlecloudcaik8s_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcaik8s/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudloggkeapiaudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudloggkeapiaudit/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// caiGKEInitialResourceStateProvider serves initial states of GKE clusters and node pools from CAI.
type caiGKEInitialResourceStateProvider struct {
	clusters  map[string]*googlecloudloggkeapiaudit_contract.InitialResourceState
	nodePools map[string]*googlecloudloggkeapiaudit_contract.InitialResourceState
}

var _ googlecloudloggkeapiaudit_contract.InitialResourceStateProvider = (*caiGKEInitialResourceStateProvider)(nil)

// ClusterInitialState implements googlecloudloggkeapiaudit_contract.InitialResourceStateProvider.
func (p *caiGKEInitialResourceStateProvider) ClusterInitialState(clusterName string) (*googlecloudloggkeapiaudit_contract.InitialResourceState, bool) {
	state, found := p.clusters[clusterName]
	return state, found
}

// NodePoolInitialState implements googlecloudloggkeapiaudit_contract.InitialResourceStateProvider.
func (p *caiGKEInitialResourceStateProvider) NodePoolInitialState(clusterName, nodePoolName string) (*googlecloudloggkeapiaudit_contract.InitialResourceState, bool) {
	state, found := p.nodePools[nodePoolKey(clusterName, nodePoolName)]
	return state, found
}

func nodePoolKey(clusterName, nodePoolName string) string {
	return fmt.Sprintf("%s/%s", clusterName, nodePoolName)
}

// newCAIGKEInitialResourceStateProvider extracts active cluster and nodepool initial states at queryStartTime.
func newCAIGKEInitialResourceStateProvider(snapshots []*googlecloudcaik8s_contract.GKEResourceSnapshot, queryStartTime time.Time) *caiGKEInitialResourceStateProvider {
	clusters := map[string]*googlecloudloggkeapiaudit_contract.InitialResourceState{}
	nodePools := map[string]*googlecloudloggkeapiaudit_contract.InitialResourceState{}

	for _, s := range snapshots {
		if s == nil || s.TemporalAsset == nil || s.TemporalAsset.Asset == nil {
			continue
		}
		ta := s.TemporalAsset
		if ta.Deleted {
			continue
		}

		var windowStartTime, windowEndTime time.Time
		if w := ta.GetWindow(); w != nil {
			if st := w.GetStartTime(); st != nil {
				windowStartTime = st.AsTime()
			}
			if et := w.GetEndTime(); et != nil {
				windowEndTime = et.AsTime()
			}
		}

		if !isActiveAt(windowStartTime, windowEndTime, queryStartTime) {
			continue
		}

		identity := parseGKEAssetName(ta.Asset.Name)
		if identity.ClusterName == "" && identity.NodePoolName == "" {
			continue
		}

		var resourceBody structured.Node
		if res := ta.Asset.GetResource(); res != nil && res.GetData() != nil {
			dataMap := res.GetData().AsMap()
			if node, err := structured.FromGoValue(dataMap, &structured.AlphabeticalGoMapKeyOrderProvider{}); err == nil {
				resourceBody = node
			}
		}
		if resourceBody == nil {
			continue
		}

		initialState := &googlecloudloggkeapiaudit_contract.InitialResourceState{
			ResourceBody: resourceBody,
		}

		if identity.IsCluster() {
			clusters[identity.ClusterName] = initialState
		} else if identity.IsNodePool() {
			nodePools[nodePoolKey(identity.ClusterName, identity.NodePoolName)] = initialState
		}
	}

	return &caiGKEInitialResourceStateProvider{
		clusters:  clusters,
		nodePools: nodePools,
	}
}

// GKEInitialResourceStateProviderTask supplies initial GKE resource manifests to the audit log parser.
var GKEInitialResourceStateProviderTask = inspectiontaskbase.NewInspectionTask(
	taskid.NewImplementationID(googlecloudloggkeapiaudit_contract.InitialResourceStateProviderRef, "cai"),
	[]coretask.Dependency{
		googlecloudcaik8s_contract.GKEResourceFetcherTaskID.Ref(),
		googlecloudcommon_contract.InputStartTimeTaskID.Ref(),
	},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (googlecloudloggkeapiaudit_contract.InitialResourceStateProvider, error) {
		if taskMode == inspectioncore_contract.TaskModeDryRun {
			return &caiGKEInitialResourceStateProvider{
				clusters:  map[string]*googlecloudloggkeapiaudit_contract.InitialResourceState{},
				nodePools: map[string]*googlecloudloggkeapiaudit_contract.InitialResourceState{},
			}, nil
		}
		snapshots := coretask.GetTaskResult(ctx, googlecloudcaik8s_contract.GKEResourceFetcherTaskID.Ref())
		queryStartTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputStartTimeTaskID.Ref())
		return newCAIGKEInitialResourceStateProvider(snapshots, queryStartTime), nil
	},
	coretask.WithSelectionPriority(1000),
	coretask.NewSubsequentTaskRefsTaskLabel(googlecloudcaik8s_contract.GKELogToTimelineMapperTaskID.Ref()),
)
