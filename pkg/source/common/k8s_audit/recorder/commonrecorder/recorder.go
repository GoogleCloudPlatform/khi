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

package commonrecorder

import (
	"context"
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/manifestutil"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/recorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/types"

	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

type commonRecorderStatus struct {
	IsFirstRevision bool
}

func Register(manager *recorder.RecorderTaskManager) error {
	manager.AddRecorder("common", []taskid.UntypedTaskReference{}, func(ctx context.Context, resourcePath string, l *types.AuditLogParserInput, prevState any, cs *history.ChangeSet, builder *history.Builder) (any, error) {
		prevTypedState := &commonRecorderStatus{
			IsFirstRevision: true,
		}
		if prevState != nil {
			prevTypedState = prevState.(*commonRecorderStatus)
		}
		return recordChangeSetForLog(ctx, resourcePath, prevTypedState, l, cs)
	}, recorder.AnyLogGroupFilter(), recorder.AnyLogFilter())
	return nil
}

func recordChangeSetForLog(ctx context.Context, resourcePathString string, prevState *commonRecorderStatus, log *types.AuditLogParserInput, cs *history.ChangeSet) (*commonRecorderStatus, error) {
	resourcePath := resourcepath.FromK8sOperation(*log.Operation)
	if log.IsErrorResponse {
		cs.RecordEvent(resourcePath)
		cs.RecordLogSeverity(enum.SeverityError)
		cs.RecordLogSummary(fmt.Sprintf("【%s】%s", log.ResponseErrorMessage, log.RequestTarget))
		return prevState, nil
	}
	if !log.GeneratedFromDeleteCollectionOperation {
		logSummary := fmt.Sprintf("%s on %s.%s.%s(%s in %s)", enum.RevisionVerbs[log.Operation.Verb].Label, log.Operation.Namespace, log.Operation.Name, log.Operation.SubResourceName, log.Operation.PluralKind, log.Operation.APIVersion)
		cs.RecordLogSummary(logSummary)
	}

	if log.Operation.Verb == enum.RevisionVerbDeleteCollection {
		return prevState, nil
	}

	if prevState.IsFirstRevision {
		creationTime := manifestutil.ParseCreationTime(log.ResourceBodyReader, log.Log.Timestamp())
		minimumDeltaToRecordInferredRevision := time.Second * 10
		if log.Log.Timestamp().Sub(creationTime) > minimumDeltaToRecordInferredRevision {
			cs.RecordRevision(resourcePath, &history.StagingResourceRevision{
				Verb: enum.RevisionVerbCreate,
				Body: `# Resource existence is inferred from '.metadata.creationTimestamp' of later logs.
# The actual resource body is not available but this resource body may be available by extending log query range.`,
				Partial:    false,
				Requestor:  "unknown",
				ChangeTime: creationTime,
				State:      enum.RevisionStateInferred,
				Inferred:   true,
			})
		}
	}

	deletionStatus := manifestutil.ParseDeletionStatus(ctx, log.ResourceBodyReader, log.Operation)
	state := enum.RevisionStateExisting
	if deletionStatus == manifestutil.DeletionStatusDeleting {
		state = enum.RevisionStateDeleting
	} else if deletionStatus == manifestutil.DeletionStatusDeleted {
		state = enum.RevisionStateDeleted
	}
	cs.RecordRevision(resourcePath, &history.StagingResourceRevision{
		Verb:       log.Operation.Verb,
		Body:       log.ResourceBodyYaml,
		Partial:    false,
		Requestor:  log.Requestor,
		ChangeTime: log.Log.Timestamp(),
		State:      state,
	})

	return &commonRecorderStatus{
		IsFirstRevision: false,
	}, nil
}
