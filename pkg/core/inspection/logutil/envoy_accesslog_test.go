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

package logutil

import (
	"testing"

	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestEnvoyAccessLogTextParser_TryParse(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  *ParseStructuredLogResult
	}{
		{
			name:  "standard envoy log from user example with status 502 and no response flags",
			input: `[2026-08-10T08:50:55.958Z] "HEAD / HTTP/1.1" 502 - via_upstream - "-" 0 0 6 5 "-" "curl/8.21.0" "55667739-e394-4814-91b2-2cdd90744892" "136.68.163.124" "136.68.163.124:80" PassthroughCluster 10.4.1.8:33606 136.68.163.124:80 10.4.1.8:59778 - allow_any`,
			want: &ParseStructuredLogResult{
				Fields: map[string]any{
					OriginalMessageFieldKey:                       `[2026-08-10T08:50:55.958Z] "HEAD / HTTP/1.1" 502 - via_upstream - "-" 0 0 6 5 "-" "curl/8.21.0" "55667739-e394-4814-91b2-2cdd90744892" "136.68.163.124" "136.68.163.124:80" PassthroughCluster 10.4.1.8:33606 136.68.163.124:80 10.4.1.8:59778 - allow_any`,
					MainMessageStructuredFieldKey:                 "502 HEAD http://136.68.163.124/",
					SeverityStructuredFieldKey:                    inspectioncore_contract.SeverityError,
					EnvoyAccessLogTimestampFieldKey:               "2026-08-10T08:50:55.958Z",
					EnvoyAccessLogMethodFieldKey:                  "HEAD",
					EnvoyAccessLogPathFieldKey:                    "/",
					EnvoyAccessLogProtocolFieldKey:                "HTTP/1.1",
					EnvoyAccessLogStatusCodeFieldKey:              "502",
					EnvoyAccessLogResponseFlagsFieldKey:           "-",
					EnvoyAccessLogAuthorityFieldKey:               "136.68.163.124",
					EnvoyAccessLogUpstreamHostFieldKey:            "136.68.163.124:80",
					EnvoyAccessLogUpstreamClusterFieldKey:         "PassthroughCluster",
					EnvoyAccessLogDurationFieldKey:                "6",
					EnvoyAccessLogUserAgentFieldKey:               "curl/8.21.0",
					EnvoyAccessLogRequestIDFieldKey:               "55667739-e394-4814-91b2-2cdd90744892",
					EnvoyAccessLogUpstreamLocalAddressFieldKey:    "10.4.1.8:33606",
					EnvoyAccessLogDownstreamLocalAddressFieldKey:  "136.68.163.124:80",
					EnvoyAccessLogDownstreamRemoteAddressFieldKey: "10.4.1.8:59778",
					EnvoyAccessLogRouteNameFieldKey:               "allow_any",
				},
			},
		},
		{
			name:  "upstream connection failure (UF) with status 503",
			input: `[2026-08-10T08:50:55.958Z] "GET / HTTP/1.1" 503 UF - - "-" 0 0 6 - "-" "curl/8.21.0" "55667739-e394-4814-91b2-2cdd90744892" "10.4.0.5" "10.4.0.5:80" outbound|80||foo.default.svc.cluster.local - - 10.4.1.8:59778 - -`,
			want: &ParseStructuredLogResult{
				Fields: map[string]any{
					OriginalMessageFieldKey:                       `[2026-08-10T08:50:55.958Z] "GET / HTTP/1.1" 503 UF - - "-" 0 0 6 - "-" "curl/8.21.0" "55667739-e394-4814-91b2-2cdd90744892" "10.4.0.5" "10.4.0.5:80" outbound|80||foo.default.svc.cluster.local - - 10.4.1.8:59778 - -`,
					MainMessageStructuredFieldKey:                 "【Upstream connection failure(UF)】503 GET http://10.4.0.5/",
					SeverityStructuredFieldKey:                    inspectioncore_contract.SeverityError,
					EnvoyAccessLogTimestampFieldKey:               "2026-08-10T08:50:55.958Z",
					EnvoyAccessLogMethodFieldKey:                  "GET",
					EnvoyAccessLogPathFieldKey:                    "/",
					EnvoyAccessLogProtocolFieldKey:                "HTTP/1.1",
					EnvoyAccessLogStatusCodeFieldKey:              "503",
					EnvoyAccessLogResponseFlagsFieldKey:           "UF",
					EnvoyAccessLogAuthorityFieldKey:               "10.4.0.5",
					EnvoyAccessLogUpstreamHostFieldKey:            "10.4.0.5:80",
					EnvoyAccessLogUpstreamClusterFieldKey:         "outbound|80||foo.default.svc.cluster.local",
					EnvoyAccessLogDurationFieldKey:                "6",
					EnvoyAccessLogUserAgentFieldKey:               "curl/8.21.0",
					EnvoyAccessLogRequestIDFieldKey:               "55667739-e394-4814-91b2-2cdd90744892",
					EnvoyAccessLogDownstreamRemoteAddressFieldKey: "10.4.1.8:59778",
				},
			},
		},
		{
			name:  "successful 200 GET request",
			input: `[2026-08-10T08:50:55.958Z] "GET /api/v1/users HTTP/1.1" 200 - - - "-" 100 200 10 9 "-" "curl/8.21.0" "55667739-e394-4814-91b2-2cdd90744892" "example.com" "10.4.0.5:80" outbound|80||foo.default.svc.cluster.local 10.4.1.8:33606 10.4.0.5:80 10.4.1.8:59778 - default`,
			want: &ParseStructuredLogResult{
				Fields: map[string]any{
					OriginalMessageFieldKey:                       `[2026-08-10T08:50:55.958Z] "GET /api/v1/users HTTP/1.1" 200 - - - "-" 100 200 10 9 "-" "curl/8.21.0" "55667739-e394-4814-91b2-2cdd90744892" "example.com" "10.4.0.5:80" outbound|80||foo.default.svc.cluster.local 10.4.1.8:33606 10.4.0.5:80 10.4.1.8:59778 - default`,
					MainMessageStructuredFieldKey:                 "200 GET http://example.com/api/v1/users",
					SeverityStructuredFieldKey:                    inspectioncore_contract.SeverityInfo,
					EnvoyAccessLogTimestampFieldKey:               "2026-08-10T08:50:55.958Z",
					EnvoyAccessLogMethodFieldKey:                  "GET",
					EnvoyAccessLogPathFieldKey:                    "/api/v1/users",
					EnvoyAccessLogProtocolFieldKey:                "HTTP/1.1",
					EnvoyAccessLogStatusCodeFieldKey:              "200",
					EnvoyAccessLogResponseFlagsFieldKey:           "-",
					EnvoyAccessLogAuthorityFieldKey:               "example.com",
					EnvoyAccessLogUpstreamHostFieldKey:            "10.4.0.5:80",
					EnvoyAccessLogUpstreamClusterFieldKey:         "outbound|80||foo.default.svc.cluster.local",
					EnvoyAccessLogDurationFieldKey:                "10",
					EnvoyAccessLogUserAgentFieldKey:               "curl/8.21.0",
					EnvoyAccessLogRequestIDFieldKey:               "55667739-e394-4814-91b2-2cdd90744892",
					EnvoyAccessLogUpstreamLocalAddressFieldKey:    "10.4.1.8:33606",
					EnvoyAccessLogDownstreamLocalAddressFieldKey:  "10.4.0.5:80",
					EnvoyAccessLogDownstreamRemoteAddressFieldKey: "10.4.1.8:59778",
					EnvoyAccessLogRouteNameFieldKey:               "default",
				},
			},
		},
		{
			name:  "404 route not found (NR)",
			input: `[2026-08-10T08:50:55.958Z] "POST /invalid HTTP/1.1" 404 NR - - "-" 0 0 1 - "-" "curl/8.21.0" "55667739-e394-4814-91b2-2cdd90744892" "example.com" "-" - - - 10.4.1.8:59778 - -`,
			want: &ParseStructuredLogResult{
				Fields: map[string]any{
					OriginalMessageFieldKey:                       `[2026-08-10T08:50:55.958Z] "POST /invalid HTTP/1.1" 404 NR - - "-" 0 0 1 - "-" "curl/8.21.0" "55667739-e394-4814-91b2-2cdd90744892" "example.com" "-" - - - 10.4.1.8:59778 - -`,
					MainMessageStructuredFieldKey:                 "【No route found(NR)】404 POST http://example.com/invalid",
					SeverityStructuredFieldKey:                    inspectioncore_contract.SeverityWarning,
					EnvoyAccessLogTimestampFieldKey:               "2026-08-10T08:50:55.958Z",
					EnvoyAccessLogMethodFieldKey:                  "POST",
					EnvoyAccessLogPathFieldKey:                    "/invalid",
					EnvoyAccessLogProtocolFieldKey:                "HTTP/1.1",
					EnvoyAccessLogStatusCodeFieldKey:              "404",
					EnvoyAccessLogResponseFlagsFieldKey:           "NR",
					EnvoyAccessLogAuthorityFieldKey:               "example.com",
					EnvoyAccessLogDurationFieldKey:                "1",
					EnvoyAccessLogUserAgentFieldKey:               "curl/8.21.0",
					EnvoyAccessLogRequestIDFieldKey:               "55667739-e394-4814-91b2-2cdd90744892",
					EnvoyAccessLogDownstreamRemoteAddressFieldKey: "10.4.1.8:59778",
				},
			},
		},
		{
			name:  "istio-proxy access log with failure details from container stdout",
			input: `[2026-08-17T04:59:08.906Z] "GET / HTTP/1.1" 503 UF upstream_reset_before_response_started{remote_connection_failure,delayed_connect_error:_111} - "delayed_connect_error:_111" 0 152 0 - "-" "GoogleHC/1.0" "b8b3448c-05f3-4f65-aed2-9e63c89cc662" "10.4.0.5" "10.4.0.5:8080" inbound|8080|| - 10.4.0.5:8080 35.191.227.195:48158 - default`,
			want: &ParseStructuredLogResult{
				Fields: map[string]any{
					OriginalMessageFieldKey:                       `[2026-08-17T04:59:08.906Z] "GET / HTTP/1.1" 503 UF upstream_reset_before_response_started{remote_connection_failure,delayed_connect_error:_111} - "delayed_connect_error:_111" 0 152 0 - "-" "GoogleHC/1.0" "b8b3448c-05f3-4f65-aed2-9e63c89cc662" "10.4.0.5" "10.4.0.5:8080" inbound|8080|| - 10.4.0.5:8080 35.191.227.195:48158 - default`,
					MainMessageStructuredFieldKey:                 "【Upstream connection failure(UF)】503 GET http://10.4.0.5/",
					SeverityStructuredFieldKey:                    inspectioncore_contract.SeverityError,
					EnvoyAccessLogTimestampFieldKey:               "2026-08-17T04:59:08.906Z",
					EnvoyAccessLogMethodFieldKey:                  "GET",
					EnvoyAccessLogPathFieldKey:                    "/",
					EnvoyAccessLogProtocolFieldKey:                "HTTP/1.1",
					EnvoyAccessLogStatusCodeFieldKey:              "503",
					EnvoyAccessLogResponseFlagsFieldKey:           "UF",
					EnvoyAccessLogAuthorityFieldKey:               "10.4.0.5",
					EnvoyAccessLogUpstreamHostFieldKey:            "10.4.0.5:8080",
					EnvoyAccessLogUpstreamClusterFieldKey:         "inbound|8080||",
					EnvoyAccessLogDurationFieldKey:                "0",
					EnvoyAccessLogUserAgentFieldKey:               "GoogleHC/1.0",
					EnvoyAccessLogRequestIDFieldKey:               "b8b3448c-05f3-4f65-aed2-9e63c89cc662",
					EnvoyAccessLogDownstreamLocalAddressFieldKey:  "10.4.0.5:8080",
					EnvoyAccessLogDownstreamRemoteAddressFieldKey: "35.191.227.195:48158",
					EnvoyAccessLogRouteNameFieldKey:               "default",
				},
			},
		},
		{
			name:  "multiple response flags (UH,URX) with status 503",
			input: `[2026-08-10T08:50:55.958Z] "GET /productpage HTTP/1.1" 503 UH,URX - - "-" 0 0 10 - "-" "curl/8.21.0" "55667739-e394-4814-91b2-2cdd90744892" "example.com" "10.4.0.5:80" outbound|80||foo.default.svc.cluster.local - - 10.4.1.8:59778 - default`,
			want: &ParseStructuredLogResult{
				Fields: map[string]any{
					OriginalMessageFieldKey:                       `[2026-08-10T08:50:55.958Z] "GET /productpage HTTP/1.1" 503 UH,URX - - "-" 0 0 10 - "-" "curl/8.21.0" "55667739-e394-4814-91b2-2cdd90744892" "example.com" "10.4.0.5:80" outbound|80||foo.default.svc.cluster.local - - 10.4.1.8:59778 - default`,
					MainMessageStructuredFieldKey:                 "【No healthy upstream, Upstream retry limit exceeded(UH,URX)】503 GET http://example.com/productpage",
					SeverityStructuredFieldKey:                    inspectioncore_contract.SeverityError,
					EnvoyAccessLogTimestampFieldKey:               "2026-08-10T08:50:55.958Z",
					EnvoyAccessLogMethodFieldKey:                  "GET",
					EnvoyAccessLogPathFieldKey:                    "/productpage",
					EnvoyAccessLogProtocolFieldKey:                "HTTP/1.1",
					EnvoyAccessLogStatusCodeFieldKey:              "503",
					EnvoyAccessLogResponseFlagsFieldKey:           "UH,URX",
					EnvoyAccessLogAuthorityFieldKey:               "example.com",
					EnvoyAccessLogUpstreamHostFieldKey:            "10.4.0.5:80",
					EnvoyAccessLogUpstreamClusterFieldKey:         "outbound|80||foo.default.svc.cluster.local",
					EnvoyAccessLogDurationFieldKey:                "10",
					EnvoyAccessLogUserAgentFieldKey:               "curl/8.21.0",
					EnvoyAccessLogRequestIDFieldKey:               "55667739-e394-4814-91b2-2cdd90744892",
					EnvoyAccessLogDownstreamRemoteAddressFieldKey: "10.4.1.8:59778",
					EnvoyAccessLogRouteNameFieldKey:               "default",
				},
			},
		},
		{
			name:  "plain text log (rejected)",
			input: "2026-08-10T08:50:55.958Z info starting server on port 8080",
			want:  nil,
		},
	}

	parser := NewEnvoyAccessLogTextParser()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := parser.TryParse(tc.input)
			if diff := cmp.Diff(tc.want, got, protocmp.Transform()); diff != "" {
				t.Errorf("TryParse() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
