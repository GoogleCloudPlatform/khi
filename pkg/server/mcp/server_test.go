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
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type testDomainHandler struct {
	toolName string
}

var _ DomainHandler = (*testDomainHandler)(nil)

func (h *testDomainHandler) Register(srv *mcpsdk.Server) {
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        h.toolName,
		Description: "test tool description",
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest, in struct{}) (*mcpsdk.CallToolResult, any, error) {
		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{
				&mcpsdk.TextContent{Text: "tool response"},
			},
		}, nil, nil
	})
}

func TestServerTransportsAndDomainHandler(t *testing.T) {
	if len(serverInstructions) > 500 {
		t.Errorf("len(serverInstructions) = %d, want <= 500", len(serverInstructions))
	}

	server := NewServer(&testDomainHandler{toolName: "test_tool"})

	mux := http.NewServeMux()
	mux.Handle("/mcp", server.HTTPHandler())
	mux.Handle("/mcp/sse", server.SSEHandler())

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	testCases := []struct {
		name          string
		makeTransport func(baseURL string) mcpsdk.Transport
	}{
		{
			name: "Streamable HTTP client transport (/mcp)",
			makeTransport: func(baseURL string) mcpsdk.Transport {
				return &mcpsdk.StreamableClientTransport{
					Endpoint: baseURL + "/mcp",
				}
			},
		},
		{
			name: "SSE client transport (/mcp/sse)",
			makeTransport: func(baseURL string) mcpsdk.Transport {
				return &mcpsdk.SSEClientTransport{
					Endpoint: baseURL + "/mcp/sse",
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			t.Cleanup(cancel)

			session := connectClient(t, ctx, tc.makeTransport(ts.URL))

			if got := session.InitializeResult().Instructions; got != serverInstructions {
				t.Errorf("InitializeResult().Instructions = %q, want %q", got, serverInstructions)
			}

			toolsRes, err := session.ListTools(ctx, nil)
			if err != nil {
				t.Fatalf("session.ListTools() failed: %v", err)
			}
			var gotNames []string
			for _, tl := range toolsRes.Tools {
				gotNames = append(gotNames, tl.Name)
			}
			if diff := cmp.Diff([]string{"test_tool"}, gotNames); diff != "" {
				t.Errorf("ListTools() names mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
