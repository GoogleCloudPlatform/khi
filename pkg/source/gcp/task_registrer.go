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
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
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

func commonPreparation(registry coreinspection.InspectionTaskRegistry) error {
	err := coretask.RegisterTasks(registry,
		task.GCPDefaultK8sResourceMergeConfigTask,
		gcpcommon.HeaderSuggestedFileNameTask,
		gke.AutocompleteClusterNames,
		aws.AutocompleteClusterNames,
		azure.AutocompleteClusterNames,
		baremetal.AutocompleteClusterNames,
		vmware.AutocompleteClusterNames,
		task.AutocompleteLocationTask,
		// Form input related tasks
		task.TimeZoneShiftInputTask,
		task.InputProjectIdTask,
		task.InputClusterNameTask,
		task.InputDurationTask,
		task.InputEndTimeTask,
		task.InputStartTimeTask,
		task.InputKindFilterTask,
		task.InputLocationsTask,
		task.InputNamespaceFilterTask,
		task.InputNodeNameFilterTask,
		k8s_container.InputContainerQueryNamespaceFilterTask,
		k8s_container.InputContainerQueryPodNamesFilterMask,
		k8scontrolplanecomponent.InputControlPlaneComponentNameFilterTask,
		// Query related tasks
		task.QueryResourceNameInputTask,
		k8sauditquery.Task,
		k8s_event.GKEK8sEventLogQueryTask,
		k8s_node.GKENodeQueryTask,
		k8s_container.GKEContainerQueryTask,
		gke_audit.GKEAuditQueryTask,
		compute_api.ComputeAPIQueryTask,
		network_api.GCPNetworkLogQueryTask,
		multicloud_api.MultiCloudAPIQueryTask,
		autoscaler.AutoscalerQueryTask,
		onprem_api.OnPremAPIQueryTask,
		k8scontrolplanecomponent.GKEK8sControlPlaneLogQueryTask,
		serialport.GKESerialPortLogQueryTask,
		k8s_event.GKEK8sEventLogParseJob,
		k8s_node.GKENodeLogParseJob,
		k8s_container.GKEContainerLogParseJob,
		gke_audit.GKEAuditLogParseJob,
		compute_api.ComputeAPIParserTask,
		network_api.NetowrkAPIParserTask,
		multicloud_api.MultiCloudAuditLogParseJob,
		autoscaler.AutoscalerParserTask,
		onprem_api.OnPremCloudAuditLogParseTask,
		k8scontrolplanecomponent.GKEK8sControlPlaneComponentLogParseTask,
		serialport.GKESerialPortLogParseTask,
		// Cluster name prefix tasks
		gke.GKEClusterNamePrefixTask,
		aws.AnthosOnAWSClusterNamePrefixTask,
		azure.AnthosOnAzureClusterNamePrefixTask,
		vmware.AnthosOnVMWareClusterNamePrefixTask,
		baremetal.AnthosOnBaremetalClusterNamePrefixTask,
		// Composer Query Task
		composer_query.ComposerMonitoringLogQueryTask,
		composer_query.ComposerDagProcessorManagerLogQueryTask,
		composer_query.ComposerSchedulerLogQueryTask,
		composer_query.ComposerWorkerLogQueryTask,
		composer_form.AutocompleteClusterNames,
		composer_task.ComposerClusterNamePrefixTask,
		// Composer Input Task
		composer_form.InputComposerEnvironmentNameTask,
		// Composer AutoComplete Task
		composer_form.AutocompleteComposerEnvironmentNames,
		// Composer Parser Task
		composer_task.AirflowSchedulerLogParseJob,
		composer_task.AirflowWorkerLogParseJob,
		composer_task.AirflowDagProcessorLogParseJob,
	)
	if err != nil {
		return err
	}

	// Register inspection types
	err = registry.AddInspectionType(gke.GKEInspectionType)
	if err != nil {
		return err
	}
	err = registry.AddInspectionType(aws.AnthosOnAWSInspectionType)
	if err != nil {
		return err
	}
	err = registry.AddInspectionType(azure.AnthosOnAzureInspectionType)
	if err != nil {
		return err
	}
	err = registry.AddInspectionType(baremetal.AnthosOnBaremetalInspectionType)
	if err != nil {
		return err
	}
	err = registry.AddInspectionType(vmware.AnthosOnVMWareInspectionType)
	if err != nil {
		return err
	}
	err = registry.AddInspectionType(composer_inspection_type.ComposerInspectionType)
	if err != nil {
		return err
	}
	// Parse related tasks
	err = k8s_audit.RegisterK8sAuditTasks(registry)
	if err != nil {
		return err
	}

	return nil
}

func Register(registry coreinspection.InspectionTaskRegistry) error {
	err := commonPreparation(registry)
	if err != nil {
		return err
	}
	return nil
}
