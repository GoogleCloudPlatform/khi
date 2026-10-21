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
	"time"

	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	"github.com/GoogleCloudPlatform/khi/pkg/server/cdp"
	"github.com/GoogleCloudPlatform/khi/pkg/server/uploadurl"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
	"github.com/gin-gonic/gin"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPServerInitializer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name           string
		jobMode        bool
		withCDPBrowser bool
		basePath       string
		wantMounted    bool
		wantRenderTool bool
		wantNotFound   []string
	}{
		{
			name:        "skips initialization in job mode",
			jobMode:     true,
			basePath:    "",
			wantMounted: false,
		},
		{
			name:           "mounts endpoints without render_timelines when CDPBrowserKey is absent",
			jobMode:        false,
			withCDPBrowser: false,
			basePath:       "",
			wantMounted:    true,
			wantRenderTool: false,
		},
		{
			name:           "mounts endpoints with render_timelines when CDPBrowserKey is present",
			jobMode:        false,
			withCDPBrowser: true,
			basePath:       "/khi-base",
			wantMounted:    true,
			wantRenderTool: true,
			wantNotFound: []string{
				"/mcp",
				"/mcp/sse",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			engine := coreinit.NewEngine(context.Background())
			ctx := engine.Context()

			coreinit.Set(ctx, JobParametersKey, &parameters.JobParameters{
				JobMode: &tc.jobMode,
			})
			host := "127.0.0.1"
			port := 8080
			coreinit.Set(ctx, ServerParametersKey, &parameters.ServerParameters{
				Host: &host,
				Port: &port,
			})
			coreinit.Set(ctx, BasePathKey, tc.basePath)
			if tc.withCDPBrowser {
				coreinit.Set(ctx, CDPBrowserKey, cdp.NewBrowserFromPageWSURL("ws://127.0.0.1:9222/devtools/page/1"))
			}

			taskServer, err := coreinspection.NewServer(nil)
			if err != nil {
				t.Fatalf("coreinspection.NewServer() failed: %v", err)
			}
			coreinit.Set(ctx, InspectionTaskServerKey, taskServer)

			indexMgr := workbench.NewInspectionIndexManager(taskServer, t.TempDir())
			workbenchMgr := workbench.NewWorkbenchManager(taskServer, indexMgr, 3)
			coreinit.Set(ctx, WorkbenchManagerKey, workbenchMgr)
			coreinit.Set(ctx, UploadURLIssuerKey, uploadurl.NewIssuer("http://127.0.0.1:8080/api/v1/file-upload/", 1024, uploadurl.DefaultTTL))

			ginEngine := gin.New()
			var router gin.IRouter = ginEngine.Group(tc.basePath)
			coreinit.Set(ctx, GinRouterKey, router)

			err = MCPServerInitializer.Init(ctx)
			if err != nil {
				t.Fatalf("MCPServerInitializer.Init() failed: %v", err)
			}

			srv, found := coreinit.Get(ctx, MCPServerKey)
			if found != tc.wantMounted {
				t.Errorf("coreinit.Get(MCPServerKey) found = %v, want %v", found, tc.wantMounted)
			}

			if !tc.wantMounted {
				return
			}
			if srv == nil {
				t.Fatal("MCPServerKey is nil")
			}

			for _, path := range tc.wantNotFound {
				req := httptest.NewRequest(http.MethodGet, path, nil)
				w := httptest.NewRecorder()
				ginEngine.ServeHTTP(w, req)
				if w.Code != http.StatusNotFound {
					t.Errorf("GET %q returned %d, want 404", path, w.Code)
				}
			}

			ts := httptest.NewServer(ginEngine)
			defer ts.Close()

			transports := []struct {
				name      string
				transport mcpsdk.Transport
			}{
				{
					name: "streamable",
					transport: &mcpsdk.StreamableClientTransport{
						Endpoint: ts.URL + tc.basePath + "/mcp",
					},
				},
				{
					name: "sse",
					transport: &mcpsdk.SSEClientTransport{
						Endpoint: ts.URL + tc.basePath + "/mcp/sse",
					},
				},
			}

			for _, tr := range transports {
				t.Run(tr.name, func(t *testing.T) {
					clientCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()

					client := mcpsdk.NewClient(&mcpsdk.Implementation{
						Name:    "test-client",
						Version: "1.0.0",
					}, nil)

					session, err := client.Connect(clientCtx, tr.transport, nil)
					if err != nil {
						t.Fatalf("client.Connect() failed for %s: %v", tr.name, err)
					}
					defer session.Close()

					tools, err := session.ListTools(clientCtx, nil)
					if err != nil {
						t.Fatalf("session.ListTools() failed for %s: %v", tr.name, err)
					}
					if tools == nil {
						t.Fatalf("session.ListTools() returned nil for %s", tr.name)
					}
					hasRenderTool := false
					for _, tool := range tools.Tools {
						if tool.Name == "render_timelines" {
							hasRenderTool = true
							break
						}
					}
					if hasRenderTool != tc.wantRenderTool {
						t.Errorf("render_timelines registered = %v, want %v", hasRenderTool, tc.wantRenderTool)
					}
				})
			}
		})
	}
}
