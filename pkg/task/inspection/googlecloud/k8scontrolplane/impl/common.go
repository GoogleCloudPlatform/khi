// Copyright 2024 Google LLC
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

package k8scontrolplane_impl

import (
	"context"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/k8scontrolplane"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

var TailTask = coretask.NewTailTask(
	k8scontrolplane.TailTaskID,
	[]coretask.Dependency{
		k8scontrolplane.SchedulerLogToTimelineMapperTaskID.Ref(),
		k8scontrolplane.ControllerManagerLogToTimelineMapperTaskID.Ref(),
		k8scontrolplane.HpaControllerLogToTimelineMapperTaskID.Ref(),
		k8scontrolplane.OtherLogToTimelineMapperTaskID.Ref(),
	},
	inspectioncore.FeatureTaskLabel(
		"Kubernetes Control Plane Component Logs",
		"Gather logs from Kubernetes control plane components (e.g., kube-scheduler, kube-controller-manager, and kube-apiserver) to troubleshoot control plane behavior.",
		9000,
		false,
	),
)

// K8sControlPlaneLogIngester is a log ingester for Kubernetes control plane component logs.
type K8sControlPlaneLogIngester struct{}

// RawLogTask implements inspectiontaskbase.LogIngester.
func (i *K8sControlPlaneLogIngester) RawLogTask() taskid.TaskReference[[]*log.Log] {
	return k8scontrolplane.ListLogEntriesTaskID.Ref()
}

// Dependencies implements inspectiontaskbase.LogIngester.
func (i *K8sControlPlaneLogIngester) Dependencies() []coretask.Dependency {
	return []coretask.Dependency{}
}

// ProcessLog implements inspectiontaskbase.LogIngester.
func (i *K8sControlPlaneLogIngester) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}
	cs.SetLogType(k8scontrolplane.LogTypeControlPlaneComponent)
	cs.SetTimestamp(l.Timestamp)

	if severity, err := gcpcommon.ExtractGCPSeverity(l.NodeReader); err == nil && severity != nil {
		cs.SetSeverity(severity)
	}

	componentFieldSet, err := k8scontrolplane.ExtractK8sControlplaneComponent(l.NodeReader)
	if err == nil && componentFieldSet.ComponentParserType() == k8scontrolplane.ComponentParserTypeHPAController {
		if hpaFS, err := k8scontrolplane.ExtractK8sHPAControllerComponent(l.NodeReader); err == nil {
			if summary := hpaFS.Summary(); summary != "" {
				cs.SetSummary(summary)
			}
		}
	} else {
		if msg, err := k8scontrolplane.ExtractK8sControlplaneCommonMessage(l.NodeReader); err == nil && msg != "" {
			cs.SetSummary(msg)
		}
	}

	return cs, nil
}

var _ inspectiontaskbase.LogIngester = (*K8sControlPlaneLogIngester)(nil)

// LogIngesterTask serializes logs to history for timeline mappers to associate event or revisions in later tasks.
var LogIngesterTask = inspectiontaskbase.NewLogIngesterTask(k8scontrolplane.LogIngesterTaskID, &K8sControlPlaneLogIngester{})
