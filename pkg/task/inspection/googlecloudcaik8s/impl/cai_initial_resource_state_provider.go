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
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudcaik8s_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcaik8s/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// caiInitialResourceStateProvider serves the manifests Cloud Asset Inventory reported for the moment
// the inspection window opened.
type caiInitialResourceStateProvider struct {
	bodiesByIdentity map[string]*structured.NodeReader
}

var _ commonlogk8saudit_contract.InitialResourceStateProvider = (*caiInitialResourceStateProvider)(nil)

// InitialResourceState implements commonlogk8saudit_contract.InitialResourceStateProvider.
func (p *caiInitialResourceStateProvider) InitialResourceState(identity *commonlogk8saudit_contract.ResourceIdentity) (*structured.NodeReader, bool) {
	body, found := p.bodiesByIdentity[initialResourceStateKey(identity)]
	return body, found
}

// newCAIInitialResourceStateProvider indexes the snapshot logs that were current at 'at'. When several
// snapshots of the same resource qualify, the one that became current last wins.
func newCAIInitialResourceStateProvider(logs []*log.Log, at time.Time) *caiInitialResourceStateProvider {
	bodiesByIdentity := map[string]*structured.NodeReader{}
	observedTimes := map[string]time.Time{}

	for _, l := range logs {
		assetWindowStartTime, assetWindowEndTime, isDeleted := extractTimeWindow(l.NodeReader)
		if isDeleted || !isActiveAt(assetWindowStartTime, assetWindowEndTime, at) {
			continue
		}
		identity := extractResourceIdentityFromLog(l.NodeReader)
		if identity.Kind == "" || identity.Name == "" {
			continue
		}
		body := extractResourceBody(l.NodeReader)
		if body == nil {
			continue
		}

		key := initialResourceStateKey(identity)
		if previous, found := observedTimes[key]; found && previous.After(assetWindowStartTime) {
			continue
		}
		bodiesByIdentity[key] = structured.NewNodeReader(body)
		observedTimes[key] = assetWindowStartTime
	}

	return &caiInitialResourceStateProvider{bodiesByIdentity: bodiesByIdentity}
}

// initialResourceStateKey renders the lookup key of a CAI resource identity. CAI leaves the namespace
// empty for cluster scoped resources while the audit log pipeline names it ClusterScopeNamespace, so
// the key adopts the audit log convention.
func initialResourceStateKey(identity *commonlogk8saudit_contract.ResourceIdentity) string {
	if identity.Namespace != "" {
		return identity.String()
	}
	clusterScoped := *identity
	clusterScoped.Namespace = commonlogk8saudit_contract.ClusterScopeNamespace
	return clusterScoped.String()
}

// InitialResourceStateProviderTask supplies the audit log parser with the resource manifests CAI
// observed before the inspection window.
//
// The subsequent task label pulls the CAI timeline mapper into the graph whenever this provider is
// selected, so enabling the Kubernetes audit log feature alone also renders the CAI revisions.
var InitialResourceStateProviderTask = inspectiontaskbase.NewInspectionTask(
	taskid.NewImplementationID(commonlogk8saudit_contract.InitialResourceStateProviderRef, "cai"),
	[]taskid.UntypedTaskReference{
		googlecloudcaik8s_contract.RawLogTaskID.Ref(),
		googlecloudcommon_contract.InputStartTimeTaskID.Ref(),
	},
	func(ctx context.Context, _ inspectioncore_contract.InspectionTaskModeType) (commonlogk8saudit_contract.InitialResourceStateProvider, error) {
		logs := coretask.GetTaskResult(ctx, googlecloudcaik8s_contract.RawLogTaskID.Ref())
		queryStartTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputStartTimeTaskID.Ref())
		return newCAIInitialResourceStateProvider(logs, queryStartTime), nil
	},
	coretask.WithSelectionPriority(1000),
	coretask.NewSubsequentTaskRefsTaskLabel(googlecloudcaik8s_contract.LogToTimelineMapperTaskID.Ref()),
)
