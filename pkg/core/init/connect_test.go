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

package coreinit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
)

func TestRegisterConnectServiceHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name         string
		basePath     string
		servicePath  string
		requestPath  string
		wantStatus   int
		wantBody     string
		wantNotFound bool
	}{
		{
			name:        "handles request without basePath",
			basePath:    "",
			servicePath: "/api.v1.TestService/",
			requestPath: "/api.v1.TestService/TestMethod",
			wantStatus:  http.StatusOK,
			wantBody:    "ok",
		},
		{
			name:        "handles request with basePath",
			basePath:    "/custom/prefix",
			servicePath: "/api.v1.TestService/",
			requestPath: "/custom/prefix/api.v1.TestService/TestMethod",
			wantStatus:  http.StatusOK,
			wantBody:    "ok",
		},
		{
			name:        "handles request with basePath having trailing slash",
			basePath:    "/custom/prefix/",
			servicePath: "/api.v1.TestService/",
			requestPath: "/custom/prefix/api.v1.TestService/TestMethod",
			wantStatus:  http.StatusOK,
			wantBody:    "ok",
		},
		{
			name:         "returns 404 for unmapped path",
			basePath:     "",
			servicePath:  "/api.v1.TestService/",
			requestPath:  "/api.v1.OtherService/TestMethod",
			wantNotFound: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			engine := gin.New()
			var router gin.IRouter = engine
			if tc.basePath != "" {
				router = engine.Group(tc.basePath)
			}

			mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("ok"))
			})

			RegisterConnectServiceHandler(router, tc.basePath, tc.servicePath, mockHandler)

			req := httptest.NewRequest(http.MethodPost, tc.requestPath, nil)
			w := httptest.NewRecorder()
			engine.ServeHTTP(w, req)

			if tc.wantNotFound {
				if diff := cmp.Diff(http.StatusNotFound, w.Code); diff != "" {
					t.Errorf("status code mismatch (-want +got):\n%s", diff)
				}
				return
			}

			if diff := cmp.Diff(tc.wantStatus, w.Code); diff != "" {
				t.Errorf("status code mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantBody, w.Body.String()); diff != "" {
				t.Errorf("body mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
