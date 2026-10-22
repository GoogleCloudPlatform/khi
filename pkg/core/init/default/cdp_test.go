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

package defaultinit

import (
	"context"
	"testing"

	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
)

func TestCDPBrowserInitializer(t *testing.T) {
	testCases := []struct {
		name               string
		jobMode            bool
		mcpDisableRenderer bool
		chromeBin          string
		wantBrowserSet     bool
		wantErr            bool
	}{
		{
			name:               "skips browser startup in job mode",
			jobMode:            true,
			mcpDisableRenderer: false,
			wantBrowserSet:     false,
			wantErr:            false,
		},
		{
			name:               "skips browser startup when --mcp-disable-renderer is true",
			jobMode:            false,
			mcpDisableRenderer: true,
			wantBrowserSet:     false,
			wantErr:            false,
		},
		{
			name:               "fails fast when renderer is enabled and Chrome binary is missing",
			jobMode:            false,
			mcpDisableRenderer: false,
			chromeBin:          "/nonexistent/chrome-binary-for-test",
			wantBrowserSet:     false,
			wantErr:            true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.chromeBin != "" {
				t.Setenv("CHROME_BIN", tc.chromeBin)
			}
			engine := coreinit.NewEngine(context.Background())
			defer func() {
				_ = engine.Terminate()
			}()
			ctx := engine.Context()

			coreinit.Set(ctx, JobParametersKey, &parameters.JobParameters{
				JobMode: &tc.jobMode,
			})
			coreinit.Set(ctx, ServerParametersKey, &parameters.ServerParameters{
				MCPDisableRenderer: &tc.mcpDisableRenderer,
			})

			err := CDPBrowserInitializer.Init(ctx)
			if (err != nil) != tc.wantErr {
				t.Fatalf("CDPBrowserInitializer.Init() error = %v, wantErr %v", err, tc.wantErr)
			}

			_, found := coreinit.Get(ctx, CDPBrowserKey)
			if found != tc.wantBrowserSet {
				t.Errorf("coreinit.Get(CDPBrowserKey) found = %v, want %v", found, tc.wantBrowserSet)
			}
		})
	}
}
