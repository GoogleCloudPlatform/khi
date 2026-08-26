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
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/core/inspection/logger"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil"
	"github.com/gin-gonic/gin"
)

func TestKHIServer_EndpointExistsWithConfigs(t *testing.T) {
	testCases := []struct {
		name           string
		serverBasePath string
		requestMethod  string
		requestPath    string
		wantCode       int
	}{
		{
			name:          "session route should serve the static resource",
			requestMethod: "GET",
			requestPath:   "/session/100",
			wantCode:      200,
		},
		{
			name:          "static resource must be served",
			requestMethod: "GET",
			requestPath:   "/test.html",
			wantCode:      200,
		},
		{
			name:           "static resource must be served with server base path",
			serverBasePath: "/custom/base/path/foo",
			requestMethod:  "GET",
			requestPath:    "/custom/base/path/foo/test.html",
			wantCode:       200,
		},
		{
			name:           "session route should serve the static resource with custom server base path",
			serverBasePath: "/custom/base/path/foo",
			requestMethod:  "GET",
			requestPath:    "/custom/base/path/foo/session/100",
			wantCode:       200,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger.InitGlobalKHILogger()
			defer testutil.MustPlaceTemporalFile(fmt.Sprintf("%s/test.html", embeddedStaticFolderPath), "")()
			recorer := httptest.NewRecorder()
			engine := gin.New()
			router := engine.Group(tc.serverBasePath)
			SetupFrontendMiddleware(engine, tc.serverBasePath, embeddedStaticFolderPath)
			SetupFrontendRoutes(router, embeddedStaticFolderPath)
			req, _ := http.NewRequest(tc.requestMethod, tc.requestPath, bytes.NewReader([]byte{}))
			engine.ServeHTTP(recorer, req)
			if recorer.Code != tc.wantCode {
				t.Errorf("got response code %d, want %d", recorer.Code, tc.wantCode)
			}
		})
	}
}

func TestKHIServerRedirects(t *testing.T) {
	testCases := []struct {
		name           string
		serverBasePath string
		requestMethod  string
		requestPath    string
		wantCode       int
		redirectTo     string
	}{
		{
			name:          "the root path should be redirected to the default session path",
			requestMethod: "GET",
			requestPath:   "/",
			redirectTo:    "/session/0",
			wantCode:      302,
		},
		{
			name:           "the root path should be redirected to the default session path with custom server base path",
			serverBasePath: "/custom/base/path",
			requestMethod:  "GET",
			requestPath:    "/custom/base/path/",
			redirectTo:     "/custom/base/path/session/0",
			wantCode:       302,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			logger.InitGlobalKHILogger()
			recorer := httptest.NewRecorder()
			engine := gin.New()
			router := engine.Group(tc.serverBasePath)
			SetupFrontendMiddleware(engine, tc.serverBasePath, "dist")
			SetupFrontendRoutes(router, "dist")
			req, _ := http.NewRequest(tc.requestMethod, tc.requestPath, bytes.NewReader([]byte{}))
			engine.ServeHTTP(recorer, req)
			if recorer.Code != tc.wantCode {
				t.Errorf("got response code %d, want %d", recorer.Code, tc.wantCode)
			}
			gotRedirectTo := recorer.Result().Header.Get("Location")
			if gotRedirectTo != tc.redirectTo {
				t.Errorf("got redirect to %s, want %s", gotRedirectTo, tc.redirectTo)
			}
		})
	}
}
