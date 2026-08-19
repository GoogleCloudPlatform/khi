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
	"strings"

	"github.com/gin-gonic/gin"
)

// RegisterConnectServiceHandler registers a Connect-RPC service handler to the Gin router, wrapping it to strip the basePath if needed.
func RegisterConnectServiceHandler(router gin.IRouter, basePathWithoutTrailingSlash string, servicePath string, handler http.Handler) {
	var wrappedHandler http.Handler = handler
	basePath := strings.TrimSuffix(basePathWithoutTrailingSlash, "/")
	if basePath != "" {
		wrappedHandler = http.StripPrefix(basePath, handler)
	}
	router.Any(servicePath+"*action", gin.WrapH(wrappedHandler))
}
