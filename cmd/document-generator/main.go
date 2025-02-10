package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/GoogleCloudPlatform/khi/pkg/document/generator"
	"github.com/GoogleCloudPlatform/khi/pkg/document/model"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/inspection/common"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp"
)

var taskSetRegistrer []inspection.PrepareInspectionServerFunc = make([]inspection.PrepareInspectionServerFunc, 0)

// fatal logs the error and exits if err is not nil.
func fatal(err error, msg string) {
	if err != nil {
		slog.Error(fmt.Sprintf("%s: %v", msg, err))
		os.Exit(1)
	}
}

func init() {
	taskSetRegistrer = append(taskSetRegistrer, common.PrepareInspectionServer)
	taskSetRegistrer = append(taskSetRegistrer, gcp.PrepareInspectionServer)
}

func main() {
	inspectionServer, err := inspection.NewServer()
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to construct the inspection server due to unexpected error\n%v", err))
	}

	for i, taskSetRegistrer := range taskSetRegistrer {
		err = taskSetRegistrer(inspectionServer)
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to call initialize calls for taskSetRegistrer(#%d)\n%v", i, err))
		}
	}

	generator, err := generator.NewDocumentGeneratorFromTemplateFileGlob("./docs/template/*.template.md")
	fatal(err, "failed to load template files")

	inspectionTypeDocumentModel := model.GetInspectionTypeDocumentModel(inspectionServer)
	err = generator.GenerateDocument("./docs/en/inspection-type.md", "inspection-type-template", inspectionTypeDocumentModel, false)
	fatal(err, "failed to generate inspection type document")

	featureDocumentModel, err := model.GetFeatureDocumentModel(inspectionServer)
	fatal(err, "failed to generate feature document model")
	err = generator.GenerateDocument("./docs/en/features.md", "feature-template", featureDocumentModel, false)
	fatal(err, "failed to generate feature document")

	relationshipDocumentModel := model.GetRelationshipDocumentModel()
	err = generator.GenerateDocument("./docs/en/relationships.md", "relationship-template", relationshipDocumentModel, false)
	fatal(err, "failed to generate relationship document")
}
