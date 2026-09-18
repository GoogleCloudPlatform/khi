package inspectioncore_impl

import (
	"context"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore"
)

// inspectionNameFormPriority places the inspection name input field at the top of the parameter form.
const inspectionNameFormPriority = 200000

// InputInspectionNameTask is a form task that allows users to specify the display name of the inspection.
var InputInspectionNameTask = formtask.NewTextFormTaskBuilder(
	inspectioncore.InputInspectionNameTaskID,
	inspectionNameFormPriority,
	"Inspection name",
).
	WithDescription("The display name of this inspection.").
	WithDefaultValueFunc(func(ctx context.Context, previousValues []string) (string, error) {
		defaultName, err := khictx.GetValue(ctx, inspectioncore.DefaultInspectionName)
		if err != nil || defaultName == "" {
			return "Inspection", nil
		}
		return defaultName, nil
	}).
	WithValidator(func(ctx context.Context, value string) (string, error) {
		if strings.TrimSpace(value) == "" {
			return "inspection name must not be empty", nil
		}
		return "", nil
	}).
	WithConverter(func(ctx context.Context, value string) (string, error) {
		trimmed := strings.TrimSpace(value)
		metadataSet := khictx.MustGetValue(ctx, inspectionmetadata.MapContextKey)
		if header, found := typedmap.Get(metadataSet, inspectionmetadata.HeaderMetadataKey); found && header != nil {
			header.InspectionName = trimmed
		}
		return trimmed, nil
	}).
	Build(coretask.NewRequiredTaskLabel())
