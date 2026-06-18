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
	inspectioncore_contract.FeatureTaskLabelV2(
		"Kubernetes Control Plane Component Logs",
		"Gather logs from Kubernetes control plane components (e.g., kube-scheduler, kube-controller-manager, and kube-apiserver) to troubleshoot control plane behavior.",
		9000,
		false,
	),
)

// K8sControlPlaneLogIngester is a log ingester for Kubernetes control plane component logs.
type K8sControlPlaneLogIngester struct{}

// RawLogTask implements inspectiontaskbase.LogIngesterV2.
func (i *K8sControlPlaneLogIngester) RawLogTask() taskid.TaskReference[[]*log.Log] {
	return googlecloudlogk8scontrolplane_contract.CommonFieldSetReaderTaskID.Ref()
}

// Dependencies implements inspectiontaskbase.LogIngesterV2.
func (i *K8sControlPlaneLogIngester) Dependencies() []taskid.UntypedTaskReference {
	return []taskid.UntypedTaskReference{}
}

// ProcessLog implements inspectiontaskbase.LogIngesterV2.
func (i *K8sControlPlaneLogIngester) ProcessLog(ctx context.Context, l *log.Log) (*khifilev6.LogChangeSet, error) {
	cs, err := khifilev6.NewLogChangeSet(l)
	if err != nil {
		return nil, err
	}
	cs.SetLogType(googlecloudlogk8scontrolplane_contract.LogTypeControlPlaneComponent)

	if commonFS, err := log.GetFieldSet(l, &log.CommonFieldSet{}); err == nil {
		cs.SetTimestamp(commonFS.Timestamp)
	}

	if severityFS, err := log.GetFieldSet(l, &inspectioncore_contract.DefaultSeverityFieldSet{}); err == nil {
		cs.SetSeverity(severityFS.Severity)
	}

	if msgFS, err := log.GetFieldSet(l, &googlecloudlogk8scontrolplane_contract.K8sControlplaneCommonMessageFieldSet{}); err == nil {
		cs.SetSummary(msgFS.Message)
	}

	return cs, nil
}

var _ inspectiontaskbase.LogIngesterV2 = (*K8sControlPlaneLogIngester)(nil)

// LogIngesterTask serializes logs to history for timeline mappers to associate event or revisions in later tasks.
var LogIngesterTask = inspectiontaskbase.NewLogIngesterTaskV2(googlecloudlogk8scontrolplane_contract.LogIngesterTaskID, &K8sControlPlaneLogIngester{})
