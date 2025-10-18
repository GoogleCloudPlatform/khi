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
	"errors"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/api/googlecloud"
	inspectiontest "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/test"
	tasktest "github.com/GoogleCloudPlatform/khi/pkg/core/task/test"
	googlecloudcommon_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/googlecloudcommon/contract"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

func TestAPIClientFactoryTask(t *testing.T) {
	mockOptionCalledCount := 0
	mockOption := func(s *googlecloud.ClientFactory) error {
		mockOptionCalledCount++
		return nil
	}
	testCases := []struct {
		desc                string
		options             []googlecloud.ClientFactoryOption
		wantOptionCallCount int
		wantErr             bool
	}{
		{
			desc:                "no options",
			options:             nil,
			wantOptionCallCount: 0,
		},
		{
			desc:                "with options",
			options:             []googlecloud.ClientFactoryOption{mockOption},
			wantOptionCallCount: 1,
		},
		{
			desc: "error option",
			options: []googlecloud.ClientFactoryOption{
				func(_ *googlecloud.ClientFactory) error {
					return errors.New("test error")
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			mockOptionCalledCount = 0
			ctx := inspectiontest.WithDefaultTestInspectionTaskContext(t.Context())
			clientFactory, _, err := inspectiontest.RunInspectionTask(ctx, APIClientFactoryTask, inspectioncore_contract.TaskModeRun, map[string]any{}, tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.APIClientFactoryOptionsTaskID.Ref(), tc.options))
			if !tc.wantErr && err != nil {
				t.Errorf("APIClientFactoryTask failed: %v", err)
			}
			if tc.wantErr {
				if err == nil {
					t.Errorf("APIClientFactoryTask didn't return error unexpectedly")
				}
				return
			}
			if clientFactory == nil {
				t.Errorf("APIClientFactoryTask returned nil")
			}

			clientFactory2, _, err := inspectiontest.RunInspectionTask(ctx, APIClientFactoryTask, inspectioncore_contract.TaskModeRun, map[string]any{}, tasktest.NewTaskDependencyValuePair(googlecloudcommon_contract.APIClientFactoryOptionsTaskID.Ref(), tc.options))
			if err != nil {
				t.Errorf("APIClientFactoryTask failed on the second time: %v", err)
			}
			if clientFactory != clientFactory2 {
				t.Errorf("APIClientFactoryTask returned different instances")
			}
			if mockOptionCalledCount != tc.wantOptionCallCount {
				t.Errorf("mockOption was called %d times, want %d", mockOptionCalledCount, tc.wantOptionCallCount)
			}
		})
	}

}
