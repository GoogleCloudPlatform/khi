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
	"fmt"
	"strings"
)

const (
	// EnvoyResponseFlagNoError indicates no response flag was set by Envoy.
	EnvoyResponseFlagNoError EnvoyResponseFlag = "-"
	// EnvoyResponseFlagNoHealthyUpstream indicates no healthy upstream host was available.
	EnvoyResponseFlagNoHealthyUpstream EnvoyResponseFlag = "UH"
	// EnvoyResponseFlagUpstreamConnectionFailure indicates a connection failure to the upstream host.
	EnvoyResponseFlagUpstreamConnectionFailure EnvoyResponseFlag = "UF"
	// EnvoyResponseFlagUpstreamOverflow indicates upstream circuit breaker overflow.
	EnvoyResponseFlagUpstreamOverflow EnvoyResponseFlag = "UO"
	// EnvoyResponseFlagNoRouteFound indicates no matching route was found for the request.
	EnvoyResponseFlagNoRouteFound EnvoyResponseFlag = "NR"
	// EnvoyResponseFlagUpstreamRetryLimitExceeded indicates the upstream retry limit was exceeded.
	EnvoyResponseFlagUpstreamRetryLimitExceeded EnvoyResponseFlag = "URX"
	// EnvoyResponseFlagNoClusterFound indicates no matching upstream cluster was found.
	EnvoyResponseFlagNoClusterFound EnvoyResponseFlag = "NC"
	// EnvoyResponseFlagDurationTimeout indicates the request exceeded the configured max duration timeout.
	EnvoyResponseFlagDurationTimeout EnvoyResponseFlag = "DT"

	// EnvoyResponseFlagDownstreamConnectionTermination indicates the downstream connection was terminated (HTTP only).
	EnvoyResponseFlagDownstreamConnectionTermination EnvoyResponseFlag = "DC"
	// EnvoyResponseFlagFailedLocalHealthCheck indicates the request failed a local health check.
	EnvoyResponseFlagFailedLocalHealthCheck EnvoyResponseFlag = "LH"
	// EnvoyResponseFlagUpstreamRequestTimeout indicates the upstream request timed out.
	EnvoyResponseFlagUpstreamRequestTimeout EnvoyResponseFlag = "UT"
	// EnvoyResponseFlagLocalReset indicates the stream was reset locally by Envoy.
	EnvoyResponseFlagLocalReset EnvoyResponseFlag = "LR"
	// EnvoyResponseFlagUpstreamRemoteReset indicates the stream was reset remotely by the upstream.
	EnvoyResponseFlagUpstreamRemoteReset EnvoyResponseFlag = "UR"
	// EnvoyResponseFlagUpstreamConnectionTermination indicates the upstream connection was terminated.
	EnvoyResponseFlagUpstreamConnectionTermination EnvoyResponseFlag = "UC"
	// EnvoyResponseFlagDelayInjected indicates fault injection delay was applied.
	EnvoyResponseFlagDelayInjected EnvoyResponseFlag = "DI"
	// EnvoyResponseFlagFaultInjected indicates a fault injection abort was applied.
	EnvoyResponseFlagFaultInjected EnvoyResponseFlag = "FI"
	// EnvoyResponseFlagRateLimited indicates the request was rate limited.
	EnvoyResponseFlagRateLimited EnvoyResponseFlag = "RL"
	// EnvoyResponseFlagUnauthorizedExternalService indicates authorization failed for an external service.
	EnvoyResponseFlagUnauthorizedExternalService EnvoyResponseFlag = "UAEX"
	// EnvoyResponseFlagRateLimitServiceError indicates an error calling the rate limit service.
	EnvoyResponseFlagRateLimitServiceError EnvoyResponseFlag = "RLSE"
	// EnvoyResponseFlagInvalidEnvoyRequestHeaders indicates invalid Envoy request headers were received.
	EnvoyResponseFlagInvalidEnvoyRequestHeaders EnvoyResponseFlag = "IH"
	// EnvoyResponseFlagStreamIdleTimeout indicates the stream exceeded its idle timeout.
	EnvoyResponseFlagStreamIdleTimeout EnvoyResponseFlag = "SI"
	// EnvoyResponseFlagDownstreamProtocolError indicates a downstream protocol error occurred.
	EnvoyResponseFlagDownstreamProtocolError EnvoyResponseFlag = "DPE"
	// EnvoyResponseFlagUpstreamProtocolError indicates an upstream protocol error occurred.
	EnvoyResponseFlagUpstreamProtocolError EnvoyResponseFlag = "UPE"
	// EnvoyResponseFlagUpstreamMaxStreamDurationReached indicates the upstream reached max stream duration.
	EnvoyResponseFlagUpstreamMaxStreamDurationReached EnvoyResponseFlag = "UMSDR"
	// EnvoyResponseFlagResponseFromCacheFilter indicates the response was served from the cache filter.
	EnvoyResponseFlagResponseFromCacheFilter EnvoyResponseFlag = "RFCF"
	// EnvoyResponseFlagNoFilterConfigFound indicates no filter configuration was found.
	EnvoyResponseFlagNoFilterConfigFound EnvoyResponseFlag = "NFCF"
	// EnvoyResponseFlagOverloadManagerTerminated indicates the stream was terminated by the overload manager.
	EnvoyResponseFlagOverloadManagerTerminated EnvoyResponseFlag = "OM"
	// EnvoyResponseFlagDnsResolutionFailed indicates DNS resolution failed for the upstream.
	EnvoyResponseFlagDnsResolutionFailed EnvoyResponseFlag = "DF"
	// EnvoyResponseFlagDropOverload indicates the request was dropped due to overload.
	EnvoyResponseFlagDropOverload EnvoyResponseFlag = "DO"
	// EnvoyResponseFlagDownstreamRemoteReset indicates the stream was reset remotely by the downstream.
	EnvoyResponseFlagDownstreamRemoteReset EnvoyResponseFlag = "DR"
	// EnvoyResponseFlagUnconditionalDropOverload indicates the request was dropped due to unconditional drop overload.
	EnvoyResponseFlagUnconditionalDropOverload EnvoyResponseFlag = "UDO"
)

