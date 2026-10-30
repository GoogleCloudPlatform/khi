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

	pngBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

	type assetServeResult struct {
		GetStatus       int
		GetContentType  string
		GetBody         []byte
		HeadStatus      int
		HeadBodyByteLen int
	}

	type initOutcome struct {
		HasMCPServer             bool
		HasAssetStore            bool
		AssetServe               *assetServeResult
		HasRenderToolByTransport map[string]bool
		ProbeStatuses            map[string]int
	}

	testCases := []struct {
		name           string
		jobMode        bool
		withCDPBrowser bool
		basePath       string
		probePaths     []string
		want           initOutcome
	}{
		{
			name:    "skips initialization in job mode",
			jobMode: true,
			want: initOutcome{
				HasMCPServer:  false,
				HasAssetStore: false,
			},
		},
		{
			name:     "mounts endpoints and asset store without render_timelines when CDPBrowserKey is absent",
			basePath: "",
			want: initOutcome{
				HasMCPServer:  true,
				HasAssetStore: true,
				AssetServe: &assetServeResult{
					GetStatus:       http.StatusOK,
					GetContentType:  "image/png",
					GetBody:         pngBytes,
					HeadStatus:      http.StatusOK,
					HeadBodyByteLen: 0,
				},
				HasRenderToolByTransport: map[string]bool{
					"streamable": false,
					"sse":        false,
				},
			},
		},
		{
			name:           "mounts endpoints with render_timelines at root base path",
			withCDPBrowser: true,
			basePath:       "",
			want: initOutcome{
				HasMCPServer:  true,
				HasAssetStore: true,
				AssetServe: &assetServeResult{
					GetStatus:       http.StatusOK,
					GetContentType:  "image/png",
					GetBody:         pngBytes,
					HeadStatus:      http.StatusOK,
					HeadBodyByteLen: 0,
				},
				HasRenderToolByTransport: map[string]bool{
					"streamable": true,
					"sse":        true,
				},
			},
		},
		{
			name:           "mounts endpoints with render_timelines under custom base path",
			withCDPBrowser: true,
			basePath:       "/khi-base",
			probePaths: []string{
				"/mcp",
				"/mcp/sse",
				"/api/v1/mcp-assets/missing.png",
			},
			want: initOutcome{
				HasMCPServer:  true,
				HasAssetStore: true,
				AssetServe: &assetServeResult{
					GetStatus:       http.StatusOK,
					GetContentType:  "image/png",
					GetBody:         pngBytes,
					HeadStatus:      http.StatusOK,
					HeadBodyByteLen: 0,
				},
				HasRenderToolByTransport: map[string]bool{
					"streamable": true,
					"sse":        true,
				},
				ProbeStatuses: map[string]int{
					"/mcp":                           http.StatusNotFound,
					"/mcp/sse":                       http.StatusNotFound,
					"/api/v1/mcp-assets/missing.png": http.StatusNotFound,
				},
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

			if err := MCPServerInitializer.Init(ctx); err != nil {
				t.Fatalf("MCPServerInitializer.Init() failed: %v", err)
			}

			_, hasMCPServer := coreinit.Get(ctx, MCPServerKey)
			assetStore, hasAssetStore := coreinit.Get(ctx, MCPAssetStoreKey)

			got := initOutcome{
				HasMCPServer:  hasMCPServer,
				HasAssetStore: hasAssetStore,
			}

			if hasAssetStore {
				grant := assetStore.Issue(pngBytes, "image/png", ".png")

				getReq := httptest.NewRequest(http.MethodGet, grant.URL, nil)
				getRec := httptest.NewRecorder()
				ginEngine.ServeHTTP(getRec, getReq)

				headReq := httptest.NewRequest(http.MethodHead, grant.URL, nil)
				headRec := httptest.NewRecorder()
				ginEngine.ServeHTTP(headRec, headReq)

				got.AssetServe = &assetServeResult{
					GetStatus:       getRec.Code,
					GetContentType:  getRec.Header().Get("Content-Type"),
					GetBody:         getRec.Body.Bytes(),
					HeadStatus:      headRec.Code,
					HeadBodyByteLen: headRec.Body.Len(),
				}
			}

			if len(tc.probePaths) > 0 {
				got.ProbeStatuses = make(map[string]int, len(tc.probePaths))
				for _, path := range tc.probePaths {
					req := httptest.NewRequest(http.MethodGet, path, nil)
					rec := httptest.NewRecorder()
					ginEngine.ServeHTTP(rec, req)
					got.ProbeStatuses[path] = rec.Code
				}
			}

			if hasMCPServer {
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

				got.HasRenderToolByTransport = make(map[string]bool, len(transports))
				for _, tr := range transports {
					clientCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					client := mcpsdk.NewClient(&mcpsdk.Implementation{
						Name:    "test-client",
						Version: "1.0.0",
					}, nil)

					session, err := client.Connect(clientCtx, tr.transport, nil)
					if err != nil {
						cancel()
						t.Fatalf("client.Connect() failed for %s: %v", tr.name, err)
					}

					tools, err := session.ListTools(clientCtx, nil)
					session.Close()
					cancel()
					if err != nil {
						t.Fatalf("session.ListTools() failed for %s: %v", tr.name, err)
					}

					got.HasRenderToolByTransport[tr.name] = slices.ContainsFunc(tools.Tools, func(tool *mcpsdk.Tool) bool {
						return tool.Name == "render_timelines"
					})
				}
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("MCPServerInitializer.Init() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
