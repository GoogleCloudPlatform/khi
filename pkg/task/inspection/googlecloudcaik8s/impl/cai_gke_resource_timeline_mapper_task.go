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

	"github.com/GoogleCloudPlatform/khi/pkg/common"
	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
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

var (
	pathClusterCreateTime = structured.CompileFieldPath("asset.resource.data.createTime")
)

// caiGKEResourceTimelineMapper maps CAI GKE cluster and nodepool snapshot logs to timeline revisions.
type caiGKEResourceTimelineMapper struct {
	inspectiontaskbase.StatelessMapperBase
}

var _ inspectiontaskbase.LogToTimelineMapper[struct{}] = (*caiGKEResourceTimelineMapper)(nil)

// LogIngesterTask returns the prerequisite log ingester task reference.
func (m *caiGKEResourceTimelineMapper) LogIngesterTask() taskid.TaskReference[struct{}] {
	return googlecloudcaik8s_contract.GKELogIngesterTaskID.Ref()
}

// GroupedLogTask returns the reference to the task providing grouped CAI GKE logs.
func (m *caiGKEResourceTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudcaik8s_contract.GKELogGrouperTaskID.Ref()
}

// Dependencies returns additional task dependencies for timeline mapping.
func (m *caiGKEResourceTimelineMapper) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{
		googlecloudk8scommon_contract.ClusterIdentityTaskID.Ref(),
		googlecloudcommon_contract.InputStartTimeTaskID.Ref(),
		googlecloudcaik8s_contract.GKEResourceFetcherTaskID.Ref(),
	}
}

// extractClusterCreateTimeFromSnapshots extracts the creation timestamp of the GKE Cluster from snapshots.
func extractClusterCreateTimeFromSnapshots(snapshots []*googlecloudcaik8s_contract.GKEResourceSnapshot) time.Time {
	for _, s := range snapshots {
		if s == nil || s.TemporalAsset == nil || s.TemporalAsset.Asset == nil {
			continue
		}
		if s.TemporalAsset.Asset.AssetType != googlecloudcaik8s_contract.GKEClusterAssetType {
			continue
		}
		if res := s.TemporalAsset.Asset.Resource; res != nil && res.Data != nil {
			str := res.Data.GetFields()["createTime"].GetStringValue()
			if str != "" {
				if t, err := common.ParseTime(str); err == nil {
					return t
				}
			}
		}
	}
	return time.Time{}
}

// ProcessLogByGroup processes a log entry and stages a timeline revision for GKE cluster or nodepool.
func (m *caiGKEResourceTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, _ struct{}) (*khifilev6.TimelineChangeSet, struct{}, error) {
	assetName := l.NodeReader.ReadStringOrDefault(pathAssetName, "")
	identity := parseGKEAssetName(assetName)
	if identity.ClusterName == "" && identity.NodePoolName == "" {
		return nil, struct{}{}, nil
	}

	assetWindowStartTime, assetWindowEndTime, isDeleted := extractTimeWindow(l.NodeReader)
	if isDeleted {
		return nil, struct{}{}, nil
	}

	queryStartTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputStartTimeTaskID.Ref())
	if !isActiveAt(assetWindowStartTime, assetWindowEndTime, queryStartTime) {
		return nil, struct{}{}, nil
	}

	targetTimeline := m.resolveTargetTimeline(ctx, identity)
	cs := khifilev6.NewTimelineChangeSet(l)
	resourceBody := extractResourceBody(l.NodeReader)

	observedTime := assetWindowStartTime
	if observedTime.IsZero() {
		observedTime = queryStartTime
	}

	if identity.IsCluster() {
		m.stageClusterInitialRevisions(cs, targetTimeline, l, observedTime, resourceBody)
	} else {
		snapshots := coretask.GetTaskResult(ctx, googlecloudcaik8s_contract.GKEResourceFetcherTaskID.Ref())
		clusterCreateTime := extractClusterCreateTimeFromSnapshots(snapshots)
		m.stageNodePoolInitialRevisions(cs, targetTimeline, clusterCreateTime, observedTime, resourceBody)
	}
	return cs, struct{}{}, nil
}

