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
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/server/mcp/md"
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

func TestKHIInvestigationSkillToolsAndResources(t *testing.T) {
	skillDir := filepath.Join("..", "..", "..", "plugins", "khi", "skills", "khi-investigation")
	skillPath := filepath.Join(skillDir, "SKILL.md")
	skillBytes, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) failed: %v", skillPath, err)
	}

	server := NewServer(
		NewInspectionHandler(nil, nil),
		NewWorkbenchHandler(nil, &fakeTimelineImageRenderer{}, nil),
	)
	ts := httptest.NewServer(server.HTTPHandler())
	t.Cleanup(ts.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	session := connectClient(t, ctx, &mcpsdk.StreamableClientTransport{Endpoint: ts.URL})

	toolsRes, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("session.ListTools() failed: %v", err)
	}
	var wantTools []string
	for _, tl := range toolsRes.Tools {
		wantTools = append(wantTools, tl.Name)
	}
	slices.Sort(wantTools)

	resourcesRes, err := session.ListResources(ctx, nil)
	if err != nil {
		t.Fatalf("session.ListResources() failed: %v", err)
	}
	templatesRes, err := session.ListResourceTemplates(ctx, nil)
	if err != nil {
		t.Fatalf("session.ListResourceTemplates() failed: %v", err)
	}
	var wantResources []string
	for _, r := range resourcesRes.Resources {
		wantResources = append(wantResources, r.URI)
	}
	for _, rt := range templatesRes.ResourceTemplates {
		wantResources = append(wantResources, rt.URITemplate)
	}
	slices.Sort(wantResources)

	toolNamePattern := regexp.MustCompile(`^[a-z]+(_[a-z]+)+$`)
	var gotTools, gotResources, gotRefs []string
	for _, span := range md.ExtractInlineCodeSpans(string(skillBytes)) {
		switch {
		case toolNamePattern.MatchString(span.Text):
			gotTools = append(gotTools, span.Text)
		case strings.HasPrefix(span.Text, "khi://"):
			gotResources = append(gotResources, span.Text)
		case strings.HasPrefix(span.Text, "references/"):
			gotRefs = append(gotRefs, span.Text)
		}
	}
	slices.Sort(gotTools)
	gotTools = slices.Compact(gotTools)
	slices.Sort(gotResources)
	gotResources = slices.Compact(gotResources)
	slices.Sort(gotRefs)
	gotRefs = slices.Compact(gotRefs)

	if diff := cmp.Diff(wantTools, gotTools); diff != "" {
		t.Errorf("SKILL.md tool names mismatch against registered MCP tools (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(wantResources, gotResources); diff != "" {
		t.Errorf("SKILL.md resource URIs mismatch against registered MCP resources (-want +got):\n%s", diff)
	}
	if len(gotRefs) == 0 {
		t.Errorf("SKILL.md references no files under references/")
	}
	for _, ref := range gotRefs {
		refPath := filepath.Join(skillDir, filepath.FromSlash(ref))
		if _, err := os.Stat(refPath); err != nil {
			t.Errorf("referenced file %q does not exist at %q: %v", ref, refPath, err)
		}
	}
}
