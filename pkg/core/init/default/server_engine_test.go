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
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"

	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	"github.com/GoogleCloudPlatform/khi/pkg/server/uploadurl"
	"github.com/GoogleCloudPlatform/khi/pkg/server/workbench"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil"
)

type fullStackMCPRequestSpec struct {
	method         string
	host           string
	xForwardedHost string
	secFetchSite   string
	origin         string
}

type fullStackMCPOutcome struct {
	Forbidden        bool
	ForbiddenBody    string
	AllowedReachable bool
}

// TestGinEngineInitializer_Init verifies job-mode skipping, valid engine initialization, and error handling for invalid AllowedHosts.
func TestGinEngineInitializer_Init(t *testing.T) {
	t.Cleanup(func() {
		gin.SetMode(gin.TestMode)
	})

	testCases := []struct {
		name          string
		jobMode       bool
		listenHost    string
		allowedHosts  string
		wantErr       bool
		wantHasEngine bool
	}{
		// Skips Gin engine creation when JobMode is enabled.
		{
			name:          "skips initialization in job mode",
			jobMode:       true,
			listenHost:    "127.0.0.1",
			allowedHosts:  "",
			wantErr:       false,
			wantHasEngine: false,
		},
		// Creates and registers GinEngineKey when ServerParameters has valid AllowedHosts.
		{
			name:          "initializes engine with valid AllowedHosts",
			jobMode:       false,
			listenHost:    "127.0.0.1",
			allowedHosts:  "khi.example.com",
			wantErr:       false,
			wantHasEngine: true,
		},
		// Fails initialization without setting GinEngineKey when AllowedHosts contains an invalid pattern.
		{
			name:          "returns error on invalid AllowedHosts",
			jobMode:       false,
			listenHost:    "127.0.0.1",
			allowedHosts:  "http://example.com",
			wantErr:       true,
			wantHasEngine: false,
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
				JobMode: testutil.P(tc.jobMode),
			})
			coreinit.Set(ctx, DebugParametersKey, &parameters.DebugParameters{
				Verbose: testutil.P(false),
			})
			coreinit.Set(ctx, ServerParametersKey, &parameters.ServerParameters{
				Host:         testutil.P(tc.listenHost),
				AllowedHosts: testutil.P(tc.allowedHosts),
			})

			err := GinEngineInitializer.Init(ctx)

			if (err != nil) != tc.wantErr {
				t.Fatalf("GinEngineInitializer.Init() error = %v, wantErr = %v", err, tc.wantErr)
			}
			ginEngine, hasEngine := coreinit.Get(ctx, GinEngineKey)
			if hasEngine != tc.wantHasEngine {
				t.Errorf("GinEngineKey present = %v, want %v", hasEngine, tc.wantHasEngine)
			}
			if tc.wantHasEngine && ginEngine == nil {
				t.Errorf("GinEngineKey value = nil, want non-nil *gin.Engine")
			}
		})
	}
}

