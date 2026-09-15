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

package caik8s_impl

import (
	"context"
	"time"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/caik8s"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
)

// caiClusterResourceTimelineMapper maps CAI cluster resource snapshot logs to timeline revisions.
type caiClusterResourceTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
}

var _ inspectiontaskbase.LogToTimelineMapper[struct{}] = (*caiClusterResourceTimelineMapper)(nil)

// LogIngesterTask returns the prerequisite log ingester task reference.
func (m *caiClusterResourceTimelineMapper) LogIngesterTask() taskid.TaskReference[struct{}] {
	return caik8s.LogIngesterTaskID.Ref()
}

// GroupedLogTask returns the reference to the task providing grouped CAI logs.
func (m *caiClusterResourceTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return caik8s.LogGrouperTaskID.Ref()
}

// Dependencies returns additional task dependencies for timeline mapping.
func (m *caiClusterResourceTimelineMapper) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{
		k8scommon.ClusterIdentityTaskID.Ref(),
		gcpcommon.InputStartTimeTaskID.Ref(),
	}
}

// creationTimestampSkewTolerance is the gap below which the period between metadata.creationTimestamp
// and the observed manifest is not rendered. metadata.creationTimestamp is truncated to seconds and
// the inventory records a version slightly after the cluster applies it, so a small gap carries no
// information about an unobserved state.
const creationTimestampSkewTolerance = time.Second

// ProcessLogByGroup processes a log entry and stages a timeline revision for existing resources.
func (m *caiClusterResourceTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	assetWindowStartTime, assetWindowEndTime, isDeleted := extractTimeWindow(l.NodeReader)
	if isDeleted {
		return nil, struct{}{}, nil
	}

	queryStartTime := coretask.GetTaskResult(ctx, gcpcommon.InputStartTimeTaskID.Ref())
	if !isActiveAt(assetWindowStartTime, assetWindowEndTime, queryStartTime) {
		return nil, struct{}{}, nil
	}

	identity := extractResourceIdentityFromLog(l.NodeReader)
	if identity.Kind == "" || identity.Name == "" {
		return nil, struct{}{}, nil
	}
	cluster := coretask.GetTaskResult(ctx, k8scommon.ClusterIdentityTaskID.Ref())
	targetPath := k8saudit.MustResourceTimeline(ctx, cluster.ClusterName, identity)

	cs := khifilev6.NewTimelineChangeSet(l)
	resourceBody := extractResourceBody(l.NodeReader)

	observedTime := assetWindowStartTime
	if observedTime.IsZero() {
		// The asset history did not report when this version became current, so the revision falls back
		// to the inspection start time where the snapshot is known to be valid.
		observedTime = queryStartTime
	}

	snapshotVerb := k8saudit.VerbCreate
	// The content between the creation and the observed manifest is unknown, so it is rendered as a
	// body-less revision the same way resources without any log are rendered.
	if creationTime, found := extractCreationTimestamp(l.NodeReader); found && observedTime.Sub(creationTime) >= creationTimestampSkewTolerance {
		cs.AddRevision(targetPath, &khifilev6.StagingRevision{
			ChangedTime:  creationTime,
			ResourceBody: nil,
			Principal:    "N/A",
			VerbType:     k8saudit.VerbCreate,
			StateType:    k8saudit.RevisionStateK8sResourceExistingLogNotFound,
		})
		snapshotVerb = k8saudit.VerbUpdate
	}

	cs.AddRevision(targetPath, &khifilev6.StagingRevision{
		ChangedTime:  observedTime,
		ResourceBody: resourceBody,
		Principal:    "N/A",
		VerbType:     snapshotVerb,
		StateType:    caik8s.RevisionStateK8sResourceExistingFromCAI,
	})

	return cs, struct{}{}, nil
}

// LogToTimelineMapperTask is the task that maps CAI cluster resource snapshots to timeline revisions.
var LogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask(
	caik8s.LogToTimelineMapperTaskID,
	&caiClusterResourceTimelineMapper{},
)
