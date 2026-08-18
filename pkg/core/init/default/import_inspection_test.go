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
	"net/http"
	"net/http/httptest"
	"testing"

	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	"github.com/gin-gonic/gin"
)

func TestImportInspectionInitializer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name        string
		jobMode     bool
		basePath    string
		requestPath string
		wantHit     bool
	}{
		{
			name:        "registers route at root when not in job mode and empty base path",
			jobMode:     false,
			basePath:    "",
			requestPath: "/api.v1.ImportInspectionService/StartImportInspection",
			wantHit:     true,
		},
		{
			name:        "registers route with custom base path when specified",
			jobMode:     false,
			basePath:    "/custom/prefix",
			requestPath: "/custom/prefix/api.v1.ImportInspectionService/StartImportInspection",
			wantHit:     true,
		},
		{
			name:        "skips registration when in job mode",
			jobMode:     true,
			basePath:    "",
			requestPath: "/api.v1.ImportInspectionService/StartImportInspection",
			wantHit:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			engine := coreinit.NewEngine(context.Background())
			ctx := engine.Context()
			jobParams := &parameters.JobParameters{
				JobMode: &tc.jobMode,
			}
			coreinit.Set(ctx, JobParametersKey, jobParams)

			ginEngine := gin.New()
			var router gin.IRouter = ginEngine.Group(tc.basePath)
			taskServer, err := coreinspection.NewServer(nil)
			if err != nil {
				t.Fatalf("failed to create InspectionTaskServer: %v", err)
			}

			coreinit.Set(ctx, InspectionTaskServerKey, taskServer)
			coreinit.Set(ctx, GinRouterKey, router)
			coreinit.Set(ctx, BasePathKey, tc.basePath)

			if err := ImportInspectionInitializer.Init(ctx); err != nil {
				t.Fatalf("ImportInspectionInitializer.Init() failed: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, tc.requestPath, nil)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			ginEngine.ServeHTTP(w, req)

			if tc.wantHit && w.Code == http.StatusNotFound {
				t.Errorf("expected route %q to be registered, got 404", tc.requestPath)
			}
			if !tc.wantHit && w.Code != http.StatusNotFound {
				t.Errorf("expected route %q not to be registered (404), got %d", tc.requestPath, w.Code)
			}
		})
	}
}