// TestGinEngineInitializer_SecurityMiddleware verifies the global host-validation and cross-origin protection middleware chain attached by GinEngineInitializer.
func TestGinEngineInitializer_SecurityMiddleware(t *testing.T) {
	t.Cleanup(func() {
		gin.SetMode(gin.TestMode)
	})

	const (
		hostedAllowlist = "khi.example.com,*-dot-khi.example.com,10.2.0.2:443"
		appAllowlist    = "app.example.com,*-dot-app.example.com"
	)

	testCases := []struct {
		name                 string
		listenHost           string
		allowedHosts         string
		method               string
		path                 string
		hostHeader           string
		xForwardedHost       string
		secFetchSite         string
		origin               string
		contentType          string
		wantStatus           int
		wantBody             string
		wantDownstreamCalled bool
	}{
		// Rejects requests with an unlisted Host on 127.0.0.1 even when Sec-Fetch-Site is same-origin.
		{
			name:                 "rejects unlisted Host on 127.0.0.1",
			listenHost:           "127.0.0.1",
			allowedHosts:         "",
			method:               http.MethodPost,
			path:                 "/api/test",
			hostHeader:           "unlisted.example.net",
			secFetchSite:         "same-origin",
			origin:               "http://unlisted.example.net",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: host not allowed",
			wantDownstreamCalled: false,
		},
		// Rejects requests with an unlisted Host when listening on wildcard 0.0.0.0.
		{
			name:                 "rejects unlisted Host on 0.0.0.0",
			listenHost:           "0.0.0.0",
			allowedHosts:         "",
			method:               http.MethodPost,
			path:                 "/api/test",
			hostHeader:           "unlisted.example.net",
			secFetchSite:         "same-origin",
			origin:               "http://unlisted.example.net",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: host not allowed",
			wantDownstreamCalled: false,
		},
		// Enforces HostValidationMiddleware before CrossOriginProtectionMiddleware when both Host and Origin are rejected.
		{
			name:                 "host validation runs before cross-origin protection",
			listenHost:           "127.0.0.1",
			allowedHosts:         "",
			method:               http.MethodPost,
			path:                 "/api/test",
			hostHeader:           "unlisted.example.net",
			secFetchSite:         "cross-site",
			origin:               "https://other.example.net",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: host not allowed",
			wantDownstreamCalled: false,
		},
		// Rejects unregistered paths with 403 rather than 404 when Host is disallowed.
		{
			name:                 "NoRoute enforces host validation with 403",
			listenHost:           "127.0.0.1",
			allowedHosts:         "",
			method:               http.MethodGet,
			path:                 "/does-not-exist",
			hostHeader:           "unlisted.example.net",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: host not allowed",
			wantDownstreamCalled: false,
		},
		// Allows Cloud Shell Web Preview requests where the VM gateway rewrites Host to 127.0.0.1 and preserves Sec-Fetch-Site: same-origin.
		{
			name:                 "allows Cloud Shell gateway POST with rewritten loopback Host",
			listenHost:           "0.0.0.0",
			allowedHosts:         "",
			method:               http.MethodPost,
			path:                 "/api/test",
			hostHeader:           "127.0.0.1:8080",
			xForwardedHost:       "8080-cs-test.cloudshell.dev",
			secFetchSite:         "same-origin",
			origin:               "https://8080-cs-test.cloudshell.dev",
			wantStatus:           http.StatusOK,
			wantBody:             "ok",
			wantDownstreamCalled: true,
		},
		// Allows configured prefix-wildcard hosts in AllowedHosts.
		{
			name:                 "allows configured prefix wildcard host",
			listenHost:           "0.0.0.0",
			allowedHosts:         hostedAllowlist,
			method:               http.MethodGet,
			path:                 "/oauth/callback",
			hostHeader:           "v1-dot-default-dot-khi.example.com:443",
			wantStatus:           http.StatusOK,
			wantBody:             "ok",
			wantDownstreamCalled: true,
		},
		// Rejects subdomains against an exact AllowedHosts entry.
		{
			name:                 "rejects unmatched subdomain on exact allowlist entry",
			listenHost:           "0.0.0.0",
			allowedHosts:         hostedAllowlist,
			method:               http.MethodGet,
			path:                 "/oauth/callback",
			hostHeader:           "sub.khi.example.com",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: host not allowed",
			wantDownstreamCalled: false,
		},
		// Allows configured single-label prefix-wildcard hosts in AllowedHosts.
		{
			name:                 "allows single-label prefix wildcard host",
			listenHost:           "0.0.0.0",
			allowedHosts:         appAllowlist,
			method:               http.MethodGet,
			path:                 "/oauth/callback",
			hostHeader:           "beta-dot-app.example.com",
			wantStatus:           http.StatusOK,
			wantBody:             "ok",
			wantDownstreamCalled: true,
		},
		// Rejects dot-separated labels inside the * portion of prefix wildcard patterns.
		{
			name:                 "rejects dot inside prefix wildcard label",
			listenHost:           "0.0.0.0",
			allowedHosts:         appAllowlist,
			method:               http.MethodGet,
			path:                 "/oauth/callback",
			hostHeader:           "sub.beta-dot-app.example.com",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: host not allowed",
			wantDownstreamCalled: false,
		},
		// Rejects cross-site text/plain POST requests targeting a valid loopback Host with a static 403 body.
		{
			name:                 "rejects cross-site POST on valid loopback host",
			listenHost:           "127.0.0.1",
			allowedHosts:         "",
			method:               http.MethodPost,
			path:                 "/api/test",
			hostHeader:           "127.0.0.1:8080",
			secFetchSite:         "cross-site",
			origin:               "https://other.example.net",
			contentType:          "text/plain",
			wantStatus:           http.StatusForbidden,
			wantBody:             "Forbidden: cross-origin request rejected",
			wantDownstreamCalled: false,
		},
		// Allows safe cross-site GET requests such as OAuth callbacks targeting a valid loopback Host.
		{
			name:                 "allows cross-site GET OAuth callback on valid host",
			listenHost:           "127.0.0.1",
			allowedHosts:         "",
			method:               http.MethodGet,
			path:                 "/oauth/callback",
			hostHeader:           "127.0.0.1:8080",
			secFetchSite:         "cross-site",
			origin:               "https://accounts.google.com",
			wantStatus:           http.StatusOK,
			wantBody:             "ok",
			wantDownstreamCalled: true,
		},
		// Allows non-browser POST requests without origin headers targeting a valid loopback Host.
		{
			name:                 "allows non-browser POST on valid host",
			listenHost:           "127.0.0.1",
			allowedHosts:         "",
			method:               http.MethodPost,
			path:                 "/api/test",
			hostHeader:           "127.0.0.1:8080",
			wantStatus:           http.StatusOK,
			wantBody:             "ok",
			wantDownstreamCalled: true,
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
				JobMode: testutil.P(false),
			})
			coreinit.Set(ctx, DebugParametersKey, &parameters.DebugParameters{
				Verbose: testutil.P(false),
			})
			coreinit.Set(ctx, ServerParametersKey, &parameters.ServerParameters{
				Host:         testutil.P(tc.listenHost),
				AllowedHosts: testutil.P(tc.allowedHosts),
			})
			if err := GinEngineInitializer.Init(ctx); err != nil {
				t.Fatalf("GinEngineInitializer.Init() unexpected error: %v", err)
			}
			ginEngine := coreinit.MustGet(ctx, GinEngineKey)
			downstreamCalled := false
			handler := func(c *gin.Context) {
				downstreamCalled = true
				c.String(http.StatusOK, "ok")
			}
			ginEngine.POST("/api/test", handler)
			ginEngine.GET("/oauth/callback", handler)

			req := httptest.NewRequest(tc.method, tc.path, nil)
			req.Host = tc.hostHeader
			setHeaderIfPresent(req.Header, "X-Forwarded-Host", tc.xForwardedHost)
			setHeaderIfPresent(req.Header, "Sec-Fetch-Site", tc.secFetchSite)
			setHeaderIfPresent(req.Header, "Origin", tc.origin)
			setHeaderIfPresent(req.Header, "Content-Type", tc.contentType)
			rec := httptest.NewRecorder()

			ginEngine.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d (body: %q)", rec.Code, tc.wantStatus, rec.Body.String())
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

// TestGinEngineAndMCPServerInitializer_SecurityIntegration verifies full-stack Host and Origin enforcement when GinEngineInitializer and MCPServerInitializer are wired together.
func TestGinEngineAndMCPServerInitializer_SecurityIntegration(t *testing.T) {
	t.Cleanup(func() {
		gin.SetMode(gin.TestMode)
	})

	testCases := []struct {
		name         string
		listenHost   string
		allowedHosts string
		path         string
		req          fullStackMCPRequestSpec
		want         fullStackMCPOutcome
	}{
		// Rejects GET requests with an unlisted Host on /mcp even when listening on wildcard 0.0.0.0.
		{
			name:       "wildcard 0.0.0.0 rejects unlisted Host GET on /mcp",
			listenHost: "0.0.0.0",
			path:       "/mcp",
			req: fullStackMCPRequestSpec{
				method:       http.MethodGet,
				host:         "unlisted.example.net",
				secFetchSite: "same-origin",
				origin:       "http://unlisted.example.net",
			},
			want: fullStackMCPOutcome{
				Forbidden:     true,
				ForbiddenBody: "Forbidden: host not allowed",
			},
		},
		// Rejects POST requests with an unlisted Host on /mcp even when listening on wildcard 0.0.0.0.
		{
			name:       "wildcard 0.0.0.0 rejects unlisted Host POST on /mcp",
			listenHost: "0.0.0.0",
			path:       "/mcp",
			req: fullStackMCPRequestSpec{
				method:       http.MethodPost,
				host:         "unlisted.example.net",
				secFetchSite: "same-origin",
				origin:       "http://unlisted.example.net",
			},
			want: fullStackMCPOutcome{
				Forbidden:     true,
				ForbiddenBody: "Forbidden: host not allowed",
			},
		},
		// Rejects GET requests with an unlisted Host on /mcp/sse even when listening on wildcard 0.0.0.0.
		{
			name:       "wildcard 0.0.0.0 rejects unlisted Host GET on /mcp/sse",
			listenHost: "0.0.0.0",
			path:       "/mcp/sse",
			req: fullStackMCPRequestSpec{
				method:       http.MethodGet,
				host:         "unlisted.example.net",
				secFetchSite: "same-origin",
				origin:       "http://unlisted.example.net",
			},
			want: fullStackMCPOutcome{
				Forbidden:     true,
				ForbiddenBody: "Forbidden: host not allowed",
			},
		},
		// Allows Cloud Shell gateway GET requests to /mcp when Host is rewritten to loopback and Sec-Fetch-Site is same-origin.
		{
			name:       "Cloud Shell gateway GET on /mcp allowed with loopback Host",
			listenHost: "0.0.0.0",
			path:       "/mcp",
			req: fullStackMCPRequestSpec{
				method:         http.MethodGet,
				host:           "127.0.0.1:8080",
				xForwardedHost: "8080-cs-test.cloudshell.dev",
				secFetchSite:   "same-origin",
				origin:         "https://8080-cs-test.cloudshell.dev",
			},
			want: fullStackMCPOutcome{AllowedReachable: true},
		},
		// Allows configured --allowed-hosts domains on /mcp over loopback connections without SDK localhost-protection interference.
		{
			name:         "allowlisted non-loopback Host on /mcp succeeds over loopback proxy",
			listenHost:   "127.0.0.1",
			allowedHosts: "khi.example.com",
			path:         "/mcp",
			req: fullStackMCPRequestSpec{
				method:       http.MethodGet,
				host:         "khi.example.com",
				secFetchSite: "same-origin",
				origin:       "https://khi.example.com",
			},
			want: fullStackMCPOutcome{AllowedReachable: true},
		},
		// Allows configured --allowed-hosts domains on /mcp/sse over loopback connections without SDK localhost-protection interference.
		{
			name:         "allowlisted non-loopback Host on /mcp/sse succeeds over loopback proxy",
			listenHost:   "127.0.0.1",
			allowedHosts: "khi.example.com",
			path:         "/mcp/sse",
			req: fullStackMCPRequestSpec{
				method:       http.MethodGet,
				host:         "khi.example.com",
				secFetchSite: "same-origin",
				origin:       "https://khi.example.com",
			},
			want: fullStackMCPOutcome{AllowedReachable: true},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ts := startFullStackMCPTestServer(t, tc.listenHost, tc.allowedHosts)
			req, cancel := newFullStackMCPRequest(t, ts.URL+tc.path, tc.req)

			resp, err := ts.Client().Do(req)
			if err != nil {
				cancel()
				t.Fatalf("client.Do() failed: %v", err)
			}
			got := readFullStackMCPOutcome(t, resp)
			_ = resp.Body.Close()
			cancel()

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("%s %s outcome mismatch (-want +got):\n%s", tc.req.method, tc.path, diff)
			}
		})
	}
}

