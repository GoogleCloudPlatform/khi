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

package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
)

// TestHostValidationMiddleware verifies that HostValidationMiddleware enforces Host header allowlisting, ignores X-Forwarded-Host, returns a static 403 body, and aborts disallowed requests.
func TestHostValidationMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name                 string
		useNilAllowlist      bool
		host                 string
		xForwardedHost       string
		wantStatus           int
		wantBody             string
		wantDownstreamCalled bool
	}{
		// Allows loopback Host headers and passes control to the downstream handler.
		{
			name:                 "allows loopback host",
			host:                 "localhost:8080",
			wantStatus:           http.StatusOK,
			wantBody:             "ok",
			wantDownstreamCalled: true,
		},
		// Fails closed with 403 even for loopback Host headers when allowlist is nil.
		{
			name:                 "fails closed when allowlist is nil",
			useNilAllowlist:      true,
			host:                 "localhost:8080",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: host not allowed",
			wantDownstreamCalled: false,
		},
		// Rejects disallowed hosts with 403 and static body without invoking downstream handlers.
		{
			name:                 "rejects foreign host without reflecting host in body",
			host:                 "unlisted.example.net",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: host not allowed",
			wantDownstreamCalled: false,
		},
		// Ignores X-Forwarded-Host headers when validating the HTTP Host header.
		{
			name:                 "ignores X-Forwarded-Host",
			host:                 "unlisted.example.net",
			xForwardedHost:       "localhost:8080",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: host not allowed",
			wantDownstreamCalled: false,
		},
		// Rejects requests with an empty Host header.
		{
			name:                 "rejects empty host",
			host:                 "",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: host not allowed",
			wantDownstreamCalled: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var allowlist *HostAllowlist
			if !tc.useNilAllowlist {
				var err error
				allowlist, err = NewHostAllowlist("127.0.0.1", "")
				if err != nil {
					t.Fatalf("NewHostAllowlist() unexpected error: %v", err)
				}
			}
			downstreamCalled := false
			engine := gin.New()
			engine.Use(HostValidationMiddleware(allowlist))
			engine.GET("/test", func(ctx *gin.Context) {
				downstreamCalled = true
				ctx.String(http.StatusOK, "ok")
			})
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Host = tc.host
			if tc.xForwardedHost != "" {
				req.Header.Set("X-Forwarded-Host", tc.xForwardedHost)
			}
			rec := httptest.NewRecorder()

			engine.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if diff := cmp.Diff(tc.wantBody, rec.Body.String()); diff != "" {
				t.Errorf("response body mismatch (-want +got):\n%s", diff)
			}
			if downstreamCalled != tc.wantDownstreamCalled {
				t.Errorf("downstreamCalled = %v, want %v", downstreamCalled, tc.wantDownstreamCalled)
			}
		})
	}
}

