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
	"net/http/httputil"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	"github.com/GoogleCloudPlatform/khi/pkg/server/upload"
	"github.com/gin-gonic/gin"
)

func TestFileParameterUploadInitializer_UploadURLRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name        string
		basePath    string
		proxyPrefix string
	}{
		{
			name:     "empty base path",
			basePath: "",
		},
		{
			name:     "non-empty base path",
			basePath: "/khi",
		},
		{
			name:        "proxy adds a prefix to the root base path",
			proxyPrefix: "/gateway/khi",
		},
		{
			name:        "proxy adds a prefix to a non-empty base path",
			basePath:    "/khi",
			proxyPrefix: "/gateway",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			engine := coreinit.NewEngine(context.Background())
			ctx := engine.Context()

			tempDir := t.TempDir()
			uploadStore := upload.NewUploadFileStore(upload.NewLocalUploadFileStoreProvider(filepath.Join(tempDir, "store")))
			jobMode := false
			uploadFolder := filepath.Join(tempDir, "upload")
			host := "127.0.0.1"
			port := 8080
			maxUploadSize := 1024
			ginEngine := gin.New()
			var router gin.IRouter = ginEngine.Group(tc.basePath)
			publicURL := ""
			var proxyServer *httptest.Server
			if tc.proxyPrefix != "" {
				upstream := httptest.NewServer(ginEngine)
				t.Cleanup(upstream.Close)
				upstreamURL, err := url.Parse(upstream.URL)
				if err != nil {
					t.Fatalf("failed to parse upstream URL: %v", err)
				}
				proxy := httputil.NewSingleHostReverseProxy(upstreamURL)
				proxyServer = httptest.NewTLSServer(http.StripPrefix(tc.proxyPrefix, proxy))
				t.Cleanup(proxyServer.Close)
				publicURL = proxyServer.URL + tc.proxyPrefix + tc.basePath + "/"
			}
			coreinit.Set(ctx, JobParametersKey, &parameters.JobParameters{JobMode: &jobMode})
			coreinit.Set(ctx, UploadStoreKey, uploadStore)
			coreinit.Set(ctx, BasePathKey, tc.basePath)
			coreinit.Set(ctx, CommonParametersKey, &parameters.CommonParameters{UploadFileStoreFolder: &uploadFolder})
			coreinit.Set(ctx, ServerParametersKey, &parameters.ServerParameters{
				Host:                     &host,
				Port:                     &port,
				PublicURL:                &publicURL,
				MaxUploadFileSizeInBytes: &maxUploadSize,
			})
			coreinit.Set(ctx, GinRouterKey, router)

			if err := FileParameterUploadInitializer.Init(ctx); err != nil {
				t.Fatalf("FileParameterUploadInitializer.Init() failed: %v", err)
			}

			issuer, found := coreinit.Get(ctx, UploadURLIssuerKey)
			if !found {
				t.Fatal("coreinit.Get(UploadURLIssuerKey) found = false, want true")
			}
			uploadToken := uploadStore.GetUploadToken("upload-token-1", nil, "field-1")
			grant := issuer.Issue(uploadToken.GetID(), "field-1")
			wantURLBase := "http://127.0.0.1:8080" + tc.basePath + "/api/v1/file-upload/"
			if proxyServer != nil {
				wantURLBase = proxyServer.URL + tc.proxyPrefix + tc.basePath + "/api/v1/file-upload/"
			}
			if !strings.HasPrefix(grant.URL, wantURLBase) {
				t.Fatalf("issued URL = %q, want prefix %q", grant.URL, wantURLBase)
			}

			// Proxy uploads must use the issued URL so an incorrect external host or path fails the request.
			if proxyServer != nil {
				req, err := http.NewRequest(http.MethodPut, grant.URL, strings.NewReader("hello"))
				if err != nil {
					t.Fatalf("failed to create upload request: %v", err)
				}
				res, err := proxyServer.Client().Do(req)
				if err != nil {
					t.Fatalf("PUT %s failed: %v", grant.URL, err)
				}
				defer res.Body.Close()
				body, err := io.ReadAll(res.Body)
				if err != nil {
					t.Fatalf("failed to read upload response: %v", err)
				}
				if res.StatusCode != http.StatusOK {
					t.Errorf("PUT %s returned %d, want %d: %s", grant.URL, res.StatusCode, http.StatusOK, body)
				}
				return
			}

			req := httptest.NewRequest(http.MethodPut, grant.URL, strings.NewReader("hello"))
			rec := httptest.NewRecorder()
			ginEngine.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("PUT %s returned %d, want %d: %s", grant.URL, rec.Code, http.StatusOK, rec.Body.String())
			}
		})
	}
}

func TestUploadURLBase(t *testing.T) {
	testCases := []struct {
		name      string
		host      string
		port      int
		basePath  string
		publicURL string
		want      string
	}{
		{
			name: "loopback IPv4 host is used as is",
			host: "127.0.0.1",
			port: 8080,
			want: "http://127.0.0.1:8080/api/v1/file-upload/",
		},
		{
			name: "IPv4 wildcard host is replaced with loopback",
			host: "0.0.0.0",
			port: 8080,
			want: "http://127.0.0.1:8080/api/v1/file-upload/",
		},
		{
			name: "IPv6 wildcard host is replaced with loopback",
			host: "::",
			port: 8080,
			want: "http://127.0.0.1:8080/api/v1/file-upload/",
		},
		{
			name: "empty host is replaced with loopback",
			host: "",
			port: 8080,
			want: "http://127.0.0.1:8080/api/v1/file-upload/",
		},
		{
			name: "IPv6 host is bracketed",
			host: "::1",
			port: 8080,
			want: "http://[::1]:8080/api/v1/file-upload/",
		},
		{
			name:     "base path is placed before the upload path",
			host:     "localhost",
			port:     9000,
			basePath: "/khi",
			want:     "http://localhost:9000/khi/api/v1/file-upload/",
		},
		{
			name:      "public URL overrides the listen address",
			host:      "0.0.0.0",
			port:      8080,
			publicURL: "https://khi.example.com",
			want:      "https://khi.example.com/api/v1/file-upload/",
		},
		{
			name:      "public URL includes the complete external base path",
			host:      "127.0.0.1",
			port:      8080,
			basePath:  "/internal/khi",
			publicURL: "https://khi.example.com/proxy/khi",
			want:      "https://khi.example.com/proxy/khi/api/v1/file-upload/",
		},
		{
			name:      "public URL root overrides a non-empty server base path",
			host:      "127.0.0.1",
			port:      8080,
			basePath:  "/khi",
			publicURL: "http://khi.example.com:9000/",
			want:      "http://khi.example.com:9000/api/v1/file-upload/",
		},
		{
			name:      "public URL trailing slashes are removed",
			publicURL: "https://khi.example.com/khi///",
			want:      "https://khi.example.com/khi/api/v1/file-upload/",
		},
		{
			name:      "public URL preserves escaped path segments",
			publicURL: "https://khi.example.com/team%20one/khi/",
			want:      "https://khi.example.com/team%20one/khi/api/v1/file-upload/",
		},
		{
			name:      "public URL supports IPv6 and a custom port",
			publicURL: "http://[2001:db8::1]:9000/khi",
			want:      "http://[2001:db8::1]:9000/khi/api/v1/file-upload/",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := uploadURLBase(&parameters.ServerParameters{
				Host:      &tc.host,
				Port:      &tc.port,
				PublicURL: &tc.publicURL,
			}, tc.basePath)
			if got != tc.want {
				t.Errorf("uploadURLBase() = %q, want %q", got, tc.want)
			}
		})
	}
}