// setHeaderIfPresent sets key to value on header when value is non-empty.
func setHeaderIfPresent(header http.Header, key string, value string) {
	if value != "" {
		header.Set(key, value)
	}
}

// startFullStackMCPTestServer wires GinEngineInitializer and MCPServerInitializer together behind a loopback httptest.Server.
func startFullStackMCPTestServer(t *testing.T, listenHost string, allowedHosts string) *httptest.Server {
	t.Helper()
	engine := coreinit.NewEngine(context.Background())
	t.Cleanup(func() {
		_ = engine.Terminate()
	})
	ctx := engine.Context()

	coreinit.Set(ctx, JobParametersKey, &parameters.JobParameters{
		JobMode: testutil.P(false),
	})
	coreinit.Set(ctx, DebugParametersKey, &parameters.DebugParameters{
		Verbose: testutil.P(false),
	})
	coreinit.Set(ctx, ServerParametersKey, &parameters.ServerParameters{
		Host:         testutil.P(listenHost),
		Port:         testutil.P(8080),
		AllowedHosts: testutil.P(allowedHosts),
	})
	coreinit.Set(ctx, BasePathKey, "")

	if err := GinEngineInitializer.Init(ctx); err != nil {
		t.Fatalf("GinEngineInitializer.Init() failed: %v", err)
	}
	ginEngine := coreinit.MustGet(ctx, GinEngineKey)
	coreinit.Set(ctx, GinRouterKey, gin.IRouter(ginEngine.Group("")))

	taskServer, err := coreinspection.NewServer(nil)
	if err != nil {
		t.Fatalf("coreinspection.NewServer() failed: %v", err)
	}
	coreinit.Set(ctx, InspectionTaskServerKey, taskServer)

	indexMgr := workbench.NewInspectionIndexManager(taskServer, t.TempDir())
	workbenchMgr := workbench.NewWorkbenchManager(taskServer, indexMgr, 3)
	coreinit.Set(ctx, WorkbenchManagerKey, workbenchMgr)
	coreinit.Set(ctx, UploadURLIssuerKey, uploadurl.NewIssuer("http://127.0.0.1:8080/api/v1/file-upload/", 1024, uploadurl.DefaultTTL))

	if err := MCPServerInitializer.Init(ctx); err != nil {
		t.Fatalf("MCPServerInitializer.Init() failed: %v", err)
	}

	ts := httptest.NewServer(ginEngine)
	t.Cleanup(ts.Close)
	return ts
}