// EnvoyResponseFlagDescriptions maps Envoy response flags to their human-readable descriptions.
var EnvoyResponseFlagDescriptions = map[EnvoyResponseFlag]string{
	EnvoyResponseFlagNoError:                          "OK",
	EnvoyResponseFlagNoHealthyUpstream:                "No healthy upstream",
	EnvoyResponseFlagUpstreamConnectionFailure:        "Upstream connection failure",
	EnvoyResponseFlagUpstreamOverflow:                 "Upstream overflow",
	EnvoyResponseFlagNoRouteFound:                     "No route found",
	EnvoyResponseFlagUpstreamRetryLimitExceeded:       "Upstream retry limit exceeded",
	EnvoyResponseFlagNoClusterFound:                   "No cluster found",
	EnvoyResponseFlagDurationTimeout:                  "Duration timeout",
	EnvoyResponseFlagDownstreamConnectionTermination:  "Downstream connection termination",
	EnvoyResponseFlagFailedLocalHealthCheck:           "Failed local health check",
	EnvoyResponseFlagUpstreamRequestTimeout:           "Upstream request timeout",
	EnvoyResponseFlagLocalReset:                       "Local reset",
	EnvoyResponseFlagUpstreamRemoteReset:              "Upstream remote reset",
	EnvoyResponseFlagUpstreamConnectionTermination:    "Upstream connection termination",
	EnvoyResponseFlagDelayInjected:                    "Delay injected",
	EnvoyResponseFlagFaultInjected:                    "Fault injected",
	EnvoyResponseFlagRateLimited:                      "Rate limited",
	EnvoyResponseFlagUnauthorizedExternalService:      "Unauthorized external service",
	EnvoyResponseFlagRateLimitServiceError:            "Rate limit service error",
	EnvoyResponseFlagInvalidEnvoyRequestHeaders:       "Invalid Envoy request headers",
	EnvoyResponseFlagStreamIdleTimeout:                "Stream idle timeout",
	EnvoyResponseFlagDownstreamProtocolError:          "Downstream protocol error",
	EnvoyResponseFlagUpstreamProtocolError:            "Upstream protocol error",
	EnvoyResponseFlagUpstreamMaxStreamDurationReached: "Upstream max stream duration reached",
	EnvoyResponseFlagResponseFromCacheFilter:          "Response from cache filter",
	EnvoyResponseFlagNoFilterConfigFound:              "No filter config found",
	EnvoyResponseFlagOverloadManagerTerminated:        "Overload manager terminated",
	EnvoyResponseFlagDnsResolutionFailed:              "DNS resolution failed",
	EnvoyResponseFlagDropOverload:                     "Drop overload",
	EnvoyResponseFlagDownstreamRemoteReset:            "Downstream remote reset",
	EnvoyResponseFlagUnconditionalDropOverload:        "Unconditional drop overload",
}

