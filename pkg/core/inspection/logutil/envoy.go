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

// EnvoyResponseFlag represents standard Envoy response flags.
// See: https://github.com/envoyproxy/envoy/blob/main/docs/root/configuration/advanced/substitution_formatter.rst
type EnvoyResponseFlag string

const (
	EnvoyResponseFlagNoError                    EnvoyResponseFlag = "-"
	EnvoyResponseFlagNoHealthyUpstream          EnvoyResponseFlag = "UH"
	EnvoyResponseFlagUpstreamConnectionFailure  EnvoyResponseFlag = "UF"
	EnvoyResponseFlagUpstreamOverflow           EnvoyResponseFlag = "UO"
	EnvoyResponseFlagNoRouteFound               EnvoyResponseFlag = "NR"
	EnvoyResponseFlagUpstreamRetryLimitExceeded EnvoyResponseFlag = "URX"
	EnvoyResponseFlagNoClusterFound             EnvoyResponseFlag = "NC"
	EnvoyResponseFlagDurationTimeout            EnvoyResponseFlag = "DT"

	// HTTP only
	EnvoyResponseFlagDownstreamConnectionTermination  EnvoyResponseFlag = "DC"
	EnvoyResponseFlagFailedLocalHealthCheck           EnvoyResponseFlag = "LH"
	EnvoyResponseFlagUpstreamRequestTimeout           EnvoyResponseFlag = "UT"
	EnvoyResponseFlagLocalReset                       EnvoyResponseFlag = "LR"
	EnvoyResponseFlagUpstreamRemoteReset              EnvoyResponseFlag = "UR"
	EnvoyResponseFlagUpstreamConnectionTermination    EnvoyResponseFlag = "UC"
	EnvoyResponseFlagDelayInjected                    EnvoyResponseFlag = "DI"
	EnvoyResponseFlagFaultInjected                    EnvoyResponseFlag = "FI"
	EnvoyResponseFlagRateLimited                      EnvoyResponseFlag = "RL"
	EnvoyResponseFlagUnauthorizedExternalService      EnvoyResponseFlag = "UAEX"
	EnvoyResponseFlagRateLimitServiceError            EnvoyResponseFlag = "RLSE"
	EnvoyResponseFlagInvalidEnvoyRequestHeaders       EnvoyResponseFlag = "IH"
	EnvoyResponseFlagStreamIdleTimeout                EnvoyResponseFlag = "SI"
	EnvoyResponseFlagDownstreamProtocolError          EnvoyResponseFlag = "DPE"
	EnvoyResponseFlagUpstreamProtocolError            EnvoyResponseFlag = "UPE"
	EnvoyResponseFlagUpstreamMaxStreamDurationReached EnvoyResponseFlag = "UMSDR"
	EnvoyResponseFlagResponseFromCacheFilter          EnvoyResponseFlag = "RFCF"
	EnvoyResponseFlagNoFilterConfigFound              EnvoyResponseFlag = "NFCF"
	EnvoyResponseFlagOverloadManagerTerminated        EnvoyResponseFlag = "OM"
	EnvoyResponseFlagDnsResolutionFailed              EnvoyResponseFlag = "DF"
	EnvoyResponseFlagDropOverload                     EnvoyResponseFlag = "DO"
	EnvoyResponseFlagDownstreamRemoteReset            EnvoyResponseFlag = "DR"
	EnvoyResponseFlagUnconditionalDropOverload        EnvoyResponseFlag = "UDO"
)

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

// FormatEnvoySummary formats an Envoy access log summary line with status code, method, request URL, and optional response flags.
func FormatEnvoySummary(statusCode int, method, requestURL string, responseFlags EnvoyResponseFlags) string {
	summary := fmt.Sprintf("%d %s %s", statusCode, method, requestURL)
	if len(responseFlags) == 0 || (len(responseFlags) == 1 && responseFlags[0] == EnvoyResponseFlagNoError) {
		return summary
	}
	return fmt.Sprintf("【%s(%s)】%s", responseFlags.Summary(), responseFlags.String(), summary)
}
