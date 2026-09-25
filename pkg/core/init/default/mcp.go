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
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	mcp "github.com/GoogleCloudPlatform/khi/pkg/server/mcp"
	"github.com/gin-gonic/gin"
)

var (
	// MCPServerKey stores the mcp.Server instance.
	MCPServerKey = typedmap.NewTypedKey[*mcp.Server]("khi.google.com/init/mcp-server")
)

// InitializerIDMCPServer identifies the Initializer that sets up the MCP server.
const InitializerIDMCPServer coreinit.InitializerID = "khi.default/mcp-server"

// MCPServerInitializer mounts MCP endpoints onto the Gin router when not in job mode.
var MCPServerInitializer = &coreinit.Initializer{
	ID: InitializerIDMCPServer,
	Dependencies: []coreinit.InitializerID{
		InitializerIDGinServer,
	},
	Before: []coreinit.InitializerID{
		InitializerIDServerRunner,
	},
	Init: func(ctx *coreinit.InitContext) error {
		jobParams := coreinit.MustGet(ctx, JobParametersKey)
		if *jobParams.JobMode {
			return nil
		}
		srv := mcp.NewServer()
		coreinit.Set(ctx, MCPServerKey, srv)

		router := coreinit.MustGet(ctx, GinRouterKey)
		router.Any("/mcp", gin.WrapH(srv.HTTPHandler()))
		router.Any("/mcp/sse", gin.WrapH(srv.SSEHandler()))
		return nil
	},
}

func init() {
	coreinit.RegisterInitializer(MCPServerInitializer)
}
