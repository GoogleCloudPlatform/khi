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

package gcp

import (
	"github.com/GoogleCloudPlatform/khi/pkg/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task"
	composer_task "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/cloud-composer"
	composer_form "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/cloud-composer/form"
	composer_inspection_type "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/cloud-composer/inspectiontype"
	composer_query "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/cloud-composer/query"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gcpcommon"
	baremetal "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gdcv-for-baremetal"
	vmware "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gdcv-for-vmware"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke"
	aws "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke-on-aws"
	azure "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke-on-azure"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/autoscaler"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/compute_api"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/gke_audit"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit"
	k8sauditquery "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/query"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_container"
	k8scontrolplanecomponent "github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_control_plane_component"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_event"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_node"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/network_api"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/serialport"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/multicloud_api"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/onprem_api"
)

func commonPreparation(inspectionServer *inspection.InspectionTaskServer) error {
	err := inspectionServer.AddTask(task.GCPDefaultK8sResourceMergeConfigTask)
	if err != nil {
		return err
	}

	err = inspectionServer.AddTask(gcpcommon.HeaderSuggestedFileNameTask)
	if err != nil {
		return err
	}

	err = inspectionServer.AddTask(gke.AutocompleteClusterNames)
	if err != nil {
		return err
	}

	err = inspectionServer.AddTask(aws.AutocompleteClusterNames)
	if err != nil {
		return err
	}

	err = inspectionServer.AddTask(azure.AutocompleteClusterNames)
	if err != nil {
		return err
	}

	err = inspectionServer.AddTask(baremetal.AutocompleteClusterNames)
	if err != nil {
		return err
	}

	err = inspectionServer.AddTask(vmware.AutocompleteClusterNames)
	if err != nil {
		return err
	}

	err = inspectionServer.AddTask(task.AutocompleteLocationTask)
	if err != nil {
		return nil
	}

	// Form input related tasks
	err = inspectionServer.AddTask(task.TimeZoneShiftInputTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(task.InputProjectIdTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(task.InputClusterNameTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(task.InputDurationTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(task.InputEndTimeTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(task.InputStartTimeTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(task.InputKindFilterTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(task.InputLocationsTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(task.InputNamespaceFilterTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(task.InputNodeNameFilterTask)
	if err != nil {
		return err
	}

	err = inspectionServer.AddTask(k8s_container.InputContainerQueryNamespaceFilterTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(k8s_container.InputContainerQueryPodNamesFilterMask)
	if err != nil {
		return err
	}

	err = inspectionServer.AddTask(k8scontrolplanecomponent.InputControlPlaneComponentNameFilterTask)
	if err != nil {
		return err
	}

	// Query related tasks
	err = inspectionServer.AddTask(k8sauditquery.Task)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(k8s_event.GKEK8sEventLogQueryTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(k8s_node.GKENodeQueryTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(k8s_container.GKEContainerQueryTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(gke_audit.GKEAuditQueryTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(compute_api.ComputeAPIQueryTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(network_api.GCPNetworkLogQueryTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(multicloud_api.MultiCloudAPIQueryTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(autoscaler.AutoscalerQueryTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(onprem_api.OnPremAPIQueryTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(k8scontrolplanecomponent.GKEK8sControlPlaneLogQueryTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(serialport.GKESerialPortLogQueryTask)
	if err != nil {
		return err
	}

	// Parse related tasks
	err = k8s_audit.RegisterK8sAuditTasks(inspectionServer)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(k8s_event.GKEK8sEventLogParseJob)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(k8s_node.GKENodeLogParseJob)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(k8s_container.GKEContainerLogParseJob)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(gke_audit.GKEAuditLogParseJob)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(compute_api.ComputeAPIParserTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(network_api.NetowrkAPIParserTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(multicloud_api.MultiCloudAuditLogParseJob)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(autoscaler.AutoscalerParserTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(onprem_api.OnPremCloudAuditLogParseTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(k8scontrolplanecomponent.GKEK8sControlPlaneComponentLogParseTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(serialport.GKESerialPortLogParseTask)
	if err != nil {
		return err
	}

	// Cluster name prefix tasks
	err = inspectionServer.AddTask(gke.GKEClusterNamePrefixTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(aws.AnthosOnAWSClusterNamePrefixTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(azure.AnthosOnAzureClusterNamePrefixTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(vmware.AnthosOnVMWareClusterNamePrefixTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(baremetal.AnthosOnBaremetalClusterNamePrefixTask)
	if err != nil {
		return err
	}

	// Register inspection types
	err = inspectionServer.AddInspectionType(gke.GKEInspectionType)
	if err != nil {
		return err
	}
	err = inspectionServer.AddInspectionType(aws.AnthosOnAWSInspectionType)
	if err != nil {
		return err
	}
	err = inspectionServer.AddInspectionType(azure.AnthosOnAzureInspectionType)
	if err != nil {
		return err
	}
	err = inspectionServer.AddInspectionType(baremetal.AnthosOnBaremetalInspectionType)
	if err != nil {
		return err
	}
	err = inspectionServer.AddInspectionType(vmware.AnthosOnVMWareInspectionType)
	if err != nil {
		return err
	}
	err = inspectionServer.AddInspectionType(composer_inspection_type.ComposerInspectionType)
	if err != nil {
		return err
	}

	// Composer Query Task
	err = inspectionServer.AddTask(composer_query.ComposerMonitoringLogQueryTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(composer_query.ComposerDagProcessorManagerLogQueryTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(composer_query.ComposerSchedulerLogQueryTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(composer_query.ComposerWorkerLogQueryTask)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(composer_form.AutocompleteClusterNames)
	if err != nil {
		return err
	}
	err = inspectionServer.AddTask(composer_task.ComposerClusterNamePrefixTask)
	if err != nil {
		return err
	}

	// Composer Input Task
	err = inspectionServer.AddTask(composer_form.InputComposerEnvironmentNameTask)
	if err != nil {
		return err
	}

	// Composer AutoComplete Task
	err = inspectionServer.AddTask(composer_form.AutocompleteComposerEnvironmentNames)
	if err != nil {
		return err
	}

	// Composer Parser Task
	err = inspectionServer.AddTask(composer_task.AirflowSchedulerLogParseJob)
	if err != nil {
		return err
	}

	err = inspectionServer.AddTask(composer_task.AirflowWorkerLogParseJob)
	if err != nil {
		return err
	}

	err = inspectionServer.AddTask(composer_task.AirflowDagProcessorLogParseJob)
	if err != nil {
		return err
	}

	return nil
}

func PrepareInspectionServer(inspectionServer *inspection.InspectionTaskServer) error {
	err := commonPreparation(inspectionServer)
	if err != nil {
		return err
	}
	return nil
}
