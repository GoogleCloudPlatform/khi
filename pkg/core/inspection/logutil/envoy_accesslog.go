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
	"regexp"
	"strconv"
	"strings"

	pb "github.com/GoogleCloudPlatform/khi/pkg/generated/khifile/v6"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"
)

// EnvoyAccessLogTimestampFieldKey is the key stored in Fields for the timestamp in an Envoy access log.
const EnvoyAccessLogTimestampFieldKey = "@timestamp"

// EnvoyAccessLogMethodFieldKey is the key stored in Fields for the HTTP method.
const EnvoyAccessLogMethodFieldKey = "method"

// EnvoyAccessLogPathFieldKey is the key stored in Fields for the request path.
const EnvoyAccessLogPathFieldKey = "path"

// EnvoyAccessLogProtocolFieldKey is the key stored in Fields for the protocol.
const EnvoyAccessLogProtocolFieldKey = "protocol"

// EnvoyAccessLogStatusCodeFieldKey is the key stored in Fields for the response status code.
const EnvoyAccessLogStatusCodeFieldKey = "status_code"

// EnvoyAccessLogResponseFlagsFieldKey is the key stored in Fields for the response flags.
const EnvoyAccessLogResponseFlagsFieldKey = "response_flags"

// EnvoyAccessLogAuthorityFieldKey is the key stored in Fields for the request authority/host.
const EnvoyAccessLogAuthorityFieldKey = "authority"

// EnvoyAccessLogUpstreamHostFieldKey is the key stored in Fields for the upstream host.
const EnvoyAccessLogUpstreamHostFieldKey = "upstream_host"

// EnvoyAccessLogUpstreamClusterFieldKey is the key stored in Fields for the upstream cluster.
const EnvoyAccessLogUpstreamClusterFieldKey = "upstream_cluster"

// EnvoyAccessLogDurationFieldKey is the key stored in Fields for the total duration.
const EnvoyAccessLogDurationFieldKey = "duration"

// EnvoyAccessLogUserAgentFieldKey is the key stored in Fields for the user agent.
const EnvoyAccessLogUserAgentFieldKey = "user_agent"

// EnvoyAccessLogRequestIDFieldKey is the key stored in Fields for the request ID.
const EnvoyAccessLogRequestIDFieldKey = "request_id"

// EnvoyAccessLogUpstreamLocalAddressFieldKey is the key stored in Fields for the upstream local address.
const EnvoyAccessLogUpstreamLocalAddressFieldKey = "upstream_local_address"

// EnvoyAccessLogDownstreamLocalAddressFieldKey is the key stored in Fields for the downstream local address.
const EnvoyAccessLogDownstreamLocalAddressFieldKey = "downstream_local_address"

// EnvoyAccessLogDownstreamRemoteAddressFieldKey is the key stored in Fields for the downstream remote address.
const EnvoyAccessLogDownstreamRemoteAddressFieldKey = "downstream_remote_address"

// EnvoyAccessLogRequestedServerNameFieldKey is the key stored in Fields for the requested server name (SNI).
const EnvoyAccessLogRequestedServerNameFieldKey = "requested_server_name"

// EnvoyAccessLogRouteNameFieldKey is the key stored in Fields for the route name.
const EnvoyAccessLogRouteNameFieldKey = "route_name"

// envoyAccessLogRegex matches default Istio access log text format:
// [%START_TIME%] "%REQ(:METHOD)% %REQ(X-ENVOY-ORIGINAL-PATH?:PATH)% %PROTOCOL%" %RESPONSE_CODE% %RESPONSE_FLAGS% %RESPONSE_CODE_DETAILS% %CONNECTION_TERMINATION_DETAILS%
// "%UPSTREAM_TRANSPORT_FAILURE_REASON%" %BYTES_RECEIVED% %BYTES_SENT% %DURATION% %RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)% "%REQ(X-FORWARDED-FOR)%" "%REQ(USER-AGENT)%" "%REQ(X-REQUEST-ID)%"
// "%REQ(:AUTHORITY)%" "%UPSTREAM_HOST%" %UPSTREAM_CLUSTER_RAW% %UPSTREAM_LOCAL_ADDRESS% %DOWNSTREAM_LOCAL_ADDRESS% %DOWNSTREAM_REMOTE_ADDRESS% %REQUESTED_SERVER_NAME% %ROUTE_NAME%
//
// See: https://istio.io/latest/docs/tasks/observability/logs/access-log/#default-access-log-format
var envoyAccessLogRegex = regexp.MustCompile(
	`^\[(?P<startTime>[^\]]+)\]\s+"(?P<method>[A-Z]+)\s+(?P<path>\S+)\s+(?P<protocol>[^"]+)"\s+(?P<statusCode>\d{3})\s+(?P<responseFlags>\S+)\s+(?P<responseCodeDetails>\S+)\s+(?P<connTerminationDetails>\S+)\s+"(?P<upstreamFailureReason>[^"]*)"\s+(?P<bytesReceived>\S+)\s+(?P<bytesSent>\S+)\s+(?P<duration>\S+)\s+(?P<upstreamServiceTime>\S+)\s+"(?P<xForwardedFor>[^"]*)"\s+"(?P<userAgent>[^"]*)"\s+"(?P<requestId>[^"]*)"\s+"(?P<authority>[^"]*)"\s+"(?P<upstreamHost>[^"]*)"\s+(?P<upstreamCluster>\S+)\s+(?P<upstreamLocalAddress>\S+)\s+(?P<downstreamLocalAddress>\S+)\s+(?P<downstreamRemoteAddress>\S+)\s+(?P<requestedServerName>\S+)\s+(?P<routeName>\S+)`,
)

