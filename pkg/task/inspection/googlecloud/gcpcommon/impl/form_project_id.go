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

package gcpcommon_impl

import (
	"context"
	"regexp"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	"github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloud/gcpcommon"
)

var projectIDValidator = regexp.MustCompile(`^\s*[0-9a-z\.:\-]+\s*$`)

// inputProjectIDTask defines a form task for inputting the Google Cloud project ID.
var inputProjectIDTask = formtask.DefineTextForm(
	gcpcommon.InputProjectIdTaskID,
	gcpcommon.PriorityForResourceIdentifierGroup+5000,
	"Project ID",
	"The project ID containing logs of the cluster to query",
	func(b *coretask.Binder) formtask.TextFormSpec[string] {
		return formtask.TextFormSpec[string]{
			ValidationTiming: inspectionmetadata.Blur,
			Validator: func(ctx context.Context, value string) (string, error) {
				if !projectIDValidator.Match([]byte(value)) {
					return "Project ID must match `^*[0-9a-z\\.:\\-]+$`", nil
				}
				return "", nil
			},
			Readonly: func(ctx context.Context) (bool, error) {
				if parameters.Auth.FixedProjectID == nil {
					return false, nil
				}
				return *parameters.Auth.FixedProjectID != "", nil
			},
			DefaultValue: func(ctx context.Context, previousValues []string) (string, error) {
				if parameters.Auth.FixedProjectID != nil && *parameters.Auth.FixedProjectID != "" {
					return *parameters.Auth.FixedProjectID, nil
				}
				if len(previousValues) > 0 {
					return previousValues[0], nil
				}
				return "", nil
			},
			Converter: func(ctx context.Context, value string) (string, error) {
				return strings.TrimSpace(value), nil
			},
		}
	},
)
