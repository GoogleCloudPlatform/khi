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
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	forbiddenHostBody        = "Forbidden: host not allowed"
	forbiddenCrossOriginBody = "Forbidden: cross-origin request rejected"
)

// HostValidationMiddleware validates Request.Host against allowlist.
func HostValidationMiddleware(allowlist *HostAllowlist) gin.HandlerFunc {
	return func(c *gin.Context) {
		if allowlist != nil && allowlist.Allows(c.Request.Host) {
			c.Next()
			return
		}

		slog.WarnContext(
			c.Request.Context(),
			"Rejected request with disallowed Host header; configure --allowed-hosts or KHI_ALLOWED_HOSTS if this host is expected",
			"host", c.Request.Host,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
		)
		c.String(http.StatusForbidden, forbiddenHostBody)
		c.Abort()
	}
}

// CrossOriginProtectionMiddleware rejects unsafe cross-origin browser requests using stdlib CrossOriginProtection.
func CrossOriginProtectionMiddleware() gin.HandlerFunc {
	protection := http.NewCrossOriginProtection()
	return func(c *gin.Context) {
		if err := protection.Check(c.Request); err != nil {
			slog.WarnContext(
				c.Request.Context(),
				"Rejected cross-origin request",
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
				"sec_fetch_site", c.Request.Header.Get("Sec-Fetch-Site"),
				"origin", c.Request.Header.Get("Origin"),
				"error", err,
			)
			c.String(http.StatusForbidden, forbiddenCrossOriginBody)
			c.Abort()
			return
		}
		c.Next()
	}
}

// MCPOriginProtectionMiddleware rejects cross-origin browser requests across all HTTP methods on MCP endpoints.
func MCPOriginProtectionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isAllowedMCPOrigin(c.Request) {
			c.Next()
			return
		}

		slog.WarnContext(
			c.Request.Context(),
			"Rejected cross-origin MCP request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"sec_fetch_site", c.Request.Header.Get("Sec-Fetch-Site"),
			"origin", c.Request.Header.Get("Origin"),
		)
		c.String(http.StatusForbidden, forbiddenCrossOriginBody)
		c.Abort()
	}
}

func isAllowedMCPOrigin(req *http.Request) bool {
	secFetchSite := strings.ToLower(strings.TrimSpace(req.Header.Get("Sec-Fetch-Site")))
	if secFetchSite != "" {
		return secFetchSite == "same-origin" || secFetchSite == "none"
	}

	rawOrigin := strings.TrimSpace(req.Header.Get("Origin"))
	if rawOrigin == "" {
		return true
	}

	parsedOrigin, err := url.Parse(rawOrigin)
	if err != nil {
		return false
	}

	scheme := strings.ToLower(parsedOrigin.Scheme)
	if scheme != "http" && scheme != "https" {
		return false
	}

	if parsedOrigin.Host == "" || parsedOrigin.User != nil || parsedOrigin.Path != "" || parsedOrigin.RawQuery != "" || parsedOrigin.Fragment != "" {
		return false
	}

	return strings.EqualFold(parsedOrigin.Host, req.Host)
}
