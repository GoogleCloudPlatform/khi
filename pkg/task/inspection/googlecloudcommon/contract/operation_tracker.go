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

package googlecloudcommon_contract

import (
	"context"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
)

// GCPOperationTracker tracks operation start/finish logs within a group and generates revisions.
type GCPOperationTracker struct {
	startedOperations map[string]struct{}
	lastManifest      string
}

// NewGCPOperationTracker creates a new GCPOperationTracker.
func NewGCPOperationTracker() *GCPOperationTracker {
	return &GCPOperationTracker{
		startedOperations: make(map[string]struct{}),
	}
}

// TrackAndGetManifest tracks the latest resource manifest from an audit log and returns it if updated.
func (t *GCPOperationTracker) TrackAndGetManifest(audit *GCPAuditLogFieldSet) (string, bool) {
	manifest := ""
	if resp, err := audit.ResponseString(); err == nil && resp != "" {
		manifest = resp
	} else if req, err := audit.RequestString(); err == nil && req != "" {
		manifest = req
	}
	if manifest == "" {
		return "", false
	}
	if t.lastManifest == manifest {
		return manifest, false
	}
	t.lastManifest = manifest
	return manifest, true
}

// ProcessOperationLog adds necessary operation revisions or events to the TimelineChangeSet.
// If an ending log is encountered without a prior starting log, it automatically prepends
// a dummy starting revision with RevisionStateOperationStartedLogNotFound at Unix time 0.
func (t *GCPOperationTracker) ProcessOperationLog(ctx context.Context, cs *khifilev6.TimelineChangeSet, targetPath *khifilev6.TimelinePath, audit *GCPAuditLogFieldSet, timestamp time.Time) {
	if audit.ImmediateOperation() {
		cs.AddEvent(targetPath)
		return
	}

	if audit.Starting() {
		t.startedOperations[audit.OperationID] = struct{}{}
	}

	_, hasStarted := t.startedOperations[audit.OperationID]
	if audit.Ending() && !hasStarted {
		cs.AddRevision(targetPath, &khifilev6.StagingRevision{
			VerbType:    VerbOperationStart,
			StateType:   RevisionStateOperationStartedLogNotFound,
			Principal:   audit.PrincipalEmail,
			ChangedTime: time.Unix(0, 0),
		})
	}

	revisionState := RevisionStateOperationStarted
	verb := VerbOperationStart
	if audit.Ending() {
		if audit.Status <= 0 { // -1 should be the default value of the field set reader.
			revisionState = RevisionStateOperationSucceed
		} else {
			revisionState = RevisionStateOperationFailed
		}
		verb = VerbOperationFinish
	}

	var body structured.Node
	if audit.Request != nil {
		body = audit.Request.Node
	}
	cs.AddRevision(targetPath, &khifilev6.StagingRevision{
		ResourceBody: body,
		VerbType:     verb,
		StateType:    revisionState,
		Principal:    audit.PrincipalEmail,
		ChangedTime:  timestamp,
	})
}
