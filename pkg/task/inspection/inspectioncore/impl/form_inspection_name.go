// Copyright 2026 Google LLC
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

package inspectioncore_impl

import (
	"context"
	"errors"
	"fmt"
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
		inspectionID := khictx.MustGetValue(ctx, inspectioncore.InspectionTaskInspectionID)
		baseName, err := khictx.GetValue(ctx, inspectioncore.InspectionTypeName)
		if err != nil || baseName == "" {
			baseName = "Inspection"
		}
		if registry, err := khictx.GetValue(ctx, inspectioncore.InspectionNameRegistryKey); err == nil && registry != nil {
			return registry.ResolveUniqueName(inspectionID, baseName), nil
		}
		return baseName, nil
	}).
	WithValidator(func(ctx context.Context, value string) (string, error) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return "inspection name must not be empty", nil
		}
		inspectionID := khictx.MustGetValue(ctx, inspectioncore.InspectionTaskInspectionID)
		if registry, err := khictx.GetValue(ctx, inspectioncore.InspectionNameRegistryKey); err == nil && registry != nil {
			if err := registry.ReserveName(inspectionID, trimmed); err != nil {
				if errors.Is(err, inspectioncore.ErrInspectionNameAlreadyInUse) {
					return fmt.Sprintf("inspection name %q is already in use", trimmed), nil
				}
				return err.Error(), nil
			}
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
