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
	"testing"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	common "github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit"
	googlecloudclustercomposer_impl "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustercomposer/impl"
	googlecloudclustergdcbaremetal_impl "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustergdcbaremetal/impl"
	googlecloudclustergdcvmware_impl "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustergdcvmware/impl"
	googlecloudclustergke_impl "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustergke/impl"
	googlecloudclustergkeonaws_impl "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustergkeonaws/impl"
	googlecloudclustergkeonazure_impl "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudclustergkeonazure/impl"
	googlecloudcommon_impl "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/impl"
	googlecloudk8scommon_impl "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudk8scommon/impl"
	googlecloudlogcomputeapiaudit_impl "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogcomputeapiaudit/impl"
	googlecloudloggkeapiaudit_impl "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudloggkeapiaudit/impl"
	googlecloudlogk8sevent_impl "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogk8sevent/impl"
	googlecloudlogmulticloudapiaudit_impl "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogmulticloudapiaudit/impl"
	googlecloudlognetworkapiaudit_impl "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlognetworkapiaudit/impl"
	googlecloudlogonpremapiaudit_impl "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogonpremapiaudit/impl"
	googlecloudlogserialport_impl "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudlogserialport/impl"
	inspection_test "github.com/GoogleCloudPlatform/khi/pkg/testutil/inspection"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func testPrepareInspectionServer(inspectionServer coreinspection.InspectionTaskRegistry) error {
	err := commonPreparation(inspectionServer)
	if err != nil {
		return err
	}
	return nil
}

func TestInspectionTasksAreResolvable(t *testing.T) {
	inspection_test.ConformanceEveryInspectionTasksAreResolvable(t, "gcp", []coreinspection.InspectionRegistrationFunc{
		common.Register,
		googlecloudcommon_impl.Register,
		googlecloudk8scommon_impl.Register,
		googlecloudclustergke_impl.Register,
		googlecloudclustergdcbaremetal_impl.Register,
		googlecloudclustergdcvmware_impl.Register,
		googlecloudclustergkeonaws_impl.Register,
		googlecloudclustergkeonazure_impl.Register,
		googlecloudclustercomposer_impl.Register,
		googlecloudlogserialport_impl.Register,
		googlecloudlogmulticloudapiaudit_impl.Register,
		googlecloudlogonpremapiaudit_impl.Register,
		googlecloudloggkeapiaudit_impl.Register,
		googlecloudlogcomputeapiaudit_impl.Register,
		googlecloudlogk8sevent_impl.Register,
		googlecloudlognetworkapiaudit_impl.Register,
		testPrepareInspectionServer,
	})
}

func TestConformanceTestForInspectionTypes(t *testing.T) {
	inspection_test.ConformanceTestForInspectionTypes(t, []coreinspection.InspectionRegistrationFunc{
		common.Register,
		googlecloudcommon_impl.Register,
		googlecloudk8scommon_impl.Register,
		googlecloudclustergke_impl.Register,
		googlecloudclustergdcbaremetal_impl.Register,
		googlecloudclustergdcvmware_impl.Register,
		googlecloudclustergkeonaws_impl.Register,
		googlecloudclustergkeonazure_impl.Register,
		googlecloudclustercomposer_impl.Register,
		googlecloudlogserialport_impl.Register,
		googlecloudlogmulticloudapiaudit_impl.Register,
		googlecloudlogonpremapiaudit_impl.Register,
		googlecloudloggkeapiaudit_impl.Register,
		googlecloudlogcomputeapiaudit_impl.Register,
		googlecloudlogk8sevent_impl.Register,
		googlecloudlognetworkapiaudit_impl.Register,
		testPrepareInspectionServer,
	})
}
