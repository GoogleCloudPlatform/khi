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
		name     string
		basePath string
	}{
		{
			name:     "empty base path",
			basePath: "",
		},
		{
			name:     "non-empty base path",
			basePath: "/khi",
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
			coreinit.Set(ctx, JobParametersKey, &parameters.JobParameters{JobMode: &jobMode})
			coreinit.Set(ctx, UploadStoreKey, uploadStore)
			coreinit.Set(ctx, BasePathKey, tc.basePath)
			coreinit.Set(ctx, CommonParametersKey, &parameters.CommonParameters{UploadFileStoreFolder: &uploadFolder})
			coreinit.Set(ctx, ServerParametersKey, &parameters.ServerParameters{
				Host:                     &host,
				Port:                     &port,
				MaxUploadFileSizeInBytes: &maxUploadSize,
			})
			ginEngine := gin.New()
			var router gin.IRouter = ginEngine.Group(tc.basePath)
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

			// The issued URL must route to the upload URL handler registered under the same base path.
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
		name     string
		host     string
		port     int
		basePath string
		want     string
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
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := uploadURLBase(tc.host, tc.port, tc.basePath)
			if got != tc.want {
				t.Errorf("uploadURLBase(%q, %d, %q) = %q, want %q", tc.host, tc.port, tc.basePath, got, tc.want)
			}
		})
	}
}
