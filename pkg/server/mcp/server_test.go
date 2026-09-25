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

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type testDomainHandler struct {
	toolName    string
	resourceURI string
	templateURI string
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

	srv.AddResource(&mcpsdk.Resource{
		URI:      h.resourceURI,
		Name:     "test resource",
		MIMEType: "text/plain",
	}, func(ctx context.Context, req *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
		return &mcpsdk.ReadResourceResult{
			Contents: []*mcpsdk.ResourceContents{
				{
					URI:      h.resourceURI,
					MIMEType: "text/plain",
					Text:     "resource content",
				},
			},
		}, nil
	})

	srv.AddResourceTemplate(&mcpsdk.ResourceTemplate{
		URITemplate: h.templateURI,
		Name:        "test template",
		MIMEType:    "text/plain",
	}, func(ctx context.Context, req *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
		return &mcpsdk.ReadResourceResult{
			Contents: []*mcpsdk.ResourceContents{
				{
					URI:      req.Params.URI,
					MIMEType: "text/plain",
					Text:     "template content",
				},
			},
		}, nil
	})
}

func TestServerTransportsAndDomainHandler(t *testing.T) {
	handler := &testDomainHandler{
		toolName:    "test_tool",
		resourceURI: "test://resource",
		templateURI: "test://template/{id}",
	}

	server := NewServer(handler)

	mux := http.NewServeMux()
	mux.Handle("/mcp", server.HTTPHandler())
	mux.Handle("/mcp/sse", server.SSEHandler())

	ts := httptest.NewServer(mux)
	defer ts.Close()

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
			defer cancel()

			client := mcpsdk.NewClient(&mcpsdk.Implementation{
				Name:    "test-client",
				Version: "1.0.0",
			}, nil)

			transport := tc.makeTransport(ts.URL)
			session, err := client.Connect(ctx, transport, nil)
			if err != nil {
				t.Fatalf("client.Connect() failed: %v", err)
			}
			defer session.Close()

			// Verify tool discovery
			toolsRes, err := session.ListTools(ctx, nil)
			if err != nil {
				t.Fatalf("session.ListTools() failed: %v", err)
			}
			if len(toolsRes.Tools) != 1 {
				t.Fatalf("len(toolsRes.Tools) = %d, want 1", len(toolsRes.Tools))
			}
			if toolsRes.Tools[0].Name != "test_tool" {
				t.Errorf("tool Name = %q, want %q", toolsRes.Tools[0].Name, "test_tool")
			}

			// Verify resource discovery
			resRes, err := session.ListResources(ctx, nil)
			if err != nil {
				t.Fatalf("session.ListResources() failed: %v", err)
			}
			if len(resRes.Resources) != 1 {
				t.Fatalf("len(resRes.Resources) = %d, want 1", len(resRes.Resources))
			}
			if resRes.Resources[0].URI != "test://resource" {
				t.Errorf("resource URI = %q, want %q", resRes.Resources[0].URI, "test://resource")
			}

			// Verify resource template discovery
			tmplRes, err := session.ListResourceTemplates(ctx, nil)
			if err != nil {
				t.Fatalf("session.ListResourceTemplates() failed: %v", err)
			}
			if len(tmplRes.ResourceTemplates) != 1 {
				t.Fatalf("len(tmplRes.ResourceTemplates) = %d, want 1", len(tmplRes.ResourceTemplates))
			}
			if tmplRes.ResourceTemplates[0].URITemplate != "test://template/{id}" {
				t.Errorf("template URITemplate = %q, want %q", tmplRes.ResourceTemplates[0].URITemplate, "test://template/{id}")
			}

			// Verify calling tool
			callRes, err := session.CallTool(ctx, &mcpsdk.CallToolParams{Name: "test_tool"})
			if err != nil {
				t.Fatalf("session.CallTool() failed: %v", err)
			}
			if len(callRes.Content) != 1 {
				t.Fatalf("len(callRes.Content) = %d, want 1", len(callRes.Content))
			}
			textContent, ok := callRes.Content[0].(*mcpsdk.TextContent)
			if !ok {
				t.Fatalf("callRes.Content[0] is %T, want *mcpsdk.TextContent", callRes.Content[0])
			}
			if textContent.Text != "tool response" {
				t.Errorf("callRes.Content[0].Text = %q, want %q", textContent.Text, "tool response")
			}

			// Verify reading resource
			readRes, err := session.ReadResource(ctx, &mcpsdk.ReadResourceParams{URI: "test://resource"})
			if err != nil {
				t.Fatalf("session.ReadResource() for resource failed: %v", err)
			}
			if len(readRes.Contents) != 1 {
				t.Fatalf("len(readRes.Contents) = %d, want 1", len(readRes.Contents))
			}
			if readRes.Contents[0].Text != "resource content" {
				t.Errorf("readRes.Contents[0].Text = %q, want %q", readRes.Contents[0].Text, "resource content")
			}

			// Verify reading resource template
			readTmplRes, err := session.ReadResource(ctx, &mcpsdk.ReadResourceParams{URI: "test://template/42"})
			if err != nil {
				t.Fatalf("session.ReadResource() for template failed: %v", err)
			}
			if len(readTmplRes.Contents) != 1 {
				t.Fatalf("len(readTmplRes.Contents) = %d, want 1", len(readTmplRes.Contents))
			}
			if readTmplRes.Contents[0].URI != "test://template/42" {
				t.Errorf("readTmplRes.Contents[0].URI = %q, want %q", readTmplRes.Contents[0].URI, "test://template/42")
			}
			if readTmplRes.Contents[0].Text != "template content" {
				t.Errorf("readTmplRes.Contents[0].Text = %q, want %q", readTmplRes.Contents[0].Text, "template content")
			}
		})
	}
}

func TestServerRegisterHandler(t *testing.T) {
	server := NewServer()

	handler := &testDomainHandler{
		toolName:    "dynamic_tool",
		resourceURI: "dynamic://resource",
		templateURI: "dynamic://template/{id}",
	}
	server.RegisterHandler(handler)

	mux := http.NewServeMux()
	mux.Handle("/mcp", server.HTTPHandler())

	ts := httptest.NewServer(mux)
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := mcpsdk.NewClient(&mcpsdk.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	transport := &mcpsdk.StreamableClientTransport{Endpoint: ts.URL + "/mcp"}
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("client.Connect() failed: %v", err)
	}
	defer session.Close()

	toolsRes, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("session.ListTools() failed: %v", err)
	}
	if len(toolsRes.Tools) != 1 {
		t.Fatalf("len(toolsRes.Tools) = %d, want 1", len(toolsRes.Tools))
	}
	if toolsRes.Tools[0].Name != "dynamic_tool" {
		t.Errorf("tool Name = %q, want %q", toolsRes.Tools[0].Name, "dynamic_tool")
	}
}
