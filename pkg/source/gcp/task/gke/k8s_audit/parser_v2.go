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
	"github.com/GoogleCloudPlatform/khi/pkg/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/recorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/recorder/bindingrecorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/recorder/commonrecorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/recorder/containerstatusrecorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/recorder/endpointslicerecorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/recorder/noderecorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/recorder/ownerreferencerecorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/recorder/snegrecorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/recorder/statusrecorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/v2commonlogparse"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/v2logconvert"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/v2manifestgenerate"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/task/gke/k8s_audit/v2timelinegrouping"
)

var PrepareK8sAuditTasks inspection.PrepareInspectionServerFunc = func(inspectionServer *inspection.InspectionTaskServer) error {
	err := inspectionServer.AddTaskDefinition(v2commonlogparse.Task)
	if err != nil {
		return err
	}

	err = inspectionServer.AddTaskDefinition(v2timelinegrouping.Task)
	if err != nil {
		return err
	}

	err = inspectionServer.AddTaskDefinition(v2manifestgenerate.Task)
	if err != nil {
		return err
	}

	err = inspectionServer.AddTaskDefinition(v2logconvert.Task)
	if err != nil {
		return err
	}
	manager := recorder.NewTaskManager()
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
	err = snegrecorder.Register(manager)
	if err != nil {
		return err
	}
	err = noderecorder.Register(manager)
	if err != nil {
		return err
	}
	err = manager.Register(inspectionServer)
	if err != nil {
		return err
	}
	return nil
}
