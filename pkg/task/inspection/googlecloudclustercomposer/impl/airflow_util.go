// Copyright 2025 Google LLC
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

package googlecloudclustercomposer_impl

import (
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
)

// tiStatusToVerb converts Taskinstance status to (enum.RevisionVerb, enum.RevisionState)
func tiStatusToVerb(ti *googlecloudclustercomposer_contract.AirflowTaskInstance) (enum.RevisionVerb, enum.RevisionState) {
	switch ti.Status() {
	case googlecloudclustercomposer_contract.TASKINSTANCE_SCHEDULED:
		return enum.RevisionVerbComposerTaskInstanceScheduled, enum.RevisionStateComposerTiScheduled
	case googlecloudclustercomposer_contract.TASKINSTANCE_QUEUED:
		return enum.RevisionVerbComposerTaskInstanceQueued, enum.RevisionStateComposerTiQueued
	case googlecloudclustercomposer_contract.TASKINSTANCE_RUNNING:
		return enum.RevisionVerbComposerTaskInstanceRunning, enum.RevisionStateComposerTiRunning
	case googlecloudclustercomposer_contract.TASKINSTANCE_SUCCESS:
		return enum.RevisionVerbComposerTaskInstanceSuccess, enum.RevisionStateComposerTiSuccess
	case googlecloudclustercomposer_contract.TASKINSTANCE_FAILED:
		return enum.RevisionVerbComposerTaskInstanceFailed, enum.RevisionStateComposerTiFailed
	case googlecloudclustercomposer_contract.TASKINSTANCE_DEFERRED:
		return enum.RevisionVerbComposerTaskInstanceDeferred, enum.RevisionStateComposerTiDeferred
	case googlecloudclustercomposer_contract.TASKINSTANCE_UP_FOR_RETRY:
		return enum.RevisionVerbComposerTaskInstanceUpForRetry, enum.RevisionStateComposerTiUpForRetry
	case googlecloudclustercomposer_contract.TASKINSTANCE_UP_FOR_RESCHEDULE:
		return enum.RevisionVerbComposerTaskInstanceUpForReschedule, enum.RevisionStateComposerTiUpForReschedule
	case googlecloudclustercomposer_contract.TASKINSTANCE_REMOVED:
		return enum.RevisionVerbComposerTaskInstanceRemoved, enum.RevisionStateComposerTiRemoved
	case googlecloudclustercomposer_contract.TASKINSTANCE_UPSTREAM_FAILED:
		return enum.RevisionVerbComposerTaskInstanceUpstreamFailed, enum.RevisionStateComposerTiUpstreamFailed
	case googlecloudclustercomposer_contract.TASKINSTANCE_ZOMBIE:
		return enum.RevisionVerbComposerTaskInstanceZombie, enum.RevisionStateComposerTiZombie
	default:
		return enum.RevisionVerbComposerTaskInstanceUnimplemented, enum.RevisionStateConditionUnknown
	}
}