// TestCrossOriginProtectionMiddleware verifies cross-origin rejection for unsafe browser requests while permitting safe methods, same-origin requests, and non-browser clients.
func TestCrossOriginProtectionMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name                 string
		method               string
		host                 string
		secFetchSite         string
		origin               string
		wantStatus           int
		wantBody             string
		wantDownstreamCalled bool
	}{
		// Rejects POST requests with Sec-Fetch-Site: cross-site and returns a static 403 body.
		{
			name:                 "rejects cross-site POST",
			method:               http.MethodPost,
			host:                 "127.0.0.1:8080",
			secFetchSite:         "cross-site",
			origin:               "https://other.example.net",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: cross-origin request rejected",
			wantDownstreamCalled: false,
		},
		// Rejects PUT requests with Sec-Fetch-Site: cross-site.
		{
			name:                 "rejects cross-site PUT",
			method:               http.MethodPut,
			host:                 "127.0.0.1:8080",
			secFetchSite:         "cross-site",
			origin:               "https://other.example.net",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: cross-origin request rejected",
			wantDownstreamCalled: false,
		},
		// Rejects PATCH requests with Sec-Fetch-Site: cross-site.
		{
			name:                 "rejects cross-site PATCH",
			method:               http.MethodPatch,
			host:                 "127.0.0.1:8080",
			secFetchSite:         "cross-site",
			origin:               "https://other.example.net",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: cross-origin request rejected",
			wantDownstreamCalled: false,
		},
		// Rejects DELETE requests with Sec-Fetch-Site: cross-site.
		{
			name:                 "rejects cross-site DELETE",
			method:               http.MethodDelete,
			host:                 "127.0.0.1:8080",
			secFetchSite:         "cross-site",
			origin:               "https://other.example.net",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: cross-origin request rejected",
			wantDownstreamCalled: false,
		},
		// Rejects unsafe requests from another port or subdomain marked Sec-Fetch-Site: same-site.
		{
			name:                 "rejects same-site cross-port POST",
			method:               http.MethodPost,
			host:                 "localhost:8080",
			secFetchSite:         "same-site",
			origin:               "http://localhost:3000",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: cross-origin request rejected",
			wantDownstreamCalled: false,
		},
		// Rejects legacy browser POST requests lacking Sec-Fetch-Site when Origin mismatches Host.
		{
			name:                 "rejects legacy POST with mismatched Origin",
			method:               http.MethodPost,
			host:                 "localhost:8080",
			origin:               "http://other.example.net",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: cross-origin request rejected",
			wantDownstreamCalled: false,
		},
		// Rejects legacy browser POST requests lacking Sec-Fetch-Site when Origin is null.
		{
			name:                 "rejects legacy POST with null Origin",
			method:               http.MethodPost,
			host:                 "localhost:8080",
			origin:               "null",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: cross-origin request rejected",
			wantDownstreamCalled: false,
		},
		// Allows safe GET requests even when Sec-Fetch-Site is cross-site (e.g. OAuth callbacks).
		{
			name:                 "allows cross-site GET",
			method:               http.MethodGet,
			host:                 "127.0.0.1:8080",
			secFetchSite:         "cross-site",
			origin:               "https://accounts.google.com",
			wantStatus:           http.StatusOK,
			wantBody:             "ok",
			wantDownstreamCalled: true,
		},
		// Allows safe HEAD requests even when Sec-Fetch-Site is cross-site.
		{
			name:                 "allows cross-site HEAD",
			method:               http.MethodHead,
			host:                 "127.0.0.1:8080",
			secFetchSite:         "cross-site",
			wantStatus:           http.StatusOK,
			wantBody:             "ok",
			wantDownstreamCalled: true,
		},
		// Allows safe OPTIONS requests even when Sec-Fetch-Site is cross-site.
		{
			name:                 "allows cross-site OPTIONS",
			method:               http.MethodOptions,
			host:                 "127.0.0.1:8080",
			secFetchSite:         "cross-site",
			wantStatus:           http.StatusOK,
			wantBody:             "ok",
			wantDownstreamCalled: true,
		},
		// Allows non-browser POST requests that omit both Sec-Fetch-Site and Origin.
		{
			name:                 "allows non-browser POST without origin headers",
			method:               http.MethodPost,
			host:                 "127.0.0.1:8080",
			wantStatus:           http.StatusOK,
			wantBody:             "ok",
			wantDownstreamCalled: true,
		},
		// Allows same-origin browser POST requests including dev proxy setups where Origin differs from proxied Host.
		{
			name:                 "allows same-origin POST with Angular dev proxy Origin",
			method:               http.MethodPost,
			host:                 "127.0.0.1:8080",
			secFetchSite:         "same-origin",
			origin:               "http://localhost:4200",
			wantStatus:           http.StatusOK,
			wantBody:             "ok",
			wantDownstreamCalled: true,
		},
		// Allows browser-initiated direct POST requests with Sec-Fetch-Site: none.
		{
			name:                 "allows POST with Sec-Fetch-Site none",
			method:               http.MethodPost,
			host:                 "127.0.0.1:8080",
			secFetchSite:         "none",
			wantStatus:           http.StatusOK,
			wantBody:             "ok",
			wantDownstreamCalled: true,
		},
		// Allows legacy browser POST requests lacking Sec-Fetch-Site when Origin host matches Host.
		{
			name:                 "allows legacy POST with matching Origin",
			method:               http.MethodPost,
			host:                 "localhost:8080",
			origin:               "http://localhost:8080",
			wantStatus:           http.StatusOK,
			wantBody:             "ok",
			wantDownstreamCalled: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			downstreamCalled := false
			engine := gin.New()
			engine.Use(CrossOriginProtectionMiddleware())
			engine.Any("/test", func(ctx *gin.Context) {
				downstreamCalled = true
				ctx.String(http.StatusOK, "ok")
			})
			req := httptest.NewRequest(tc.method, "/test", nil)
			req.Host = tc.host
			if tc.secFetchSite != "" {
				req.Header.Set("Sec-Fetch-Site", tc.secFetchSite)
			}
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			rec := httptest.NewRecorder()

			engine.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if gotBody := rec.Body.String(); gotBody != tc.wantBody {
				t.Errorf("body = %q, want %q", gotBody, tc.wantBody)
			}
			if downstreamCalled != tc.wantDownstreamCalled {
				t.Errorf("downstreamCalled = %v, want %v", downstreamCalled, tc.wantDownstreamCalled)
			}
		})
	}
}

