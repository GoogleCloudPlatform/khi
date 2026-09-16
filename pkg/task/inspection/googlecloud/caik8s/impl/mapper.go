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

	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/common/k8saudit"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/caik8s"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scommon"
)

func mapClusterResourceInitialRevision(ctx context.Context, l *log.Log, identity *k8saudit.ResourceIdentity, observedTime time.Time) (gcpcommon.CAIInitialSnapshotRevisionSpec, bool, error) {
	cluster := coretask.GetTaskResult(ctx, k8scommon.ClusterIdentityTaskID.Ref())
	targetPath := k8saudit.MustResourceTimeline(ctx, cluster.ClusterName, identity)
	creationTime := extractCreationTimestamp(l.NodeReader)

	return gcpcommon.CAIInitialSnapshotRevisionSpec{
		TargetTimeline:    targetPath,
		CreationTime:      creationTime,
		ObservedTime:      observedTime,
		ResourceBody:      extractK8sResourceBody(l.NodeReader),
		CreationStateType: k8saudit.RevisionStateK8sResourceExistingLogNotFound,
		SnapshotStateType: caik8s.RevisionStateK8sResourceExistingFromCAI,
	}, false, nil
}
