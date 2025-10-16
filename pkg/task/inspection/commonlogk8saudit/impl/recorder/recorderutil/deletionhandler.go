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

package recorderutil

import (
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history"
	"github.com/GoogleCloudPlatform/khi/pkg/model/history/resourcepath"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/impl/recorder"
)

// ParentDeletionHandler is a utility to manage and record deletion revisions for child resources
// when their parent resources are deleted. It tracks the state of parent revisions
// to ensure that child resources correctly reflect their deletion status.
type ParentDeletionHandler struct {
	ProcessedParentRevisionIndex int
}

// BeforeRecordingCurrent checks for parent resource deletions that occurred before the current log's timestamp
// and records a corresponding "deleted" revision for the child resources if a parent was deleted.
func (p *ParentDeletionHandler) BeforeRecordingCurrent(req *recorder.RecorderRequest, currentPaths []resourcepath.ResourcePath) {
	commonField := log.MustGetFieldSet(req.LogParseResult.Log, &log.CommonFieldSet{})
	// Check parent revision from last log to current log. Insert a deleted revision on the first parent revision deleted.
	deletedByParent := false
	indexUpdated := false
	for i := p.ProcessedParentRevisionIndex; i < len(req.ParentTimelineRevisions); i++ {
		parentRevision := req.ParentTimelineRevisions[i]
		if parentRevision.ChangeTime.Sub(commonField.Timestamp) > 0 {
			p.ProcessedParentRevisionIndex = i
			indexUpdated = true
			break
		}
		if parentRevision.State == enum.RevisionStateDeleted && !deletedByParent {
			deletedByParent = true
			requestor := "error: failed to read requestor"
			if parentRevision.Requestor != nil {
				requestorBytes, err := req.Builder.BinaryBuilder.Read(parentRevision.Requestor)
				if err == nil {
					requestor = string(requestorBytes)
				}
			}
			for _, cp := range currentPaths {
				req.ChangeSet.AddRevision(cp, &history.StagingResourceRevision{
					Verb:       parentRevision.Verb,
					Body:       "",
					Partial:    false,
					Requestor:  requestor,
					ChangeTime: parentRevision.ChangeTime,
					State:      enum.RevisionStateDeleted,
				})
			}
		}
	}
	if !indexUpdated {
		p.ProcessedParentRevisionIndex = len(req.ParentTimelineRevisions)
	}
}

// AfterRecordingCurrent checks for parent resource deletions that occurred after the current log's timestamp
// (specifically when processing the last log in a group) and records a "deleted" revision for child resources.
func (p *ParentDeletionHandler) AfterRecordingCurrent(req *recorder.RecorderRequest, currentPaths []resourcepath.ResourcePath) {
	// Check parent revisions from the last log to the end. Insert a deleted revision on first parent revision deleted.
	if req.IsLastLog {
		for i := p.ProcessedParentRevisionIndex; i < len(req.ParentTimelineRevisions); i++ {
			parentRevision := req.ParentTimelineRevisions[i]
			requestor := "error: failed to read requestor"
			if parentRevision.Requestor != nil {
				requestorBytes, err := req.Builder.BinaryBuilder.Read(parentRevision.Requestor)
				if err == nil {
					requestor = string(requestorBytes)
				}
			}
			if parentRevision.State == enum.RevisionStateDeleted {
				for _, cp := range currentPaths {
					req.ChangeSet.AddRevision(cp, &history.StagingResourceRevision{
						Verb:       parentRevision.Verb,
						Body:       "",
						Partial:    false,
						Requestor:  requestor,
						ChangeTime: parentRevision.ChangeTime,
						State:      enum.RevisionStateDeleted,
					})
				}
				break
			}
		}
	}

}
