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

package googlecloudlogk8scontrolplane_impl

import (
	"context"

	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	khifilev6 "github.com/GoogleCloudPlatform/khi/pkg/model/khifile/v6"
	"github.com/GoogleCloudPlatform/khi/pkg/model/log"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	googlecloudlogk8scontrolplane_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8scontrolplane/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

var TailTask = inspectiontaskbase.NewInspectionTask(googlecloudlogk8scontrolplane_contract.TailTaskID,
	[]taskid.UntypedTaskReference{
		googlecloudlogk8scontrolplane_contract.SchedulerLogToTimelineMapperTaskID.Ref(),
		googlecloudlogk8scontrolplane_contract.ControllerManagerLogToTimelineMapperTaskID.Ref(),
		googlecloudlogk8scontrolplane_contract.OtherLogToTimelineMapperTaskID.Ref(),
	},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (struct{}, error) {
		return struct{}{}, nil
	},
	inspectioncore_contract.FeatureTaskLabel(
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
	return googlecloudlogk8scontrolplane_contract.ListLogEntriesTaskID.Ref()
}

// Dependencies implements inspectiontaskbase.LogIngester.
func (i *K8sControlPlaneLogIngester) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// ProcessLog implements inspectiontaskbase.LogIngester.
func (i *K8sControlPlaneLogIngester) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}
	cs.SetLogType(googlecloudlogk8scontrolplane_contract.LogTypeControlPlaneComponent)
	cs.SetTimestamp(l.Timestamp)

	if severity, err := googlecloudcommon_contract.ExtractGCPSeverity(l.NodeReader); err == nil && severity != nil {
		cs.SetSeverity(severity)
	}

	if msg, err := googlecloudlogk8scontrolplane_contract.ExtractK8sControlplaneCommonMessage(l.NodeReader); err == nil && msg != "" {
		cs.SetSummary(msg)
	}

	return cs, nil
}

var _ inspectiontaskbase.LogIngester = (*K8sControlPlaneLogIngester)(nil)

// LogIngesterTask serializes logs to history for timeline mappers to associate event or revisions in later tasks.
var LogIngesterTask = inspectiontaskbase.NewLogIngesterTask(googlecloudlogk8scontrolplane_contract.LogIngesterTaskID, &K8sControlPlaneLogIngester{})
