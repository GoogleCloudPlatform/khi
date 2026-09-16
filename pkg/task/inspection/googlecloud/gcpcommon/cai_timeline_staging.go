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

package gcpcommon

import (
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
)

// CAICreationTimestampSkewTolerance is the gap below which the period between creation time and
// the observed snapshot is not rendered as a separate body-less revision.
const CAICreationTimestampSkewTolerance = time.Second

// CAIInitialSnapshotRevisionSpec defines how a CAI snapshot active at queryStartTime should be staged on a timeline.
type CAIInitialSnapshotRevisionSpec struct {
	TargetTimeline    *khifilev6.TimelinePath
	CreationTime      time.Time
	ObservedTime      time.Time
	ResourceBody      structured.Node
	CreationStateType *pb.RevisionState
	SnapshotStateType *pb.RevisionState
}

// StageCAIInitialSnapshotRevisions stages the initial snapshot revisions on cs according to spec.
// If CreationTime is non-zero and ObservedTime - CreationTime >= CAICreationTimestampSkewTolerance,
// it stages a body-less VerbCreate revision at CreationTime followed by a VerbUpdate revision at ObservedTime.
// Otherwise, it stages a single VerbCreate revision at ObservedTime with ResourceBody.
func StageCAIInitialSnapshotRevisions(cs *khifilev6.TimelineChangeSet, spec CAIInitialSnapshotRevisionSpec) {
	snapshotVerb := k8saudit.VerbCreate
	if !spec.CreationTime.IsZero() && spec.ObservedTime.Sub(spec.CreationTime) >= CAICreationTimestampSkewTolerance {
		cs.AddRevision(spec.TargetTimeline, &khifilev6.StagingRevision{
			ChangedTime:  spec.CreationTime,
			ResourceBody: nil,
			Principal:    "N/A",
			VerbType:     k8saudit.VerbCreate,
			StateType:    spec.CreationStateType,
		})
		snapshotVerb = k8saudit.VerbUpdate
	}

	cs.AddRevision(spec.TargetTimeline, &khifilev6.StagingRevision{
		ChangedTime:  spec.ObservedTime,
		ResourceBody: spec.ResourceBody,
		Principal:    "N/A",
		VerbType:     snapshotVerb,
		StateType:    spec.SnapshotStateType,
	})
}
