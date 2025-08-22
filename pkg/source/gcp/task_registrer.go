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
)

func commonPreparation(registry coreinspection.InspectionTaskRegistry) error {
	err := coretask.RegisterTasks(registry,
		// Form input related tasks
		task.InputNodeNameFilterTask,
		k8s_container.InputContainerQueryNamespaceFilterTask,
		k8s_container.InputContainerQueryPodNamesFilterMask,
		k8scontrolplanecomponent.InputControlPlaneComponentNameFilterTask,
		// Query related tasks
		k8sauditquery.Task,
		k8s_event.GKEK8sEventLogQueryTask,
		k8s_node.GKENodeQueryTask,
		k8s_container.GKEContainerQueryTask,
		gke_audit.GKEAuditQueryTask,
		compute_api.ComputeAPIQueryTask,
		network_api.GCPNetworkLogQueryTask,
		autoscaler.AutoscalerQueryTask,
		k8scontrolplanecomponent.GKEK8sControlPlaneLogQueryTask,
		serialport.GKESerialPortLogQueryTask,
		k8s_event.GKEK8sEventLogParseJob,
		k8s_node.GKENodeLogParseJob,
		k8s_container.GKEContainerLogParseJob,
		gke_audit.GKEAuditLogParseJob,
		compute_api.ComputeAPIParserTask,
		network_api.NetowrkAPIParserTask,
		autoscaler.AutoscalerParserTask,
		k8scontrolplanecomponent.GKEK8sControlPlaneComponentLogParseTask,
		serialport.GKESerialPortLogParseTask,
	)
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
