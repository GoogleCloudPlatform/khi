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
	"errors"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	optionspkg "github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/options"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/ratelimit"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khictx"
	"github.com/GoogleCloudPlatform/khi/pkg/common/khierrors"
	inspectiontaskbase "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/taskbase"
	coretask "github.com/GoogleCloudPlatform/khi/pkg/core/task"
	"github.com/GoogleCloudPlatform/khi/pkg/core/task/taskid"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// APIClientFactoryOptionsTask is the default implementation to provide the list of googlecloud.ClientFactoryOption.
// User can extend this behavior with defining new task for googlecloudcommon_contract.APIClientFactoryOptionsTaskID with higher selection priority.
var APIClientFactoryOptionsTask = inspectiontaskbase.NewInspectionTask(
	googlecloudcommon_contract.APIClientFactoryOptionsTaskID,
	[]taskid.UntypedTaskReference{},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) ([]googlecloud.ClientFactoryOption, error) {
		var options []googlecloud.ClientFactoryOption
		optionsFromContext, err := khictx.GetValue(ctx, googlecloudcommon_contract.APIClientFactoryOptionsContextKey)
		if err != nil && !errors.Is(err, khierrors.ErrNotFound) {
			return nil, err
		}
		if optionsFromContext != nil {
			options = *optionsFromContext
		}

		if parameters.RateLimit.LoggingAdaptiveRateLimitEnabled == nil || *parameters.RateLimit.LoggingAdaptiveRateLimitEnabled {
			quotaProjectID := ""
			if parameters.Auth.QuotaProjectID != nil {
				quotaProjectID = *parameters.Auth.QuotaProjectID
			}
			cfg := ratelimit.DefaultAdaptiveRateLimiterConfig()
			if parameters.RateLimit.LoggingInitialQPS != nil {
				cfg.InitialQPS = *parameters.RateLimit.LoggingInitialQPS
			}
			if parameters.RateLimit.LoggingMinQPS != nil {
				cfg.MinQPS = *parameters.RateLimit.LoggingMinQPS
			}
			if parameters.RateLimit.LoggingMaxQPS != nil {
				cfg.MaxQPS = *parameters.RateLimit.LoggingMaxQPS
			}

			limiterPool := ratelimit.NewPool(cfg, quotaProjectID)
			options = append(options, optionspkg.AdaptiveRateLimiter(limiterPool))
		}

		return options, nil
	},
	coretask.WithSelectionPriority(googlecloudcommon_contract.DefaultAPIClientOptionTasksPriority),
)

// APICallOptionsInjectorTask is the default implementation to provide the CallOptionInjector.
// Each APIClient use must call this injector method before to supply parameters correctly.
var APICallOptionsInjectorTask = inspectiontaskbase.NewInspectionTask(
	googlecloudcommon_contract.APIClientCallOptionsInjectorTaskID,
	[]taskid.UntypedTaskReference{},
	func(ctx context.Context, taskMode inspectioncore_contract.InspectionTaskModeType) (*googlecloud.CallOptionInjector, error) {
		var options []googlecloud.CallOptionInjectorOption
		optionsFromContext, err := khictx.GetValue(ctx, googlecloudcommon_contract.APICallOptionsInjectorContextKey)
		if err != nil && !errors.Is(err, khierrors.ErrNotFound) {
			return nil, err
		}
		if optionsFromContext != nil {
			options = *optionsFromContext
		}
		return googlecloud.NewCallOptionInjector(options...), nil
	},
	coretask.WithSelectionPriority(googlecloudcommon_contract.DefaultAPIClientOptionTasksPriority),
)
