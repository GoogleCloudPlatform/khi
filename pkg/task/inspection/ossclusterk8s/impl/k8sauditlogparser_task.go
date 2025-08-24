// Copyright 2025 Google LLC
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

package ossclusterk8s_impl

import (
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/recorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/recorder/bindingrecorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/recorder/commonrecorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/recorder/containerstatusrecorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/recorder/endpointslicerecorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/recorder/noderecorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/recorder/ownerreferencerecorder"
	"github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit/recorder/statusrecorder"
	ossclusterk8s_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/ossclusterk8s/contract"
)

// RegisterK8sAuditTasks registers tasks needed for parsing OSS k8s audit logs on the inspection server.
var RegisterK8sAuditTasks coreinspection.InspectionRegistrationFunc = func(registry coreinspection.InspectionTaskRegistry) error {
	manager := recorder.NewAuditRecorderTaskManager(ossclusterk8s_contract.OSSK8sAuditLogParserTaskID, "oss")
	err := commonrecorder.Register(manager)
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

	err = manager.Register(registry, ossclusterk8s_contract.InspectionTypeID)
	if err != nil {
		return err
	}
	return nil
}
