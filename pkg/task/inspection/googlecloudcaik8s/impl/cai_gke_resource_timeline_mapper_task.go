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

// gkeTimelineMapperState tracks whether the initial snapshot of a log group has been processed.
type gkeTimelineMapperState struct {
	hasProcessedInitialSnapshot bool
}

// caiGKEResourceTimelineMapper maps CAI GKE cluster and nodepool snapshot logs to timeline revisions.
type caiGKEResourceTimelineMapper struct {
	inspectiontaskbase.SinglePassMapperBase[gkeTimelineMapperState]
}

var _ inspectiontaskbase.LogToTimelineMapper[gkeTimelineMapperState] = (*caiGKEResourceTimelineMapper)(nil)

// LogIngesterTask returns the prerequisite log ingester task reference.
func (m *caiGKEResourceTimelineMapper) LogIngesterTask() taskid.TaskReference[struct{}] {
	return googlecloudcaik8s_contract.GKELogIngesterTaskID.Ref()
}

// GroupedLogTask returns the reference to the task providing grouped CAI GKE logs.
func (m *caiGKEResourceTimelineMapper) GroupedLogTask() taskid.TaskReference[inspectiontaskbase.LogGroupMap] {
	return googlecloudcaik8s_contract.GKELogGrouperTaskID.Ref()
}

