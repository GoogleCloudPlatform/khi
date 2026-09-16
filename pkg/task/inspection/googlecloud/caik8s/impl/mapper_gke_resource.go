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

	"github.com/GoogleCloudPlatform/khi/pkg/common/structured"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/caik8s"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
)

var (
	pathClusterCreateTime = structured.CompileFieldPath("asset.resource.data.createTime")
)

// extractClusterCreateTimeFromLogs extracts the creation timestamp of the GKE Cluster from raw logs.
func extractClusterCreateTimeFromLogs(logs []*log.Log) time.Time {
	for _, l := range logs {
		if gcpcommon.ExtractCAIAssetType(l.NodeReader) != caik8s.GKEClusterAssetType {
			continue
		}
		if t := l.NodeReader.ReadTimestampOrDefault(pathClusterCreateTime, time.Time{}); !t.IsZero() {
			return t
		}
	}
	return time.Time{}
}

func resolveGKETargetTimeline(ctx context.Context, identity gkeResourceIdentity) *khifilev6.TimelinePath {
	clusterIdentity := coretask.GetTaskResult(ctx, k8scommon.ClusterIdentityTaskID.Ref())
	projectTimeline := gcpcommon.MustGCPProjectTimeline(ctx, clusterIdentity.ProjectID)
	clusterTimeline := gcpcommon.MustGKEClusterTimeline(ctx, projectTimeline, clusterIdentity.ClusterName)

	if identity.IsNodePool() {
		return gcpcommon.MustGKENodePoolTimeline(ctx, clusterTimeline, identity.NodePoolName)
	}
	return clusterTimeline
}

func mapGKEResourceInitialRevision(ctx context.Context, l *log.Log, identity gkeResourceIdentity, observedTime time.Time) (gcpcommon.CAIInitialSnapshotRevisionSpec, bool, error) {
	targetTimeline := resolveGKETargetTimeline(ctx, identity)
	resourceBody := extractGKEResourceBody(l.NodeReader)

	if identity.IsCluster() {
		clusterCreateTime := l.NodeReader.ReadTimestampOrDefault(pathClusterCreateTime, time.Time{})
		return gcpcommon.CAIInitialSnapshotRevisionSpec{
			TargetTimeline:    targetTimeline,
			CreationTime:      clusterCreateTime,
			ObservedTime:      observedTime,
			ResourceBody:      resourceBody,
			CreationStateType: k8saudit.RevisionStateK8sClusterExistingLogNotFound,
			SnapshotStateType: caik8s.RevisionStateGKEClusterSnapshotFromCAI,
		}, false, nil
	}

	logs := coretask.GetTaskResult(ctx, caik8s.GKEResourceTaskIDs.RawLog.Ref())
	clusterCreateTime := extractClusterCreateTimeFromLogs(logs)
	return gcpcommon.CAIInitialSnapshotRevisionSpec{
		TargetTimeline:    targetTimeline,
		CreationTime:      clusterCreateTime,
		ObservedTime:      observedTime,
		ResourceBody:      resourceBody,
		CreationStateType: caik8s.RevisionStateGKENodePoolExistenceUndetermined,
		SnapshotStateType: caik8s.RevisionStateGKENodePoolSnapshotFromCAI,
	}, false, nil
}
