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

package apacheairflow

import (
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

var ErrorStates = []model.Tistate{
	model.TASKINSTANCE_FAILED,
	model.TASKINSTANCE_UPSTREAM_FAILED,
	model.TASKINSTANCE_ZOMBIE,
}

var WarnStates = []model.Tistate{
	model.TASKINSTANCE_UP_FOR_RETRY,
	model.TASKINSTANCE_UP_FOR_RESCHEDULE,
	model.TASKINSTANCE_REMOVED,
}

// StringToTiState converts a string representation of an Airflow task state
// to the corresponding model.Tistate constant.
func StringToTiState(stateStr string) (model.Tistate, error) {
	switch stateStr {
	case "scheduled":
		return model.TASKINSTANCE_SCHEDULED, nil
	case "queued":
		return model.TASKINSTANCE_QUEUED, nil
	case "running":
		return model.TASKINSTANCE_RUNNING, nil
	case "success":
		return model.TASKINSTANCE_SUCCESS, nil
	case "failed":
		return model.TASKINSTANCE_FAILED, nil
	case "deferred":
		return model.TASKINSTANCE_DEFERRED, nil
	case "up_for_retry":
		return model.TASKINSTANCE_UP_FOR_RETRY, nil
	case "up_for_reschedule":
		return model.TASKINSTANCE_UP_FOR_RESCHEDULE, nil
	case "removed":
		return model.TASKINSTANCE_REMOVED, nil
	case "upstream_failed":
		return model.TASKINSTANCE_UPSTREAM_FAILED, nil
	case "zombie":
		return model.TASKINSTANCE_ZOMBIE, nil
	default:
		return "", fmt.Errorf("unknown Airflow task state: %s", stateStr)
	}
}

// TiStatusToVerb converts Taskinstance status to (enum.RevisionVerb, enum.RevisionState)
func TiStatusToVerb(ti *model.AirflowTaskInstance) (enum.RevisionVerb, enum.RevisionState) {
	switch ti.Status() {
	case model.TASKINSTANCE_SCHEDULED:
		return enum.RevisionVerbComposerTaskInstanceScheduled, enum.RevisionStateComposerTiScheduled
	case model.TASKINSTANCE_QUEUED:
		return enum.RevisionVerbComposerTaskInstanceQueued, enum.RevisionStateComposerTiQueued
	case model.TASKINSTANCE_RUNNING:
		return enum.RevisionVerbComposerTaskInstanceRunning, enum.RevisionStateComposerTiRunning
	case model.TASKINSTANCE_SUCCESS:
		return enum.RevisionVerbComposerTaskInstanceSuccess, enum.RevisionStateComposerTiSuccess
	case model.TASKINSTANCE_FAILED:
		return enum.RevisionVerbComposerTaskInstanceFailed, enum.RevisionStateComposerTiFailed
	case model.TASKINSTANCE_DEFERRED:
		return enum.RevisionVerbComposerTaskInstanceDeferred, enum.RevisionStateComposerTiDeferred
	case model.TASKINSTANCE_UP_FOR_RETRY:
		return enum.RevisionVerbComposerTaskInstanceUpForRetry, enum.RevisionStateComposerTiUpForRetry
	case model.TASKINSTANCE_UP_FOR_RESCHEDULE:
		return enum.RevisionVerbComposerTaskInstanceUpForReschedule, enum.RevisionStateComposerTiUpForReschedule
	case model.TASKINSTANCE_REMOVED:
		return enum.RevisionVerbComposerTaskInstanceRemoved, enum.RevisionStateComposerTiRemoved
	case model.TASKINSTANCE_UPSTREAM_FAILED:
		return enum.RevisionVerbComposerTaskInstanceUpstreamFailed, enum.RevisionStateComposerTiUpstreamFailed
	case model.TASKINSTANCE_ZOMBIE:
		return enum.RevisionVerbComposerTaskInstanceZombie, enum.RevisionStateComposerTiZombie
	default:
		return enum.RevisionVerbComposerTaskInstanceUnimplemented, enum.RevisionStateConditionUnknown
	}
}