// Dependencies returns additional task dependencies for timeline mapping.
func (m *caiGKEResourceTimelineMapper) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{
		googlecloudk8scommon_contract.ClusterIdentityTaskID.Ref(),
		googlecloudcommon_contract.InputStartTimeTaskID.Ref(),
		googlecloudcommon_contract.InputEndTimeTaskID.Ref(),
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
func (m *caiGKEResourceTimelineMapper) ProcessLogByGroup(ctx context.Context, l *log.Log, state gkeTimelineMapperState) (*khifilev6.TimelineChangeSet, gkeTimelineMapperState, error) {
	assetName := l.NodeReader.ReadStringOrDefault(pathAssetName, "")
	identity := parseGKEAssetName(assetName)
	if identity.ClusterName == "" && identity.NodePoolName == "" {
		return nil, state, nil
	}

	assetWindowStartTime, assetWindowEndTime, isDeleted := extractTimeWindow(l.NodeReader)
	if isDeleted {
		return nil, state, nil
	}

	queryStartTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputStartTimeTaskID.Ref())
	queryEndTime := coretask.GetTaskResult(ctx, googlecloudcommon_contract.InputEndTimeTaskID.Ref())

	if !assetWindowEndTime.IsZero() && !assetWindowEndTime.After(queryStartTime) {
		return nil, state, nil
	}
	if !assetWindowStartTime.IsZero() && assetWindowStartTime.After(queryEndTime) {
		return nil, state, nil
	}

	clusterIdentity := coretask.GetTaskResult(ctx, googlecloudk8scommon_contract.ClusterIdentityTaskID.Ref())
	projectTimeline := googlecloudcommon_contract.MustGCPProjectTimeline(ctx, clusterIdentity.ProjectID)
	clusterTimeline := googlecloudcommon_contract.MustGKEClusterTimeline(ctx, projectTimeline, clusterIdentity.ClusterName)

	var targetTimeline *khifilev6.TimelinePath
	if identity.IsNodePool() {
		targetTimeline = googlecloudcommon_contract.MustGKENodePoolTimeline(ctx, clusterTimeline, identity.NodePoolName)
	} else {
		targetTimeline = clusterTimeline
	}

	cs := khifilev6.NewTimelineChangeSet(l)
	resourceBody := extractResourceBody(l.NodeReader)

	observedTime := assetWindowStartTime
	if observedTime.IsZero() || observedTime.Before(queryStartTime) {
		observedTime = queryStartTime
	}

	if !state.hasProcessedInitialSnapshot {
		state.hasProcessedInitialSnapshot = true

		if !identity.IsNodePool() {
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
				StateType:    googlecloudcaik8s_contract.RevisionStateGKEClusterExistingFromCAI,
			})
		} else {
			snapshots := coretask.GetTaskResult(ctx, googlecloudcaik8s_contract.GKEResourceFetcherTaskID.Ref())
			clusterCreateTime := extractClusterCreateTimeFromSnapshots(snapshots)
			earliestWindowStartTime := assetWindowStartTime

			switch {
			case !clusterCreateTime.IsZero() && !earliestWindowStartTime.IsZero() && earliestWindowStartTime.Sub(clusterCreateTime) >= creationTimestampSkewTolerance:
				// Stage 1: Undetermined existence between cluster creation and earliest recorded snapshot.
				cs.AddRevision(targetTimeline, &khifilev6.StagingRevision{
					ChangedTime:  clusterCreateTime,
					ResourceBody: nil,
					Principal:    "N/A",
					VerbType:     commonlogk8saudit_contract.VerbCreate,
					StateType:    googlecloudcaik8s_contract.RevisionStateGKENodePoolExistenceUndetermined,
				})

				// Stage 2: Confirmed existence from earliest snapshot to query start time.
				if earliestWindowStartTime.Before(queryStartTime) {
					cs.AddRevision(targetTimeline, &khifilev6.StagingRevision{
						ChangedTime:  earliestWindowStartTime,
						ResourceBody: nil,
						Principal:    "N/A",
						VerbType:     commonlogk8saudit_contract.VerbUpdate,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sNodepoolExistingLogNotFound,
					})
				}

				// Stage 3: Initial manifest at query start time.
				cs.AddRevision(targetTimeline, &khifilev6.StagingRevision{
					ChangedTime:  observedTime,
					ResourceBody: resourceBody,
					Principal:    "N/A",
					VerbType:     commonlogk8saudit_contract.VerbUpdate,
					StateType:    googlecloudcaik8s_contract.RevisionStateGKENodePoolExistingFromCAI,
				})
			case !clusterCreateTime.IsZero() && (earliestWindowStartTime.IsZero() || earliestWindowStartTime.Sub(clusterCreateTime) < creationTimestampSkewTolerance):
				// NodePool created alongside the cluster (within skew tolerance).
				if observedTime.Sub(clusterCreateTime) >= creationTimestampSkewTolerance {
					cs.AddRevision(targetTimeline, &khifilev6.StagingRevision{
						ChangedTime:  clusterCreateTime,
						ResourceBody: nil,
						Principal:    "N/A",
						VerbType:     commonlogk8saudit_contract.VerbCreate,
						StateType:    commonlogk8saudit_contract.RevisionStateK8sNodepoolExistingLogNotFound,
					})
					cs.AddRevision(targetTimeline, &khifilev6.StagingRevision{
						ChangedTime:  observedTime,
						ResourceBody: resourceBody,
						Principal:    "N/A",
						VerbType:     commonlogk8saudit_contract.VerbUpdate,
						StateType:    googlecloudcaik8s_contract.RevisionStateGKENodePoolExistingFromCAI,
					})
				} else {
					cs.AddRevision(targetTimeline, &khifilev6.StagingRevision{
						ChangedTime:  observedTime,
						ResourceBody: resourceBody,
						Principal:    "N/A",
						VerbType:     commonlogk8saudit_contract.VerbCreate,
						StateType:    googlecloudcaik8s_contract.RevisionStateGKENodePoolExistingFromCAI,
					})
				}
			default:
				// Fallback when cluster creation time is not available.
				cs.AddRevision(targetTimeline, &khifilev6.StagingRevision{
					ChangedTime:  observedTime,
					ResourceBody: resourceBody,
					Principal:    "N/A",
					VerbType:     commonlogk8saudit_contract.VerbCreate,
					StateType:    googlecloudcaik8s_contract.RevisionStateGKENodePoolExistingFromCAI,
				})
			}
		}

		return cs, state, nil
	}

	// Secondary updates within the inspection time window.
	updateTime := assetWindowStartTime
	if updateTime.IsZero() {
		updateTime = l.Timestamp
	}

	stateType := googlecloudcaik8s_contract.RevisionStateGKEClusterExistingFromCAI
	if identity.IsNodePool() {
		stateType = googlecloudcaik8s_contract.RevisionStateGKENodePoolExistingFromCAI
	}

	cs.AddRevision(targetTimeline, &khifilev6.StagingRevision{
		ChangedTime:  updateTime,
		ResourceBody: resourceBody,
		Principal:    "N/A",
		VerbType:     commonlogk8saudit_contract.VerbUpdate,
		StateType:    stateType,
	})

	return cs, state, nil
}

// GKELogToTimelineMapperTask is the task that maps CAI GKE snapshots to timeline revisions.
var GKELogToTimelineMapperTask = inspectiontaskbase.NewLogToTimelineMapperTask(
	googlecloudcaik8s_contract.GKELogToTimelineMapperTaskID,
	&caiGKEResourceTimelineMapper{},
)
