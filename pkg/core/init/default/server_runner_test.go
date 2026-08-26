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

	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
)

func TestServerRunnerInitializer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name    string
		jobMode bool
	}{
		{
			name:    "skips server registration when in job mode",
			jobMode: true,
		},
		{
			name:    "registers server runner when not in job mode",
			jobMode: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			engine := coreinit.NewEngine(context.Background())
			ctx := engine.Context()

			coreinit.Set(ctx, JobParametersKey, &parameters.JobParameters{
				JobMode: &tc.jobMode,
			})
			host := "127.0.0.1"
			port := 8080
			coreinit.Set(ctx, ServerParametersKey, &parameters.ServerParameters{
				Host: &host,
				Port: &port,
			})
			noColor := true
			coreinit.Set(ctx, DebugParametersKey, &parameters.DebugParameters{
				NoColor: &noColor,
			})
			coreinit.Set(ctx, GinEngineKey, gin.New())

			err := ServerRunnerInitializer.Init(ctx)
			if err != nil {
				t.Fatalf("ServerRunnerInitializer.Init() failed: %v", err)
			}
		})
	}
}

func TestServerRunnerHTTP2AndHTTP1Support(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ginEngine := gin.New()
	ginEngine.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	protocols := &http.Protocols{}
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)

	ts := httptest.NewUnstartedServer(ginEngine)
	ts.Config.Protocols = protocols
	ts.Start()
	defer ts.Close()

	h2Protocols := &http.Protocols{}
	h2Protocols.SetUnencryptedHTTP2(true)

	testCases := []struct {
		name      string
		client    *http.Client
		wantProto string
	}{
		{
			name: "standard HTTP/1.1 client request",
			client: &http.Client{
				Transport: &http.Transport{},
			},
			wantProto: "HTTP/1.1",
		},
		{
			name: "HTTP/2 cleartext (h2c) client request",
			client: &http.Client{
				Transport: &http.Transport{
					Protocols: h2Protocols,
				},
			},
			wantProto: "HTTP/2.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := tc.client.Get(ts.URL + "/ping")
			if err != nil {
				t.Fatalf("failed to send request: %v", err)
			}
			defer resp.Body.Close()

			if diff := cmp.Diff(tc.wantProto, resp.Proto); diff != "" {
				t.Errorf("protocol mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(http.StatusOK, resp.StatusCode); diff != "" {
				t.Errorf("status code mismatch (-want +got):\n%s", diff)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}
			if diff := cmp.Diff("pong", string(body)); diff != "" {
				t.Errorf("response body mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
