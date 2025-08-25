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

package googlecloudlogk8saudit_impl

import (
	"context"

	commonlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/impl/recorder"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/impl/recorder/bindingrecorder"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/impl/recorder/commonrecorder"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/impl/recorder/containerstatusrecorder"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/impl/recorder/endpointslicerecorder"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/impl/recorder/noderecorder"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/impl/recorder/ownerreferencerecorder"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/impl/recorder/snegrecorder"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/commonlogk8saudit/impl/recorder/statusrecorder"
	googlecloudinspectiontypegroup_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudinspectiontypegroup/contract"
	googlecloudlogk8saudit_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8saudit/contract"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8saudit/impl/fieldextractor"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
)

// GCPK8sAuditLogSourceTask creates an AuditLogParserLogSource for GCP Kubernetes audit logs.
// It retrieves the logs from the K8sAuditQueryTask and provides them along with a
// GCP-specific field extractor to downstream parsing tasks.
var GCPK8sAuditLogSourceTask = inspectiontaskbase.NewInspectionTask(googlecloudlogk8saudit_contract.GKEK8sAuditLogSourceTaskID, []taskid.UntypedTaskReference{
	googlecloudlogk8saudit_contract.K8sAuditQueryTaskID.Ref(),
}, func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (*commonlogk8saudit_contract.AuditLogParserLogSource, error) {
	if taskMode == inspectioncore_contract.TaskModeDryRun {
		return nil, nil
	}
	logs := coretask.GetTaskResult(ctx, googlecloudlogk8saudit_contract.K8sAuditQueryTaskID.Ref())

	return &commonlogk8saudit_contract.AuditLogParserLogSource{
		Logs:      logs,
		Extractor: &fieldextractor.GCPAuditLogFieldExtractor{},
	}, nil
}, inspectioncore_contract.InspectionTypeLabel(googlecloudinspectiontypegroup_contract.GCPK8sClusterInspectionTypes...))

// RegisterK8sAuditTasks registers all the tasks required for parsing GKE Kubernetes audit logs.
// This includes the common audit log recorders as well as GKE-specific ones like the SNEG recorder.
var RegisterK8sAuditTasks coreinspection.InspectionRegistrationFunc = func(registry coreinspection.InspectionTaskRegistry) error {
	err := registry.AddTask(GCPK8sAuditLogSourceTask)
	if err != nil {
		return err
	}

	manager := recorder.NewAuditRecorderTaskManager(googlecloudlogk8saudit_contract.K8sAuditParseTaskID, "gke")
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

	err = manager.Register(registry, googlecloudinspectiontypegroup_contract.GCPK8sClusterInspectionTypes...)
	if err != nil {
		return err
	}
	return nil
}
