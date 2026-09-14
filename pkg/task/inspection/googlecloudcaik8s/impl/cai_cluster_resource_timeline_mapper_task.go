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

package googlecloudcaik8s_impl

import (
	"context"
	"time"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	googlecloudcaik8s_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcaik8s/contract"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudk8scommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/contract"
)

// caiClusterResourceTimelineMapper maps CAI cluster resource snapshot logs to timeline revisions.
type caiClusterResourceTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
}

var _ inspectiontaskbase.LogToTimelineMapper[struct{}] = (*caiClusterResourceTimelineMapper)(nil)

// LogIngesterTask returns the prerequisite log ingester task reference.
func (m *caiClusterResourceTimelineMapper) LogIngesterTask() taskid.TaskReference[struct{}] {
	return googlecloudcaik8s_contract.LogIngesterTaskID.Ref()
}

// GroupedLogTask returns the reference to the task providing grouped CAI logs.
func (m *caiClusterResourceTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudcaik8s_contract.LogGrouperTaskID.Ref()
}

// Dependencies returns additional task dependencies for timeline mapping.
func (m *caiClusterResourceTimelineMapper) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudk8scommon_contract.ClusterIdentityTaskID.Ref(),
		googlecloudcommon_contract.InputStartTimeTaskID.Ref(),
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

	queryStartTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputStartTimeTaskID.Ref())
	if !isActiveAt(assetWindowStartTime, assetWindowEndTime, queryStartTime) {
		return nil, struct{}{}, nil
	}

	identity := extractResourceIdentityFromLog(l.NodeReader)
	if identity.Kind == "" || identity.Name == "" {
		return nil, struct{}{}, nil
	}
	cluster := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.ClusterIdentityTaskID.Ref())
	targetPath := commonlogk8saudit_contract.MustResourceTimeline(ctx, cluster.ClusterName, identity)

	cs := khifilev6.NewTimelineChangeSet(l)
	resourceBody := extractResourceBody(l.NodeReader)

	observedTime := assetWindowStartTime
	if observedTime.IsZero() {
		// The asset history did not report when this version became current, so the revision falls back
		// to the inspection start time where the snapshot is known to be valid.
		observedTime = queryStartTime
	}

	snapshotVerb := commonlogk8saudit_contract.VerbCreate
	// The content between the creation and the observed manifest is unknown, so it is rendered as a
	// body-less revision the same way resources without any log are rendered.
	if creationTime, found := extractCreationTimestamp(l.NodeReader); found && observedTime.Sub(creationTime) >= creationTimestampSkewTolerance {
		cs.AddRevision(targetPath, &khifilev6.StagingRevision{
			ChangedTime:  creationTime,
			ResourceBody: nil,
			Principal:    "N/A",
			VerbType:     commonlogk8saudit_contract.VerbCreate,
			StateType:    commonlogk8saudit_contract.RevisionStateK8sResourceExistingLogNotFound,
		})
		snapshotVerb = commonlogk8saudit_contract.VerbUpdate
	}

	cs.AddRevision(targetPath, &khifilev6.StagingRevision{
		ChangedTime:  observedTime,
		ResourceBody: resourceBody,
		Principal:    "N/A",
		VerbType:     snapshotVerb,
		StateType:    googlecloudcaik8s_contract.RevisionStateK8sResourceExistingFromCAI,
	})

	return cs, struct{}{}, nil
}

// LogToTimelineMapperTask is the task that maps CAI cluster resource snapshots to timeline revisions.
var LogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask(
	googlecloudcaik8s_contract.LogToTimelineMapperTaskID,
	&caiClusterResourceTimelineMapper{},
)
