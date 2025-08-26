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

package googlecloudcommon_impl

import (
	"context"
	"regexp"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/formtask"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
)

var projectIdValidator = regexp.MustCompile(`^\s*[0-9a-z\.:\-]+\s*$`)

// InputProjectIdTask defines a form task for inputting the Google Cloud project ID.
var InputProjectIdTask = formtask.NewTextFormTaskBuilder(googlecloudcommon_contract.InputProjectIdTaskID, googlecloudcommon_contract.PriorityForResourceIdentifierGroup+5000, "Project ID").
	WithDescription("The project ID containing logs of the cluster to query").
	WithValidator(func(ctx context.Context, value string) (string, error) {
		if !projectIdValidator.Match([]byte(value)) {
			return "Project ID must match `^*[0-9a-z\\.:\\-]+$`", nil
		}
		return "", nil
	}).
	WithReadonlyFunc(func(ctx context.Context) (bool, error) {
		if parameters.Auth.FixedProjectID == nil {
			return false, nil
		}
		return *parameters.Auth.FixedProjectID != "", nil
	}).
	WithDefaultValueFunc(func(ctx context.Context, previousValues []string) (string, error) {
		if parameters.Auth.FixedProjectID != nil && *parameters.Auth.FixedProjectID != "" {
			return *parameters.Auth.FixedProjectID, nil
		}
		if len(previousValues) > 0 {
			return previousValues[0], nil
		}
		return "", nil
	}).
	WithConverter(func(ctx context.Context, value string) (string, error) {
		return strings.TrimSpace(value), nil
	}).
	Build()