// TestMCPOriginProtectionMiddleware verifies that MCPOriginProtectionMiddleware blocks cross-origin and malformed Origin browser requests including GET/HEAD while permitting non-browser MCP clients and same-origin requests.
func TestMCPOriginProtectionMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name                 string
		method               string
		host                 string
		secFetchSite         string
		origin               string
		wantStatus           int
		wantBody             string
		wantDownstreamCalled bool
	}{
		// Allows non-browser MCP GET requests omitting both Sec-Fetch-Site and Origin.
		{
			name:                 "allows non-browser GET without origin headers",
			method:               http.MethodGet,
			host:                 "127.0.0.1:8080",
			wantStatus:           http.StatusOK,
			wantBody:             "ok",
			wantDownstreamCalled: true,
		},
		// Allows non-browser MCP POST requests omitting both Sec-Fetch-Site and Origin.
		{
			name:                 "allows non-browser POST without origin headers",
			method:               http.MethodPost,
			host:                 "127.0.0.1:8080",
			wantStatus:           http.StatusOK,
			wantBody:             "ok",
			wantDownstreamCalled: true,
		},
		// Allows GET requests marked Sec-Fetch-Site: same-origin.
		{
			name:                 "allows Sec-Fetch-Site same-origin GET",
			method:               http.MethodGet,
			host:                 "127.0.0.1:8080",
			secFetchSite:         "same-origin",
			wantStatus:           http.StatusOK,
			wantBody:             "ok",
			wantDownstreamCalled: true,
		},
		// Allows GET requests marked Sec-Fetch-Site: none.
		{
			name:                 "allows Sec-Fetch-Site none GET",
			method:               http.MethodGet,
			host:                 "127.0.0.1:8080",
			secFetchSite:         "none",
			wantStatus:           http.StatusOK,
			wantBody:             "ok",
			wantDownstreamCalled: true,
		},
		// Rejects cross-site GET requests to protect SSE and streamable MCP reads without reflecting Origin.
		{
			name:                 "rejects Sec-Fetch-Site cross-site GET",
			method:               http.MethodGet,
			host:                 "127.0.0.1:8080",
			secFetchSite:         "cross-site",
			origin:               "https://other.example.net",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: cross-origin request rejected",
			wantDownstreamCalled: false,
		},
		// Rejects cross-site HEAD requests to MCP endpoints.
		{
			name:                 "rejects Sec-Fetch-Site cross-site HEAD",
			method:               http.MethodHead,
			host:                 "127.0.0.1:8080",
			secFetchSite:         "cross-site",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: cross-origin request rejected",
			wantDownstreamCalled: false,
		},
		// Rejects same-site cross-port or cross-subdomain GET requests to MCP endpoints.
		{
			name:                 "rejects Sec-Fetch-Site same-site GET",
			method:               http.MethodGet,
			host:                 "localhost:8080",
			secFetchSite:         "same-site",
			origin:               "http://localhost:3000",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: cross-origin request rejected",
			wantDownstreamCalled: false,
		},
		// Allows legacy GET requests without Sec-Fetch-Site when HTTP Origin host matches Request.Host case-insensitively.
		{
			name:                 "allows matching http Origin case-insensitively",
			method:               http.MethodGet,
			host:                 "LOCALHOST:8080",
			origin:               "http://localhost:8080",
			wantStatus:           http.StatusOK,
			wantBody:             "ok",
			wantDownstreamCalled: true,
		},
		// Allows legacy GET requests without Sec-Fetch-Site when HTTPS Origin host matches Request.Host.
		{
			name:                 "allows matching https Origin",
			method:               http.MethodGet,
			host:                 "khi.example.com",
			origin:               "https://khi.example.com",
			wantStatus:           http.StatusOK,
			wantBody:             "ok",
			wantDownstreamCalled: true,
		},
		// Rejects GET requests without Sec-Fetch-Site when Origin points to a foreign host.
		{
			name:                 "rejects foreign Origin GET",
			method:               http.MethodGet,
			host:                 "127.0.0.1:8080",
			origin:               "http://other.example.net",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: cross-origin request rejected",
			wantDownstreamCalled: false,
		},
		// Rejects GET requests without Sec-Fetch-Site when Origin port differs from Request.Host.
		{
			name:                 "rejects cross-port Origin GET",
			method:               http.MethodGet,
			host:                 "127.0.0.1:8080",
			origin:               "http://127.0.0.1:3000",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: cross-origin request rejected",
			wantDownstreamCalled: false,
		},
		// Rejects GET requests with opaque Origin: null.
		{
			name:                 "rejects null Origin GET",
			method:               http.MethodGet,
			host:                 "127.0.0.1:8080",
			origin:               "null",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: cross-origin request rejected",
			wantDownstreamCalled: false,
		},
		// Rejects GET requests whose Origin uses a non-HTTP/HTTPS scheme even with a matching host.
		{
			name:                 "rejects non-http scheme Origin GET",
			method:               http.MethodGet,
			host:                 "127.0.0.1:8080",
			origin:               "ftp://127.0.0.1:8080",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: cross-origin request rejected",
			wantDownstreamCalled: false,
		},
		// Rejects GET requests with a syntactically malformed Origin header.
		{
			name:                 "rejects malformed Origin GET",
			method:               http.MethodGet,
			host:                 "127.0.0.1:8080",
			origin:               "://invalid",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: cross-origin request rejected",
			wantDownstreamCalled: false,
		},
		// Rejects an Origin header with an empty host even when Request.Host is empty.
		{
			name:                 "rejects empty origin host with empty request host",
			method:               http.MethodGet,
			host:                 "",
			origin:               "http://",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: cross-origin request rejected",
			wantDownstreamCalled: false,
		},
		// Rejects an Origin header containing userinfo.
		{
			name:                 "rejects userinfo in origin",
			method:               http.MethodGet,
			host:                 "127.0.0.1:8080",
			origin:               "http://user@127.0.0.1:8080",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: cross-origin request rejected",
			wantDownstreamCalled: false,
		},
		// Rejects an Origin header containing a URL path.
		{
			name:                 "rejects path in origin",
			method:               http.MethodGet,
			host:                 "127.0.0.1:8080",
			origin:               "http://127.0.0.1:8080/mcp",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: cross-origin request rejected",
			wantDownstreamCalled: false,
		},
		// Rejects an Origin header containing a query string.
		{
			name:                 "rejects query in origin",
			method:               http.MethodGet,
			host:                 "127.0.0.1:8080",
			origin:               "http://127.0.0.1:8080?foo=bar",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: cross-origin request rejected",
			wantDownstreamCalled: false,
		},
		// Rejects an Origin header containing a URL fragment.
		{
			name:                 "rejects fragment in origin",
			method:               http.MethodGet,
			host:                 "127.0.0.1:8080",
			origin:               "http://127.0.0.1:8080#frag",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: cross-origin request rejected",
			wantDownstreamCalled: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			downstreamCalled := false
			engine := gin.New()
			engine.Use(MCPOriginProtectionMiddleware())
			engine.Any("/mcp", func(ctx *gin.Context) {
				downstreamCalled = true
				ctx.String(http.StatusOK, "ok")
			})
			req := httptest.NewRequest(tc.method, "/mcp", nil)
			req.Host = tc.host
			if tc.secFetchSite != "" {
				req.Header.Set("Sec-Fetch-Site", tc.secFetchSite)
			}
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			rec := httptest.NewRecorder()

			engine.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if gotBody := rec.Body.String(); gotBody != tc.wantBody {
				t.Errorf("body = %q, want %q", gotBody, tc.wantBody)
			}
			if downstreamCalled != tc.wantDownstreamCalled {
				t.Errorf("downstreamCalled = %v, want %v", downstreamCalled, tc.wantDownstreamCalled)
			}
		})
	}
}
