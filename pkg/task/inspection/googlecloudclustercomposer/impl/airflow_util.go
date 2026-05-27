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
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudclustercomposer_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/contract"
)

// tiStatusToVerb converts Taskinstance status to (*pb.Verb, *pb.RevisionState).
func tiStatusToVerb(ti *googlecloudclustercomposer_contract.AirflowTaskInstance) (*pb.Verb, *pb.RevisionState) {
	switch ti.Status() {
	case googlecloudclustercomposer_contract.TASKINSTANCE_SCHEDULED:
		return googlecloudclustercomposer_contract.VerbComposerTaskInstanceScheduled, googlecloudclustercomposer_contract.RevisionStateComposerTiScheduled
	case googlecloudclustercomposer_contract.TASKINSTANCE_QUEUED:
		return googlecloudclustercomposer_contract.VerbComposerTaskInstanceQueued, googlecloudclustercomposer_contract.RevisionStateComposerTiQueued
	case googlecloudclustercomposer_contract.TASKINSTANCE_RUNNING:
		return googlecloudclustercomposer_contract.VerbComposerTaskInstanceRunning, googlecloudclustercomposer_contract.RevisionStateComposerTiRunning
	case googlecloudclustercomposer_contract.TASKINSTANCE_SUCCESS:
		return googlecloudclustercomposer_contract.VerbComposerTaskInstanceSuccess, googlecloudclustercomposer_contract.RevisionStateComposerTiSuccess
	case googlecloudclustercomposer_contract.TASKINSTANCE_FAILED:
		return googlecloudclustercomposer_contract.VerbComposerTaskInstanceFailed, googlecloudclustercomposer_contract.RevisionStateComposerTiFailed
	case googlecloudclustercomposer_contract.TASKINSTANCE_DEFERRED:
		return googlecloudclustercomposer_contract.VerbComposerTaskInstanceDeferred, googlecloudclustercomposer_contract.RevisionStateComposerTiDeferred
	case googlecloudclustercomposer_contract.TASKINSTANCE_UP_FOR_RETRY:
		return googlecloudclustercomposer_contract.VerbComposerTaskInstanceUpForRetry, googlecloudclustercomposer_contract.RevisionStateComposerTiUpForRetry
	case googlecloudclustercomposer_contract.TASKINSTANCE_UP_FOR_RESCHEDULE:
		return googlecloudclustercomposer_contract.VerbComposerTaskInstanceUpForReschedule, googlecloudclustercomposer_contract.RevisionStateComposerTiUpForReschedule
	case googlecloudclustercomposer_contract.TASKINSTANCE_REMOVED:
		return googlecloudclustercomposer_contract.VerbComposerTaskInstanceRemoved, googlecloudclustercomposer_contract.RevisionStateComposerTiRemoved
	case googlecloudclustercomposer_contract.TASKINSTANCE_UPSTREAM_FAILED:
		return googlecloudclustercomposer_contract.VerbComposerTaskInstanceUpstreamFailed, googlecloudclustercomposer_contract.RevisionStateComposerTiUpstreamFailed
	case googlecloudclustercomposer_contract.TASKINSTANCE_ZOMBIE:
		return googlecloudclustercomposer_contract.VerbComposerTaskInstanceZombie, googlecloudclustercomposer_contract.RevisionStateComposerTiZombie
	case googlecloudclustercomposer_contract.TASKINSTANCE_SKIPPED:
		return googlecloudclustercomposer_contract.VerbComposerTaskInstanceSkipped, googlecloudclustercomposer_contract.RevisionStateComposerTiSkipped
	default:
		return googlecloudclustercomposer_contract.VerbComposerTaskInstanceUnimplemented, commonlogk8saudit_contract.RevisionStateConditionUnknown
	}
}
