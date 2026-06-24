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
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
)

// GCPOperationTracker tracks operation start/finish logs within a group and generates revisions.
type GCPOperationTracker struct {
	startedOperations   map[string]struct{}
	hasResourceRevision map[uint32]struct{}
	lastManifest        string
}

// NewGCPOperationTracker creates a new GCPOperationTracker.
func NewGCPOperationTracker() *GCPOperationTracker {
	return &GCPOperationTracker{
		startedOperations:   make(map[string]struct{}),
		hasResourceRevision: make(map[uint32]struct{}),
	}
}

// HasStarted returns true if the operation start log for the given operation ID was observed.
func (t *GCPOperationTracker) HasStarted(operationID string) bool {
	_, hasStarted := t.startedOperations[operationID]
	return hasStarted
}

// HasResourceRevision returns true if any revision has been added to the given resource timeline path.
func (t *GCPOperationTracker) HasResourceRevision(path *khifilev6.TimelinePath) bool {
	_, has := t.hasResourceRevision[path.ID]
	return has
}

// MarkResourceRevision records that a revision was added to the given resource timeline path.
func (t *GCPOperationTracker) MarkResourceRevision(path *khifilev6.TimelinePath) {
	t.hasResourceRevision[path.ID] = struct{}{}
}

// ProcessGCPClusterNodepoolOperationLog processes a GCP operation log for a cluster or node pool resource timeline.
// It generates resource creation/deletion/enrollment/unenrollment revisions, handles missing start logs by prepending
// appropriate LogNotFound revisions at Unix time 0, and updates operation tracking.
func ProcessGCPClusterNodepoolOperationLog(
	ctx context.Context,
	cs *khifilev6.TimelineChangeSet,
	tracker *GCPOperationTracker,
	targetTimeline *khifilev6.TimelinePath,
	operationTimeline *khifilev6.TimelinePath,
	audit *GCPAuditLogFieldSet,
	common *log.CommonFieldSet,
	shortMethodName string,
	isCluster bool,
) {
	if audit.ImmediateOperation() {
		cs.AddEvent(targetTimeline)
		return
	}

	resourceBodyField := "nodePool"
	if isCluster {
		resourceBodyField = "cluster"
	}

	switch shortMethodName {
	case "CreateCluster", "CreateNodePool", "EnrollCluster", "EnrollNodePool":
		var bodyNode structured.Node
		if audit.Request != nil {
			if subReader, err := audit.Request.GetReader(resourceBodyField); err == nil {
				bodyNode = subReader.Node
			}
		}

		if audit.Ending() && !tracker.HasStarted(audit.OperationID) {
			var stateLogNotFound *pb.RevisionState
			if isCluster {
				stateLogNotFound = commonlogk8saudit_contract.RevisionStateK8sClusterProvisioningLogNotFound
			} else {
				stateLogNotFound = commonlogk8saudit_contract.RevisionStateK8sNodepoolProvisioningLogNotFound
			}
			cs.AddRevision(targetTimeline, &khifilev6.StagingRevision{
				VerbType:     commonlogk8saudit_contract.VerbCreate,
				StateType:    stateLogNotFound,
				Principal:    audit.PrincipalEmail,
				ChangedTime:  time.Unix(0, 0),
				ResourceBody: nil,
			})
			tracker.MarkResourceRevision(targetTimeline)
		}

		var state *pb.RevisionState
		if isCluster {
			if audit.Ending() {
				state = commonlogk8saudit_contract.RevisionStateK8sClusterExisting
			} else {
				state = commonlogk8saudit_contract.RevisionStateK8sClusterProvisioning
			}
		} else {
			if audit.Ending() {
				state = commonlogk8saudit_contract.RevisionStateK8sNodepoolExisting
			} else {
				state = commonlogk8saudit_contract.RevisionStateK8sNodepoolProvisioning
			}
		}

		cs.AddRevision(targetTimeline, &khifilev6.StagingRevision{
			VerbType:     commonlogk8saudit_contract.VerbCreate,
			StateType:    state,
			Principal:    audit.PrincipalEmail,
			ChangedTime:  common.Timestamp,
			ResourceBody: bodyNode,
		})
		tracker.MarkResourceRevision(targetTimeline)

	case "DeleteCluster", "DeleteNodePool", "UnenrollCluster", "UnenrollNodePool":
		if !audit.Ending() && !tracker.HasResourceRevision(targetTimeline) {
			var stateExistingNotFound *pb.RevisionState
			if isCluster {
				stateExistingNotFound = commonlogk8saudit_contract.RevisionStateK8sClusterExistingLogNotFound
			} else {
				stateExistingNotFound = commonlogk8saudit_contract.RevisionStateK8sNodepoolExistingLogNotFound
			}
			cs.AddRevision(targetTimeline, &khifilev6.StagingRevision{
				VerbType:     commonlogk8saudit_contract.VerbCreate,
				StateType:    stateExistingNotFound,
				Principal:    audit.PrincipalEmail,
				ChangedTime:  time.Unix(0, 0),
				ResourceBody: nil,
			})
			tracker.MarkResourceRevision(targetTimeline)
		}

		if audit.Ending() && !tracker.HasStarted(audit.OperationID) {
			var stateDeletingNotFound *pb.RevisionState
			if isCluster {
				stateDeletingNotFound = commonlogk8saudit_contract.RevisionStateK8sClusterDeletingLogNotFound
			} else {
				stateDeletingNotFound = commonlogk8saudit_contract.RevisionStateK8sNodepoolDeletingLogNotFound
			}
			cs.AddRevision(targetTimeline, &khifilev6.StagingRevision{
				VerbType:     commonlogk8saudit_contract.VerbDelete,
				StateType:    stateDeletingNotFound,
				Principal:    audit.PrincipalEmail,
				ChangedTime:  time.Unix(0, 0),
				ResourceBody: nil,
			})
			tracker.MarkResourceRevision(targetTimeline)
		}

		var state *pb.RevisionState
		if isCluster {
			if audit.Ending() {
				state = commonlogk8saudit_contract.RevisionStateK8sClusterDeleted
			} else {
				state = commonlogk8saudit_contract.RevisionStateK8sClusterDeleting
			}
		} else {
			if audit.Ending() {
				state = commonlogk8saudit_contract.RevisionStateK8sNodepoolDeleted
			} else {
				state = commonlogk8saudit_contract.RevisionStateK8sNodepoolDeleting
			}
		}
		cs.AddRevision(targetTimeline, &khifilev6.StagingRevision{
			VerbType:     commonlogk8saudit_contract.VerbDelete,
			StateType:    state,
			Principal:    audit.PrincipalEmail,
			ChangedTime:  common.Timestamp,
			ResourceBody: nil,
		})
		tracker.MarkResourceRevision(targetTimeline)
	}

	tracker.ProcessOperationLog(ctx, cs, operationTimeline, audit, common.Timestamp)
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
