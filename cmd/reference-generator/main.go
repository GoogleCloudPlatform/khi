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

package main

// cmd/reference-generator/main.go
// Generates KHI reference documents from the task graph or constants defined in code base.

import (
	"fmt"
	"log/slog"
	"os"

	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/document/generator"
	"github.com/GoogleCloudPlatform/khi/pkg/document/model"
	common_k8saudit "github.com/GoogleCloudPlatform/khi/pkg/source/common/k8s_audit"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp"
	"github.com/GoogleCloudPlatform/khi/pkg/source/oss"
	inspection_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/contract"
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
)

var taskSetRegistrer []coreinspection.InspectionRegistrationFunc = make([]coreinspection.InspectionRegistrationFunc, 0)

// fatal logs the error and exits if err is not nil.
func fatal(err error, msg string) {
	if err != nil {
		slog.Error(fmt.Sprintf("%s: %v", msg, err))
		os.Exit(1)
	}
}

func init() {
	taskSetRegistrer = append(taskSetRegistrer, gcp.Register)
	taskSetRegistrer = append(taskSetRegistrer, oss.Register)
	taskSetRegistrer = append(taskSetRegistrer, common_k8saudit.Register)
	taskSetRegistrer = append(taskSetRegistrer, googlecloudcommon_impl.Register)
	taskSetRegistrer = append(taskSetRegistrer, googlecloudk8scommon_impl.Register)
	taskSetRegistrer = append(taskSetRegistrer, googlecloudclustergke_impl.Register)
	taskSetRegistrer = append(taskSetRegistrer, googlecloudclustergdcvmware_impl.Register)
	taskSetRegistrer = append(taskSetRegistrer, googlecloudclustergdcbaremetal_impl.Register)
	taskSetRegistrer = append(taskSetRegistrer, googlecloudclustergkeonaws_impl.Register)
	taskSetRegistrer = append(taskSetRegistrer, googlecloudclustergkeonazure_impl.Register)
	taskSetRegistrer = append(taskSetRegistrer, googlecloudclustercomposer_impl.Register)
	taskSetRegistrer = append(taskSetRegistrer, googlecloudlogserialport_impl.Register)
	taskSetRegistrer = append(taskSetRegistrer, googlecloudlogmulticloudapiaudit_impl.Register)
	taskSetRegistrer = append(taskSetRegistrer, googlecloudlogonpremapiaudit_impl.Register)
	taskSetRegistrer = append(taskSetRegistrer, googlecloudloggkeapiaudit_impl.Register)
	taskSetRegistrer = append(taskSetRegistrer, googlecloudlogcomputeapiaudit_impl.Register)
	taskSetRegistrer = append(taskSetRegistrer, googlecloudlogk8sevent_impl.Register)
	taskSetRegistrer = append(taskSetRegistrer, googlecloudlognetworkapiaudit_impl.Register)
}

func main() {
	ioconfig, err := inspection_contract.NewIOConfigForTest()
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to construct the IOConfig from parameter\n%v", err))
		os.Exit(1)
	}
	inspectionServer, err := coreinspection.NewServer(ioconfig)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to construct the inspection server due to unexpected error\n%v", err))
	}

	for i, taskSetRegistrer := range taskSetRegistrer {
		err = taskSetRegistrer(inspectionServer)
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to call initialize calls for taskSetRegistrer(#%d)\n%v", i, err))
		}
	}

	generator, err := generator.NewDocumentGeneratorFromTemplateFileGlob("./docs/template/reference/*.template.md")
	fatal(err, "failed to load template files")

	// Generate the reference for inspection types
	inspectionTypeDocumentModel := model.GetInspectionTypeDocumentModel(inspectionServer)
	err = generator.GenerateDocument("./docs/en/reference/inspection-type.md", "inspection-type-template", inspectionTypeDocumentModel, false)
	fatal(err, "failed to generate inspection type document")

	featureDocumentModel, err := model.GetFeatureDocumentModel(inspectionServer)
	fatal(err, "failed to generate feature document model")
	err = generator.GenerateDocument("./docs/en/reference/features.md", "feature-template", featureDocumentModel, false)
	fatal(err, "failed to generate feature document")

	formDocumentModel, err := model.GetFormDocumentModel(inspectionServer)
	fatal(err, "failed to generate form document model")
	err = generator.GenerateDocument("./docs/en/reference/forms.md", "form-template", formDocumentModel, false)
	fatal(err, "failed to generate form document")

	relationshipDocumentModel := model.GetRelationshipDocumentModel()
	err = generator.GenerateDocument("./docs/en/reference/relationships.md", "relationship-template", relationshipDocumentModel, false)
	fatal(err, "failed to generate relationship document")
}
