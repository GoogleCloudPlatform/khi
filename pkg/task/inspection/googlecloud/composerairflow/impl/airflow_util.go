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

package composerairflow_impl

import (
	"context"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/composerairflow"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
)

// processAirflowLog extracts common metadata such as log type, timestamp, severity, and summary from an Airflow log.
func processAirflowLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}
	cs.SetLogType(composerairflow.LogTypeManagedAirflowEnvironment)
	cs.SetTimestamp(l.Timestamp)

	if severity, err := gcpcommon.ExtractGCPSeverity(l.NodeReader); err == nil {
		cs.SetSeverity(severity)
	}

	if message, err := gcpcommon.ExtractGCPMainMessage(l.NodeReader); err == nil {
		cs.SetSummary(message)
	}

	return cs, nil
}

// tiStatusToVerb converts Taskinstance status to (*pb.Verb, *pb.RevisionState).
func tiStatusToVerb(ti *composerairflow.AirflowTaskInstance) (*pb.Verb, *pb.RevisionState) {
	if ti == nil {
		return composerairflow.VerbComposerTaskInstanceUnimplemented, k8saudit.RevisionStateConditionUnknown
	}
	switch ti.Status() {
	case composerairflow.TASKINSTANCE_SCHEDULED:
		return composerairflow.VerbComposerTaskInstanceScheduled, composerairflow.RevisionStateComposerTiScheduled
	case composerairflow.TASKINSTANCE_QUEUED:
		return composerairflow.VerbComposerTaskInstanceQueued, composerairflow.RevisionStateComposerTiQueued
	case composerairflow.TASKINSTANCE_RUNNING:
		return composerairflow.VerbComposerTaskInstanceRunning, composerairflow.RevisionStateComposerTiRunning
	case composerairflow.TASKINSTANCE_SUCCESS:
		return composerairflow.VerbComposerTaskInstanceSuccess, composerairflow.RevisionStateComposerTiSuccess
	case composerairflow.TASKINSTANCE_FAILED:
		return composerairflow.VerbComposerTaskInstanceFailed, composerairflow.RevisionStateComposerTiFailed
	case composerairflow.TASKINSTANCE_DEFERRED:
		return composerairflow.VerbComposerTaskInstanceDeferred, composerairflow.RevisionStateComposerTiDeferred
	case composerairflow.TASKINSTANCE_UP_FOR_RETRY:
		return composerairflow.VerbComposerTaskInstanceUpForRetry, composerairflow.RevisionStateComposerTiUpForRetry
	case composerairflow.TASKINSTANCE_UP_FOR_RESCHEDULE:
		return composerairflow.VerbComposerTaskInstanceUpForReschedule, composerairflow.RevisionStateComposerTiUpForReschedule
	case composerairflow.TASKINSTANCE_REMOVED:
		return composerairflow.VerbComposerTaskInstanceRemoved, composerairflow.RevisionStateComposerTiRemoved
	case composerairflow.TASKINSTANCE_UPSTREAM_FAILED:
		return composerairflow.VerbComposerTaskInstanceUpstreamFailed, composerairflow.RevisionStateComposerTiUpstreamFailed
	case composerairflow.TASKINSTANCE_ZOMBIE:
		return composerairflow.VerbComposerTaskInstanceZombie, composerairflow.RevisionStateComposerTiZombie
	case composerairflow.TASKINSTANCE_SKIPPED:
		return composerairflow.VerbComposerTaskInstanceSkipped, composerairflow.RevisionStateComposerTiSkipped
	default:
		return composerairflow.VerbComposerTaskInstanceUnimplemented, k8saudit.RevisionStateConditionUnknown
	}
}
