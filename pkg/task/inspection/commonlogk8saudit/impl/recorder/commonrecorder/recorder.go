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
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_impl "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/impl"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/impl/recorder"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/impl/recorder/recorderutil"

	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
)

type commonRecorderStatus struct {
	parentDeletionHandler *recorderutil.ParentDeletionHandler
}

func Register(manager *recorder.RecorderTaskManager) error {
	manager.AddRecorder("common", []taskid.UntypedTaskReference{}, func(ctx context.Context, req *recorder.RecorderRequest) (any, error) {
		return recordChangeSetForLog(ctx, req)
	}, recorder.AnyLogGroupFilter(), recorder.AnyLogFilter())
	return nil
}

func recordChangeSetForLog(ctx context.Context, req *recorder.RecorderRequest) (*commonRecorderStatus, error) {
	commonField := log.MustGetFieldSet(req.LogParseResult.Log, &log.CommonFieldSet{})

	resourcePath := resourcepath.ResourcePath{
		Path:               req.TimelineResourceStringPath,
		ParentRelationship: enum.RelationshipChild,
	}
	var prevState *commonRecorderStatus
	if req.PreviousState == nil {
		prevState = &commonRecorderStatus{
			parentDeletionHandler: &recorderutil.ParentDeletionHandler{},
		}
	} else {
		prevState = req.PreviousState.(*commonRecorderStatus)
	}

	if req.LogParseResult.IsErrorResponse {
		req.ChangeSet.AddEvent(resourcePath)
		req.ChangeSet.SetLogSeverity(enum.SeverityError)
		req.ChangeSet.SetLogSummary(fmt.Sprintf("【%s】%s", req.LogParseResult.ResponseErrorMessage, req.LogParseResult.RequestTarget))
		return prevState, nil
	}
	if !req.LogParseResult.GeneratedFromDeleteCollectionOperation {
		logSummary := fmt.Sprintf("%s on %s.%s.%s(%s in %s)", enum.RevisionVerbs[req.LogParseResult.Operation.Verb].Label, req.LogParseResult.Operation.Namespace, req.LogParseResult.Operation.Name, req.LogParseResult.Operation.SubResourceName, req.LogParseResult.Operation.PluralKind, req.LogParseResult.Operation.APIVersion)
		req.ChangeSet.SetLogSummary(logSummary)
	}

	if req.LogParseResult.Operation.Verb == enum.RevisionVerbDeleteCollection {
		return prevState, nil
	}

	if req.IsFirstLog {
		creationTime := commonlogk8saudit_impl.ParseCreationTime(req.LogParseResult.ResourceBodyReader, commonField.Timestamp)
		minimumDeltaToRecordInferredRevision := time.Second * 10
		if commonField.Timestamp.Sub(creationTime) > minimumDeltaToRecordInferredRevision {
			req.ChangeSet.AddRevision(resourcePath, &history.StagingResourceRevision{
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

	deletionStatus := commonlogk8saudit_impl.ParseDeletionStatus(ctx, req.LogParseResult.ResourceBodyReader, req.LogParseResult.Operation)
	state := enum.RevisionStateExisting
	if deletionStatus == commonlogk8saudit_impl.DeletionStatusDeleting {
		state = enum.RevisionStateDeleting
	} else if deletionStatus == commonlogk8saudit_impl.DeletionStatusDeleted {
		state = enum.RevisionStateDeleted
	}

	prevState.parentDeletionHandler.BeforeRecordingCurrent(req, []resourcepath.ResourcePath{resourcePath})

	req.ChangeSet.AddRevision(resourcePath, &history.StagingResourceRevision{
		Verb:       req.LogParseResult.Operation.Verb,
		Body:       req.LogParseResult.ResourceBodyYaml,
		Partial:    false,
		Requestor:  req.LogParseResult.Requestor,
		ChangeTime: commonField.Timestamp,
		State:      state,
	})

	prevState.parentDeletionHandler.AfterRecordingCurrent(req, []resourcepath.ResourcePath{resourcePath})

	return prevState, nil
}