// EnvoyResponseFlag represents standard Envoy response flags.
// See: https://github.com/envoyproxy/envoy/blob/main/docs/root/configuration/advanced/substitution_formatter.rst
type EnvoyResponseFlag string

// EnvoyResponseFlags represents a collection of Envoy response flags.
type EnvoyResponseFlags []EnvoyResponseFlag

// ParseEnvoyResponseFlags parses single or comma-separated Envoy response flag strings.
func ParseEnvoyResponseFlags(raw string) EnvoyResponseFlags {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return EnvoyResponseFlags{}
	}
	parts := strings.Split(raw, ",")
	flags := make(EnvoyResponseFlags, len(parts))
	for i, p := range parts {
		flags[i] = EnvoyResponseFlag(strings.TrimSpace(p))
	}
	return flags
}

// String returns the comma-separated flag representation (e.g. "UH,URX").
func (f EnvoyResponseFlags) String() string {
	strs := make([]string, len(f))
	for i, flag := range f {
		strs[i] = string(flag)
	}
	return strings.Join(strs, ",")
}

// Summary returns human-readable descriptions of the flags joined by comma.
func (f EnvoyResponseFlags) Summary() string {
	msgs := make([]string, len(f))
	for i, flag := range f {
		if desc, ok := EnvoyResponseFlagDescriptions[flag]; ok {
			msgs[i] = desc
		} else {
			msgs[i] = string(flag)
		}
	}
	return strings.Join(msgs, ", ")
}

// HasError returns true if there is any error flag present, excluding non-error flags such as NoError, RFCF, and DI.
func (f EnvoyResponseFlags) HasError() bool {
	for _, flag := range f {
		if flag != EnvoyResponseFlagNoError &&
			flag != EnvoyResponseFlagResponseFromCacheFilter &&
			flag != EnvoyResponseFlagDelayInjected {
			return true
		}
	}
	return false
}

// Contains returns true if the specified flag is present in the response flags.
func (f EnvoyResponseFlags) Contains(flag EnvoyResponseFlag) bool {
	for _, item := range f {
		if item == flag {
			return true
		}
	}
	return false
}

// FormatEnvoySummary formats an Envoy access log summary line with status code, method, request URL, and optional response flags.
func FormatEnvoySummary(statusCode int, method, requestURL string, responseFlags EnvoyResponseFlags) string {
	summary := fmt.Sprintf("%d %s %s", statusCode, method, requestURL)
	if len(responseFlags) == 0 || (len(responseFlags) == 1 && responseFlags[0] == EnvoyResponseFlagNoError) {
		return summary
	}
	return fmt.Sprintf("【%s(%s)】%s", responseFlags.Summary(), responseFlags.String(), summary)
}
