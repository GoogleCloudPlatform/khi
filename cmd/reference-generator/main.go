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
	"github.com/GoogleCloudPlatform/khi/pkg/generated"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// fatal logs the error and exits if err is not nil.
func fatal(err error, msg string) {
	if err != nil {
		slog.Error(fmt.Sprintf("%s: %v", msg, err))
		os.Exit(1)
	}
}

func main() {
	ioconfig, err := inspectioncore_contract.NewIOConfigForTest()
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to construct the IOConfig from parameter\n%v", err))
		os.Exit(1)
	}
	inspectionServer, err := coreinspection.NewServer(ioconfig)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to construct the inspection server due to unexpected error\n%v", err))
	}

	err = generated.RegisterAllInspectionTasks(inspectionServer)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
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
