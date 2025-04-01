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

package k8s_audit

import (
	"context"

	"github.com/GoogleCloudPlatform/khi/pkg/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/metadata/progress"
	inspection_task "github.com/GoogleCloudPlatform/khi/pkg/inspection/task"
	"github.com/GoogleCloudPlatform/khi/pkg/log"
	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/constants"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/recorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/recorder/bindingrecorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/recorder/commonrecorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/recorder/containerstatusrecorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/recorder/endpointslicerecorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/recorder/noderecorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/recorder/ownerreferencerecorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/recorder/snegrecorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/recorder/statusrecorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/types"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/inspectiontype"
	gcp_k8s_audit_constants "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/constants"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/fieldextractor"

	"github.com/GoogleCloudPlatform/khi/pkg/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/taskid"
)

// GCPK8sAuditLogSourceTask receives logs generated from the previous tasks specific to OSS audit log parsing and inject dependencies specific to this OSS inspection type.
var GCPK8sAuditLogSourceTask = inspection_task.NewInspectionProcessor(taskid.NewTaskImplementationId(constants.CommonAuitLogSource+"#gcp").String(), []string{
	gcp_k8s_audit_constants.K8sAuditQueryTaskID,
}, func(ctx context.Context, taskMode int, v *task.VariableSet, progress *progress.TaskProgress) (any, error) {
	if taskMode == inspection_task.TaskModeDryRun {
		return struct{}{}, nil
	}
	logs, err := task.GetTypedVariableFromTaskVariable[[]*log.LogEntity](v, gcp_k8s_audit_constants.K8sAuditQueryTaskID, nil)
	if err != nil {
		return nil, err
	}
	return &types.AuditLogParserLogSource{
		Logs:      logs,
		Extractor: &fieldextractor.GCPAuditLogFieldExtractor{},
	}, nil
}, inspection_task.InspectionTypeLabel(inspectiontype.GCPK8sClusterInspectionTypes...))

var RegisterK8sAuditTasks inspection.PrepareInspectionServerFunc = func(inspectionServer *inspection.InspectionTaskServer) error {
	err := inspectionServer.AddTaskDefinition(GCPK8sAuditLogSourceTask)
	if err != nil {
		return err
	}

	manager := recorder.NewAuditRecorderTaskManager(taskid.NewTaskImplementationId(gcp_k8s_audit_constants.K8sAuditParseTaskID))
	err = commonrecorder.Register(manager)
	if err != nil {
		return err
	}
	err = statusrecorder.Register(manager)
	if err != nil {
		return err
	}
	err = bindingrecorder.Register(manager)
	if err != nil {
		return err
	}
	err = endpointslicerecorder.Register(manager)
	if err != nil {
		return err
	}
	err = ownerreferencerecorder.Register(manager)
	if err != nil {
		return err
	}
	err = containerstatusrecorder.Register(manager)
	if err != nil {
		return err
	}
	err = noderecorder.Register(manager)
	if err != nil {
		return err
	}

	// GKE specific resource
	err = snegrecorder.Register(manager)
	if err != nil {
		return err
	}

	err = manager.Register(inspectionServer, inspectiontype.GCPK8sClusterInspectionTypes, inspection_task.FeatureTaskLabel("Kubernetes Audit Log", `Gather kubernetes audit logs and visualize resource modifications.`, enum.LogTypeAudit, true))
	if err != nil {
		return err
	}
	return nil
}
