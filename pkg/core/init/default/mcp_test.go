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
	"slices"
	"testing"
	"time"

	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	"github.com/GoogleCloudPlatform/khi/pkg/server/cdp"
	"github.com/GoogleCloudPlatform/khi/pkg/server/uploadurl"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPServerInitializer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name                      string
		jobMode                   bool
		withCDPBrowser            bool
		basePath                  string
		wantMCP                   bool
		wantRenderTool            bool
		wantRenderedTimelineStore bool
		wantNotFound              []string
	}{
		{
			name:    "skips initialization in job mode",
			jobMode: true,
		},
		{
			name:     "mounts endpoints without render_timelines when CDPBrowserKey is absent",
			basePath: "",
			wantMCP:  true,
		},
		{
			name:                      "mounts endpoints with render_timelines at root base path",
			withCDPBrowser:            true,
			basePath:                  "",
			wantMCP:                   true,
			wantRenderTool:            true,
			wantRenderedTimelineStore: true,
		},
		{
			name:                      "mounts endpoints with render_timelines under custom base path",
			withCDPBrowser:            true,
			basePath:                  "/khi-base",
			wantMCP:                   true,
			wantRenderTool:            true,
			wantRenderedTimelineStore: true,
			wantNotFound: []string{
				"/mcp",
				"/mcp/sse",
				"/api/v1/rendered-timelines/missing.png",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			engine := coreinit.NewEngine(context.Background())
			t.Cleanup(func() {
				_ = engine.Terminate()
			})
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

			_, found := coreinit.Get(ctx, MCPServerKey)
			if found != tc.wantMCP {
				t.Errorf("coreinit.Get(MCPServerKey) found = %v, want %v", found, tc.wantMCP)
			}

			imageStore, storeFound := coreinit.Get(ctx, RenderedTimelineStoreKey)
			if storeFound != tc.wantRenderedTimelineStore {
				t.Errorf("coreinit.Get(RenderedTimelineStoreKey) found = %v, want %v", storeFound, tc.wantRenderedTimelineStore)
			}

			if !tc.wantMCP {
				return
			}

			if tc.wantRenderedTimelineStore {
				pngBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
				grant := imageStore.Issue(pngBytes)

				getReq := httptest.NewRequest(http.MethodGet, grant.URL, nil)
				getRec := httptest.NewRecorder()
				ginEngine.ServeHTTP(getRec, getReq)
				if getRec.Code != http.StatusOK {
					t.Errorf("GET %s returned %d, want %d: %s", grant.URL, getRec.Code, http.StatusOK, getRec.Body.String())
				}
				if gotType := getRec.Header().Get("Content-Type"); gotType != "image/png" {
					t.Errorf("GET %s Content-Type = %q, want %q", grant.URL, gotType, "image/png")
				}
				if diff := cmp.Diff(pngBytes, getRec.Body.Bytes()); diff != "" {
					t.Errorf("GET %s body mismatch (-want +got):\n%s", grant.URL, diff)
				}

				headReq := httptest.NewRequest(http.MethodHead, grant.URL, nil)
				headRec := httptest.NewRecorder()
				ginEngine.ServeHTTP(headRec, headReq)
				if headRec.Code != http.StatusOK {
					t.Errorf("HEAD %s returned %d, want %d", grant.URL, headRec.Code, http.StatusOK)
				}
				if headRec.Body.Len() != 0 {
					t.Errorf("HEAD %s body len = %d, want 0", grant.URL, headRec.Body.Len())
				}
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
					hasRenderTool := slices.ContainsFunc(tools.Tools, func(tool *mcpsdk.Tool) bool {
						return tool.Name == "render_timelines"
					})
					if hasRenderTool != tc.wantRenderTool {
						t.Errorf("render_timelines registered = %v, want %v", hasRenderTool, tc.wantRenderTool)
					}
				})
			}
		})
	}
}
