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
	"embed"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

const embeddedStaticFolderPath = "dist/browser"

//go:embed dist/browser
var embeddedStaticFolder embed.FS

func redirectMiddleware(exactPath string, redirectTo string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if ctx.Request.URL.Path == exactPath {
			ctx.Redirect(302, redirectTo)
			return
		}
		ctx.Next()
	}
}

// SetupFrontendMiddleware mounts base redirect and static file serving middlewares onto engine.
func SetupFrontendMiddleware(engine *gin.Engine, basePath string, staticFolderPath string) {
	basePathWithoutTrailingSlash := strings.TrimSuffix(basePath, "/")
	exactPath := basePathWithoutTrailingSlash + "/"
	redirectTo := basePathWithoutTrailingSlash + "/session/0"
	engine.Use(redirectMiddleware(exactPath, redirectTo))

	webFS := embedFolder(embeddedStaticFolder, embeddedStaticFolderPath)
	webFSDebugMessage := "Using embedded static web files."
	if staticFolderPath != "" {
		webFS = static.LocalFile(staticFolderPath, false)
		webFSDebugMessage = fmt.Sprintf("Using local file system for static web files from %s", staticFolderPath)
	}
	slog.Debug(webFSDebugMessage)
	engine.Use(static.Serve(exactPath, webFS))
}

// SetupFrontendRoutes mounts the SPA fallback session routes onto router.
func SetupFrontendRoutes(router gin.IRouter, staticFolderPath string) {
	appHtmlPath := path.Join(embeddedStaticFolderPath, "/index.html")
	fileReaderFunc := embeddedStaticFolder.ReadFile
	if staticFolderPath != "" {
		appHtmlPath = path.Join(staticFolderPath, "/index.html")
		fileReaderFunc = os.ReadFile
	}

	// frontend uses Angular router. All frontend routing path should return the app html.
	router.GET("/session/*wild", func(ctx *gin.Context) {
		ctx.Header("Content-Type", "text/html")
		file, err := fileReaderFunc(appHtmlPath)
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}
		originalIndexHTML := string(file)
		replacedIndexHtml, err := replaceDynamicPartOfIndex(originalIndexHTML)
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}
		ctx.Writer.Write([]byte(replacedIndexHtml))
	})
}
