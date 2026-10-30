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
	"net"
	"strconv"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	mcp "github.com/GoogleCloudPlatform/khi/pkg/server/mcp"
	"github.com/gin-gonic/gin"
)

var (
	// MCPServerKey stores the mcp.Server instance.
	MCPServerKey = typedmap.NewTypedKey[*mcp.Server]("khi.google.com/init/mcp-server")
	// MCPAssetStoreKey stores the mcp.AssetStore instance used to serve MCP-generated assets over HTTP.
	MCPAssetStoreKey = typedmap.NewTypedKey[*mcp.AssetStore]("khi.google.com/init/mcp-asset-store")
)

// mcpAssetsPath is the route prefix of MCP-generated asset URLs under the base path.
const mcpAssetsPath = "/api/v1/mcp-assets/"

// InitializerIDMCPServer identifies the Initializer that sets up the MCP server.
const InitializerIDMCPServer coreinit.InitializerID = "khi.default/mcp-server"

// MCPServerInitializer mounts MCP endpoints onto the Gin router when not in job mode.
var MCPServerInitializer = &coreinit.Initializer{
	ID: InitializerIDMCPServer,
	Dependencies: []coreinit.InitializerID{
		InitializerIDGinServer,
		InitializerIDInspectionTaskServer,
		InitializerIDWorkbenchService,
		InitializerIDFileParameterUpload,
		InitializerIDCDPBrowser,
	},
	Before: []coreinit.InitializerID{
		InitializerIDServerRunner,
	},
	Init: func(ctx *coreinit.InitContext) error {
		jobParams := coreinit.MustGet(ctx, JobParametersKey)
		if *jobParams.JobMode {
			return nil
		}
		inspectionServer := coreinit.MustGet(ctx, InspectionTaskServerKey)
		workbenchManager := coreinit.MustGet(ctx, WorkbenchManagerKey)
		uploadURLIssuer := coreinit.MustGet(ctx, UploadURLIssuerKey)
		serverParams := coreinit.MustGet(ctx, ServerParametersKey)
		basePath := coreinit.MustGet(ctx, BasePathKey)
		router := coreinit.MustGet(ctx, GinRouterKey)

		baseURL := localServerBaseURL(*serverParams.Host, *serverParams.Port, basePath)

		assetStore := mcp.NewAssetStore(baseURL+mcpAssetsPath, mcp.DefaultAssetTTL)
		coreinit.Set(ctx, MCPAssetStoreKey, assetStore)
		serveAsset := func(c *gin.Context) {
			assetStore.ServeAsset(c.Writer, c.Request, c.Param("fileName"))
		}
		router.GET(mcpAssetsPath+":fileName", serveAsset)
		router.HEAD(mcpAssetsPath+":fileName", serveAsset)

		var timelineRenderer mcp.TimelineImageRenderer
		if browser, found := coreinit.Get(ctx, CDPBrowserKey); found && browser != nil {
			renderer := mcp.NewChromeTimelineImageRenderer(browser, baseURL)
			ctx.OnTerminate(func() error {
				return renderer.Close()
			})
			timelineRenderer = renderer
		}

		srv := mcp.NewServer(
			mcp.NewInspectionHandler(inspectionServer, uploadURLIssuer),
			mcp.NewWorkbenchHandler(workbenchManager, timelineRenderer, assetStore),
		)
		coreinit.Set(ctx, MCPServerKey, srv)

		router.Any("/mcp", gin.WrapH(srv.HTTPHandler()))
		router.Any("/mcp/sse", gin.WrapH(srv.SSEHandler()))
		return nil
	},
}

func localServerBaseURL(host string, port int, basePath string) string {
	switch host {
	case "", "0.0.0.0", "::":
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(port)) + basePath
}

func init() {
	coreinit.RegisterInitializer(MCPServerInitializer)
}
