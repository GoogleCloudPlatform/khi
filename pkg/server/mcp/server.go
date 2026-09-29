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

package mcp

import (
	"net/http"

	"github.com/GoogleCloudPlatform/khi/pkg/common/constants"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// DomainHandler registers tools, resources, and resource templates to an MCP server.
type DomainHandler interface {
	Register(srv *mcpsdk.Server)
}

// Server wraps an MCP server with Streamable HTTP and SSE handlers.
type Server struct {
	mcpServer         *mcpsdk.Server
	streamableHandler *mcpsdk.StreamableHTTPHandler
	sseHandler        *mcpsdk.SSEHandler
}

// NewServer creates a new Server instance and registers the provided domain handlers.
func NewServer(handlers ...DomainHandler) *Server {
	impl := &mcpsdk.Implementation{
		Name:    "khi",
		Version: constants.VERSION,
	}
	mcpServer := mcpsdk.NewServer(impl, nil)
	s := &Server{
		mcpServer: mcpServer,
		streamableHandler: mcpsdk.NewStreamableHTTPHandler(func(*http.Request) *mcpsdk.Server {
			return mcpServer
		}, nil),
		sseHandler: mcpsdk.NewSSEHandler(func(*http.Request) *mcpsdk.Server {
			return mcpServer
		}, nil),
	}
	for _, h := range handlers {
		s.RegisterHandler(h)
	}
	return s
}

// RegisterHandler registers a domain handler to the underlying MCP server.
func (s *Server) RegisterHandler(handler DomainHandler) {
	handler.Register(s.mcpServer)
}

// HTTPHandler returns the http.Handler for the Streamable HTTP transport (/mcp).
func (s *Server) HTTPHandler() http.Handler {
	return s.streamableHandler
}

// SSEHandler returns the http.Handler for the SSE transport (/mcp/sse).
func (s *Server) SSEHandler() http.Handler {
	return s.sseHandler
}
