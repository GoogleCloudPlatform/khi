package defaultinit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	"github.com/GoogleCloudPlatform/khi/pkg/server/upload"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// InitializerIDJobRunner executes batch inspection tasks in job mode.
const InitializerIDJobRunner coreinit.InitializerID = "khi.default/job-runner"

// JobRunnerInitializer handles batch execution when JobMode is enabled.
var JobRunnerInitializer = &coreinit.Initializer{
	ID: InitializerIDJobRunner,
	Dependencies: []coreinit.InitializerID{
		InitializerIDInspectionTaskServer,
		InitializerIDParameterParse,
	},
	Init: func(ctx *coreinit.InitContext) error {
		jobParams := coreinit.MustGet(ctx, JobParametersKey)
		if !*jobParams.JobMode {
			return nil
		}
		upload.DefaultUploadFileStore = upload.NewJobModeStore()
		inspectionServer := coreinit.MustGet(ctx, InspectionTaskServerKey)

		ctx.OnRun(func(runCtx context.Context) error {
			slog.Info("Starting Kubernetes History Inspector as job mode...")
			queryParametersInJson := *jobParams.InspectionValues
			var values map[string]any
			if err := json.Unmarshal([]byte(queryParametersInJson), &values); err != nil {
				return fmt.Errorf("failed to parse inspection value %s: %w", queryParametersInJson, err)
			}
			inspectionID, err := inspectionServer.CreateInspection(*jobParams.InspectionType)
			if err != nil {
				return fmt.Errorf("failed to create inspection type %s: %w", *jobParams.InspectionType, err)
			}

			features := strings.Split(*jobParams.InspectionFeatures, ",")
			t := inspectionServer.GetInspection(inspectionID)
			if len(features) == 1 && strings.ToUpper(features[0]) == "ALL" {
				availableFeatures, err := t.FeatureList()
				if err != nil {
					return fmt.Errorf("failed to obtain feature list: %w", err)
				}
				allFeatures := []string{}
				for _, af := range availableFeatures {
					allFeatures = append(allFeatures, af.Id)
				}
				features = allFeatures
			}
			if err := t.SetFeatureList(features); err != nil {
				return fmt.Errorf("failed to set features: %w", err)
			}
			if err := t.Run(runCtx, &inspectioncore_contract.InspectionRequest{Values: values}); err != nil {
				return fmt.Errorf("failed to run inspection task: %w", err)
			}
			<-t.Wait()
			result, err := t.Result()
			if err != nil {
				return fmt.Errorf("failed to get inspection result: %w", err)
			}
			reader, err := result.ResultStore.GetReader()
			if err != nil {
				return fmt.Errorf("failed to get result reader: %w", err)
			}
			defer reader.Close()
			file, err := os.OpenFile(*jobParams.ExportDestination, os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				return fmt.Errorf("failed to open export file: %w", err)
			}
			defer file.Close()
			if _, err := io.Copy(file, reader); err != nil {
				return fmt.Errorf("failed to write export file: %w", err)
			}
			return nil
		})
		return nil
	},
}

func init() {
	coreinit.RegisterInitializer(JobRunnerInitializer)
}