// EnvoyAccessLogTextParser parses access log lines formatted in the default Istio text format.
type EnvoyAccessLogTextParser struct{}

// NewEnvoyAccessLogTextParser creates a new EnvoyAccessLogTextParser instance.
func NewEnvoyAccessLogTextParser() *EnvoyAccessLogTextParser {
	return &EnvoyAccessLogTextParser{}
}

// TryParse attempts to parse the given message as an Istio access log.
func (p *EnvoyAccessLogTextParser) TryParse(message string) *ParseStructuredLogResult {
	match := envoyAccessLogRegex.FindStringSubmatch(message)
	if match == nil {
		return nil
	}

	subexpNames := envoyAccessLogRegex.SubexpNames()
	fields := make(map[string]string)
	for i, name := range subexpNames {
		if i != 0 && name != "" {
			fields[name] = match[i]
		}
	}

	statusCodeStr := fields["statusCode"]
	statusCode, err := strconv.Atoi(statusCodeStr)
	if err != nil {
		return nil
	}

	method := fields["method"]
	path := fields["path"]
	protocol := fields["protocol"]
	responseFlagsStr := fields["responseFlags"]
	responseFlags := ParseEnvoyResponseFlags(responseFlagsStr)
	authority := fields["authority"]
	startTime := fields["startTime"]

	requestURL := buildEnvoyRequestURL(authority, path)
	summary := FormatEnvoySummary(statusCode, method, requestURL, responseFlags)
	severity := parseEnvoySeverity(statusCode, responseFlags)

	parsedFields := map[string]any{
		OriginalMessageFieldKey:             message,
		MainMessageStructuredFieldKey:       summary,
		SeverityStructuredFieldKey:          severity,
		EnvoyAccessLogTimestampFieldKey:     startTime,
		EnvoyAccessLogMethodFieldKey:        method,
		EnvoyAccessLogPathFieldKey:          path,
		EnvoyAccessLogProtocolFieldKey:      protocol,
		EnvoyAccessLogStatusCodeFieldKey:    statusCodeStr,
		EnvoyAccessLogResponseFlagsFieldKey: responseFlagsStr,
		EnvoyAccessLogAuthorityFieldKey:     authority,
		EnvoyAccessLogDurationFieldKey:      fields["duration"],
		EnvoyAccessLogUserAgentFieldKey:     fields["userAgent"],
		EnvoyAccessLogRequestIDFieldKey:     fields["requestId"],
	}

	if upstreamHost, ok := fields["upstreamHost"]; ok && upstreamHost != "" && upstreamHost != "-" {
		parsedFields[EnvoyAccessLogUpstreamHostFieldKey] = upstreamHost
	}
	if upstreamCluster, ok := fields["upstreamCluster"]; ok && upstreamCluster != "" && upstreamCluster != "-" {
		parsedFields[EnvoyAccessLogUpstreamClusterFieldKey] = upstreamCluster
	}
	if upstreamLocalAddress, ok := fields["upstreamLocalAddress"]; ok && upstreamLocalAddress != "" && upstreamLocalAddress != "-" {
		parsedFields[EnvoyAccessLogUpstreamLocalAddressFieldKey] = upstreamLocalAddress
	}
	if downstreamLocalAddress, ok := fields["downstreamLocalAddress"]; ok && downstreamLocalAddress != "" && downstreamLocalAddress != "-" {
		parsedFields[EnvoyAccessLogDownstreamLocalAddressFieldKey] = downstreamLocalAddress
	}
	if downstreamRemoteAddress, ok := fields["downstreamRemoteAddress"]; ok && downstreamRemoteAddress != "" && downstreamRemoteAddress != "-" {
		parsedFields[EnvoyAccessLogDownstreamRemoteAddressFieldKey] = downstreamRemoteAddress
	}
	if requestedServerName, ok := fields["requestedServerName"]; ok && requestedServerName != "" && requestedServerName != "-" {
		parsedFields[EnvoyAccessLogRequestedServerNameFieldKey] = requestedServerName
	}
	if routeName, ok := fields["routeName"]; ok && routeName != "" && routeName != "-" {
		parsedFields[EnvoyAccessLogRouteNameFieldKey] = routeName
	}

	return &ParseStructuredLogResult{
		Fields: parsedFields,
	}
}

var _ StructuredLogParser = (*EnvoyAccessLogTextParser)(nil)

func buildEnvoyRequestURL(authority, path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if authority != "" && authority != "-" {
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		return "http://" + authority + path
	}
	return path
}

func parseEnvoySeverity(statusCode int, responseFlags EnvoyResponseFlags) *pb.Severity {
	if statusCode >= 500 {
		return inspectioncore_contract.SeverityError
	}
	if statusCode >= 400 {
		return inspectioncore_contract.SeverityWarning
	}
	if responseFlags.HasError() {
		return inspectioncore_contract.SeverityError
	}
	return inspectioncore_contract.SeverityInfo
}