func (m *caiGKEResourceTimelineMapper) resolveTargetTimeline(ctx context.Context, identity gkeResourceIdentity) *khifilev6.TimelinePath {
	clusterIdentity := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.ClusterIdentityTaskID.Ref())
	projectTimeline := googlecloudcommon_contract.MustGCPProjectTimeline(ctx, clusterIdentity.ProjectID)
	clusterTimeline := googlecloudcommon_contract.MustGKEClusterTimeline(ctx, projectTimeline, clusterIdentity.ClusterName)

	if identity.IsNodePool() {
		return googlecloudcommon_contract.MustGKENodePoolTimeline(ctx, clusterTimeline, identity.NodePoolName)
	}
	return clusterTimeline
}

func (m *caiGKEResourceTimelineMapper) stageClusterInitialRevisions(cs *khifilev6.TimelineChangeSet, targetTimeline *khifilev6.TimelinePath, l *log.Log, observedTime time.Time, resourceBody structured.Node) {
	clusterCreateTime := l.NodeReader.ReadTimestampOrDefault(pathClusterCreateTime, time.Time{})
	snapshotVerb := commonlogk8saudit_contract.VerbCreate
	if !clusterCreateTime.IsZero() && observedTime.Sub(clusterCreateTime) >= creationTimestampSkewTolerance {
		cs.AddRevision(targetTimeline, &khifilev6.StagingRevision{
			ChangedTime:  clusterCreateTime,
			ResourceBody: nil,
			Principal:    "N/A",
			VerbType:     commonlogk8saudit_contract.VerbCreate,
			StateType:    commonlogk8saudit_contract.RevisionStateK8sClusterExistingLogNotFound,
		})
		snapshotVerb = commonlogk8saudit_contract.VerbUpdate
	}
	cs.AddRevision(targetTimeline, &khifilev6.StagingRevision{
		ChangedTime:  observedTime,
		ResourceBody: resourceBody,
		Principal:    "N/A",
		VerbType:     snapshotVerb,
		StateType:    googlecloudcaik8s_contract.RevisionStateGKEClusterSnapshotFromCAI,
	})
}

func (m *caiGKEResourceTimelineMapper) stageNodePoolInitialRevisions(cs *khifilev6.TimelineChangeSet, targetTimeline *khifilev6.TimelinePath, clusterCreateTime, observedTime time.Time, resourceBody structured.Node) {
	snapshotVerb := commonlogk8saudit_contract.VerbCreate
	if !clusterCreateTime.IsZero() && observedTime.Sub(clusterCreateTime) >= creationTimestampSkewTolerance {
		cs.AddRevision(targetTimeline, &khifilev6.StagingRevision{
			ChangedTime:  clusterCreateTime,
			ResourceBody: nil,
			Principal:    "N/A",
			VerbType:     commonlogk8saudit_contract.VerbCreate,
			StateType:    googlecloudcaik8s_contract.RevisionStateGKENodePoolExistenceUndetermined,
		})
		snapshotVerb = commonlogk8saudit_contract.VerbUpdate
	}

	cs.AddRevision(targetTimeline, &khifilev6.StagingRevision{
		ChangedTime:  observedTime,
		ResourceBody: resourceBody,
		Principal:    "N/A",
		VerbType:     snapshotVerb,
		StateType:    googlecloudcaik8s_contract.RevisionStateGKENodePoolSnapshotFromCAI,
	})
}

// GKELogToTimelineMapperTask is the task that maps CAI GKE snapshots to timeline revisions.
var GKELogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask(
	googlecloudcaik8s_contract.GKELogToTimelineMapperTaskID,
	&caiGKEResourceTimelineMapper{},
)
