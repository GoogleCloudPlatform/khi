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
	"net/http"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud/options"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
)

type mockCallerOptionsInjectorOpton struct {
}

// ApplyToCallContext implements googlecloud.CallOptionInjectorOption.
func (m *mockCallerOptionsInjectorOpton) ApplyToCallContext(ctx context.Context, container googlecloud.ResourceContainer) context.Context {
	return ctx
}

// ApplyToRawHTTPHeader implements googlecloud.CallOptionInjectorOption.
func (m *mockCallerOptionsInjectorOpton) ApplyToRawHTTPHeader(header http.Header, container googlecloud.ResourceContainer) {
}

var _ googlecloud.CallOptionInjectorOption = (*mockCallerOptionsInjectorOpton)(nil)

func TestAPIClientFactoryOptionsTask(t *testing.T) {
	option1 := options.QuotaProject("foo")
	option2 := options.QuotaProject("bar")

	testCases := []struct {
		desc           string
		prepareContext func(ctx context.Context) context.Context
		disabled       bool
		wantCount      int
	}{
		{
			desc: "without options in context includes adaptive rate limiter by default",
			prepareContext: func(ctx context.Context) context.Context {
				return ctx
			},
			disabled:  false,
			wantCount: 1, // AdaptiveRateLimiter option
		},
		{
			desc: "with options in context",
			prepareContext: func(ctx context.Context) context.Context {
				opt1 := coreinspection.RunContextOptionArrayElementFromValue(googlecloudcommon_contract.APIClientFactoryOptionsContextKey, option1)
				opt2 := coreinspection.RunContextOptionArrayElementFromValue(googlecloudcommon_contract.APIClientFactoryOptionsContextKey, option2)
				ctx, _ = opt1(ctx, inspectioncore_contract.TaskModeRun)
				ctx, _ = opt2(ctx, inspectioncore_contract.TaskModeRun)
				return ctx
			},
			disabled:  false,
			wantCount: 3, // 2 from context + 1 AdaptiveRateLimiter option
		},
		{
			desc: "with adaptive rate limit disabled",
			prepareContext: func(ctx context.Context) context.Context {
				return ctx
			},
			disabled:  true,
			wantCount: 0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			enabledBefore := parameters.RateLimit.LoggingAdaptiveRateLimitEnabled
			disabledVal := !tc.disabled
			parameters.RateLimit.LoggingAdaptiveRateLimitEnabled = &disabledVal
			defer func() {
				parameters.RateLimit.LoggingAdaptiveRateLimitEnabled = enabledBefore
			}()

			ctx := tc.prepareContext(context.Background())
			ctx = inspectiontest.WithDefaultTestInspectionTaskContext(ctx)
			gotOptions, _, err := inspectiontest.RunInspectionTask(ctx, APIClientFactoryOptionsTask, inspectioncore_contract.TaskModeRun, map[string]any{})
			if err != nil {
				t.Fatalf("APIClientFactoryOptionsTask failed: %v", err)
			}
			if len(gotOptions) != tc.wantCount {
				t.Errorf("got %d options, want %d", len(gotOptions), tc.wantCount)
			}
		})
	}

}

func TestAPICallOptionsInjectorTask(t *testing.T) {
	option1 := &mockCallerOptionsInjectorOpton{}
	option2 := &mockCallerOptionsInjectorOpton{}

	testCases := []struct {
		desc                    string
		prepareContext          func(ctx context.Context) context.Context
		wantCallOptionsInjector *googlecloud.CallOptionInjector
	}{
		{
			desc: "without options in context",
			prepareContext: func(ctx context.Context) context.Context {
				return ctx
			},
			wantCallOptionsInjector: googlecloud.NewCallOptionInjector(),
		},
		{
			desc: "with options",
			prepareContext: func(ctx context.Context) context.Context {
				opt1 := coreinspection.RunContextOptionArrayElementFromValue[googlecloud.CallOptionInjectorOption](googlecloudcommon_contract.APICallOptionsInjectorContextKey, option1)
				opt2 := coreinspection.RunContextOptionArrayElementFromValue[googlecloud.CallOptionInjectorOption](googlecloudcommon_contract.APICallOptionsInjectorContextKey, option2)
				ctx, _ = opt1(ctx, inspectioncore_contract.TaskModeRun)
				ctx, _ = opt2(ctx, inspectioncore_contract.TaskModeRun)
				return ctx
			},
			wantCallOptionsInjector: googlecloud.NewCallOptionInjector(option1, option2),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := tc.prepareContext(context.Background())
			ctx = inspectiontest.WithDefaultTestInspectionTaskContext(ctx)
			gotCallOptionsInjector, _, err := inspectiontest.RunInspectionTask(ctx, APICallOptionsInjectorTask, inspectioncore_contract.TaskModeRun, map[string]any{})
			if err != nil {
				t.Fatalf("APICallOptionsInjectorTask failed: %v", err)
			}
			if diff := cmp.Diff(tc.wantCallOptionsInjector, gotCallOptionsInjector, cmp.AllowUnexported(googlecloud.CallOptionInjector{})); diff != "" {
				t.Errorf("APICallOptionsInjectorTask returned unexpected diff (-want +got):\n%s", diff)
			}
		})
	}
}