// newFullStackMCPRequest builds a timed HTTP request with the Host and security headers from spec.
func newFullStackMCPRequest(t *testing.T, targetURL string, spec fullStackMCPRequestSpec) (*http.Request, context.CancelFunc) {
	t.Helper()
	reqCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	req, err := http.NewRequestWithContext(reqCtx, spec.method, targetURL, nil)
	if err != nil {
		cancel()
		t.Fatalf("http.NewRequestWithContext() failed: %v", err)
	}
	req.Host = spec.host
	setHeaderIfPresent(req.Header, "X-Forwarded-Host", spec.xForwardedHost)
	setHeaderIfPresent(req.Header, "Sec-Fetch-Site", spec.secFetchSite)
	setHeaderIfPresent(req.Header, "Origin", spec.origin)
	return req, cancel
}

// readFullStackMCPOutcome converts an HTTP response into a fullStackMCPOutcome without blocking on open SSE streams.
func readFullStackMCPOutcome(t *testing.T, resp *http.Response) fullStackMCPOutcome {
	t.Helper()
	if resp.StatusCode == http.StatusForbidden {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("io.ReadAll() failed: %v", err)
		}
		return fullStackMCPOutcome{
			Forbidden:     true,
			ForbiddenBody: string(body),
		}
	}
	return fullStackMCPOutcome{
		AllowedReachable: resp.StatusCode != http.StatusNotFound && resp.StatusCode < http.StatusInternalServerError,
	}
}
