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
	"context"
	"embed"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/filter"
	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coreinspection "github.com/GoogleCloudPlatform/khi/pkg/core/inspection"
	inspectionmetadata "github.com/GoogleCloudPlatform/khi/pkg/core/inspection/metadata"
	inspectioncore_contract "github.com/GoogleCloudPlatform/khi/pkg/task/inspection/inspectioncore/contract"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

const embeddedStaticFolderPath = "dist/browser"

//go:embed dist/browser
var embeddedStaticFolder embed.FS

// InspectionIndexer defines the contract for asynchronously indexing an inspection and invalidating stale indices.
type InspectionIndexer interface {
	StartAsyncIndexing(ctx context.Context, inspectionID string)
	InvalidateInspectionIndex(inspectionID string)
}

type ServerConfig struct {
	StaticFolderPath string
	ResourceMonitor  ResourceMonitor
	ServerBasePath   string
	IndexManager     InspectionIndexer
}

func redirectMiddleware(exactPath string, redirectTo string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if ctx.Request.URL.Path == exactPath {
			ctx.Redirect(302, redirectTo)
			return
		}
		ctx.Next()
	}
}

func coepCoopMiddleware() gin.HandlerFunc {
	// KHI uses SharedArrayBuffer to share memory between the main thread and worker threads.
	// COOP and COEP headers are required for SharedArrayBuffer to work.
	// See https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/SharedArrayBuffer#api_availability
	return func(ctx *gin.Context) {
		ctx.Header("Cross-Origin-Opener-Policy", "same-origin")
		ctx.Header("Cross-Origin-Embedder-Policy", "require-corp")
		ctx.Next()
	}
}

// SetupKHIServerRoutes mounts base middlewares, static files, and standard REST routes onto engine.
// It returns the normalized base path without trailing slash and the created gin.IRouter.
func SetupKHIServerRoutes(engine *gin.Engine, inspectionServer *coreinspection.InspectionTaskServer, serverConfig *ServerConfig) (string, gin.IRouter) {
	if serverConfig.IndexManager == nil {
		panic("serverConfig.IndexManager is required")
	}
	engine.Use(coepCoopMiddleware())
	basePathWithoutTrailingSlash := strings.TrimSuffix(serverConfig.ServerBasePath, "/")
	engine.Use(redirectMiddleware(basePathWithoutTrailingSlash+"/", basePathWithoutTrailingSlash+"/session/0")) // Request for `/` shouldn't be handled by `static.Serve`, redirect `/session/0` to be handled by patternToString

	// By default, use the embedded web files. If the static folder path is set, use the local file system.
	appHtmlPath := path.Join(embeddedStaticFolderPath, "/index.html")
	webFS := embedFolder(embeddedStaticFolder, embeddedStaticFolderPath)
	webFSDebugMessage := "Using embedded static web files."
	fileReaderFunc := embeddedStaticFolder.ReadFile
	if serverConfig.StaticFolderPath != "" {
		appHtmlPath = path.Join(serverConfig.StaticFolderPath, "/index.html")
		webFS = static.LocalFile(serverConfig.StaticFolderPath, false)
		webFSDebugMessage = fmt.Sprintf("Using local file system for static web files from %s", serverConfig.StaticFolderPath)
		fileReaderFunc = os.ReadFile
	}
	slog.Debug(webFSDebugMessage)
	engine.Use(static.Serve(basePathWithoutTrailingSlash+"/", webFS))

	router := engine.Group(basePathWithoutTrailingSlash)

	// frontend uses Angular router. All frontend routing path should return the app html
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

	// GET /api/v3/inspection/types
	// Returns the list of inspection types available on the inspection server.
	router.GET("/api/v3/inspection/types", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, &GetInspectionTypesResponse{
			Types: inspectionServer.GetAllInspectionTypes(),
		})
	})

	// GET /api/v3/inspection
	// Returns the all started inspections on the inspection server.
	router.GET("/api/v3/inspection", func(ctx *gin.Context) {
		inspections := inspectionServer.GetAllRunners()
		responseInspections := map[string]SerializedMetadata{}
		for _, inspection := range inspections {
			if inspection.Started() {
				md, err := inspection.GetCurrentMetadata()
				if err != nil {
					ctx.String(http.StatusInternalServerError, err.Error())
					return
				}

				m, err := inspectionmetadata.GetSerializableSubsetMapFromMetadataSet(md, filter.NewEnabledFilter(inspectionmetadata.LabelKeyIncludedInTaskListFlag, false))
				if err != nil {
					ctx.String(http.StatusInternalServerError, err.Error())
					return
				}
				responseInspections[inspection.ID] = m
			}
		}

		ctx.JSON(http.StatusOK, &GetInspectionsResponse{
			Inspections: responseInspections,
			ServerStat: &ServerStat{
				CurrentMemoryUsage: serverConfig.ResourceMonitor.GetUsedMemory(),
				TotalMemory:        serverConfig.ResourceMonitor.GetTotalMemory(),
			},
		})
	})

	// POST /api/v3/inspection/types/:typeID
	router.POST("/api/v3/inspection/types/:typeID", func(ctx *gin.Context) {
		typeID := ctx.Param("typeID")
		inspectionId, err := inspectionServer.CreateInspection(typeID)
		if err != nil {
			// only the not found error is expected here
			ctx.String(http.StatusNotFound, err.Error())
			return
		}
		ctx.JSON(http.StatusAccepted, &PostInspectionResponse{InspectionID: inspectionId})
	})
	// PATCH /api/v3/inspection/<inspection-id>
	router.PATCH("/api/v3/inspection/:inspectionID", func(ctx *gin.Context) {
		inspectionID := ctx.Param("inspectionID")
		task := inspectionServer.GetInspection(inspectionID)
		if task == nil {
			ctx.String(http.StatusNotFound, fmt.Sprintf("inspection %s was not found", inspectionID))
			return
		}
		var reqBody PatchInspectionRequest
		if err := ctx.ShouldBindJSON(&reqBody); err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}
		md, err := task.GetCurrentMetadata()
		if err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}
		var header *inspectionmetadata.HeaderMetadata
		header, found := typedmap.Get(md, inspectionmetadata.HeaderMetadataKey)
		if !found {
			ctx.String(http.StatusBadRequest, "header not found")
			return
		}
		header.InspectionName = reqBody.Name
		header.SuggestedFileName = fmt.Sprintf("%s.khi", reqBody.Name)
		ctx.String(http.StatusAccepted, "ok")
	})
	// PUT /api/v3/inspection/<inspection-id>/features
	router.PUT("/api/v3/inspection/:inspectionID/features", func(ctx *gin.Context) {
		inspectionID := ctx.Param("inspectionID")
		task := inspectionServer.GetInspection(inspectionID)
		if task == nil {
			ctx.String(http.StatusNotFound, fmt.Sprintf("inspecton %s was not found", inspectionID))
			return
		}
		var reqBody PutInspectionFeatureRequest
		if err := ctx.ShouldBindJSON(&reqBody); err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}
		err := task.SetFeatureList(reqBody.Features)
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}
		ctx.String(http.StatusAccepted, "ok")
	})
	// PATCH /api/v3/inspection/<inspection-id>/features
	router.PATCH("/api/v3/inspection/:inspectionID/features", func(ctx *gin.Context) {
		inspectionID := ctx.Param("inspectionID")
		task := inspectionServer.GetInspection(inspectionID)
		if task == nil {
			ctx.String(http.StatusNotFound, fmt.Sprintf("inspecton %s was not found", inspectionID))
			return
		}
		var reqBody PatchInspectionFeatureRequest
		if err := ctx.ShouldBindJSON(&reqBody); err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}
		err := task.UpdateFeatureMap(reqBody.Features)
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}
		ctx.String(http.StatusAccepted, "ok")
	})
	// GET /api/v3/inspection/<inspection-id>/features
	router.GET("/api/v3/inspection/:inspectionID/features", func(ctx *gin.Context) {
		inspectionID := ctx.Param("inspectionID")
		task := inspectionServer.GetInspection(inspectionID)
		if task == nil {
			ctx.String(http.StatusNotFound, fmt.Sprintf("inspecton %s was not found", inspectionID))
			return
		}
		features, err := task.FeatureList()
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}
		ctx.JSON(http.StatusOK, GetInspectionFeatureResponse{
			Features: features,
		})
	})

	router.POST("/api/v3/inspection/:inspectionID/dryrun", func(ctx *gin.Context) {
		inspectionID := ctx.Param("inspectionID")
		currentTask := inspectionServer.GetInspection(inspectionID)
		if currentTask == nil {
			ctx.String(http.StatusNotFound, fmt.Sprintf("inspecton %s was not found", inspectionID))
			return
		}
		var reqBody PostInspectionDryRunRequest
		if err := ctx.ShouldBindJSON(&reqBody); err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}
		result, err := currentTask.DryRun(ctx, &inspectioncore_contract.InspectionRequest{
			Values: reqBody,
		})
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}
		ctx.JSON(http.StatusOK, result)
	})

	router.POST("/api/v3/inspection/:inspectionID/run", func(ctx *gin.Context) {
		inspectionID := ctx.Param("inspectionID")
		currentTask := inspectionServer.GetInspection(inspectionID)
		if currentTask == nil {
			ctx.String(http.StatusNotFound, fmt.Sprintf("inspecton %s was not found", inspectionID))
			return
		}
		var reqBody PostInspectionDryRunRequest
		if err := ctx.ShouldBindJSON(&reqBody); err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}
		err := currentTask.Run(ctx, &inspectioncore_contract.InspectionRequest{
			Values: reqBody,
		})
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}
		go func() {
			<-currentTask.Wait()
			if res, err := currentTask.Result(); err == nil && res != nil {
				serverConfig.IndexManager.InvalidateInspectionIndex(inspectionID)
				serverConfig.IndexManager.StartAsyncIndexing(context.Background(), inspectionID)
			}
		}()
		ctx.String(http.StatusAccepted, "ok")
	})

	router.POST("/api/v3/inspection/:inspectionID/cancel", func(ctx *gin.Context) {
		inspectionID := ctx.Param("inspectionID")
		currentTask := inspectionServer.GetInspection(inspectionID)
		if currentTask == nil {
			ctx.String(http.StatusNotFound, fmt.Sprintf("inspecton %s was not found", inspectionID))
			return
		}
		err := currentTask.Cancel()
		if err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}
		ctx.String(http.StatusOK, "ok")
	})

	router.GET("/api/v3/inspection/:inspectionID/metadata", func(ctx *gin.Context) {
		inspectionID := ctx.Param("inspectionID")
		currentTask := inspectionServer.GetInspection(inspectionID)
		if currentTask == nil {
			ctx.String(http.StatusNotFound, fmt.Sprintf("inspecton %s was not found", inspectionID))
			return
		}
		result, err := currentTask.Metadata()
		if err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}
		ctx.JSON(http.StatusOK, result)
	})

	router.GET("/api/v3/inspection/:inspectionID/data", func(ctx *gin.Context) {
		inspectionID := ctx.Param("inspectionID")
		currentTask := inspectionServer.GetInspection(inspectionID)
		if currentTask == nil {
			ctx.String(http.StatusNotFound, fmt.Sprintf("inspecton %s was not found", inspectionID))
			return
		}

		// parse range queries
		var rangeStart int64
		var maxSize int64 = math.MaxInt64
		startQueryStr := ctx.Query("start")
		maxSizeQueryStr := ctx.Query("maxSize")
		if startQueryStr != "" {
			var err error
			rangeStart, err = strconv.ParseInt(startQueryStr, 10, 64)
			if err != nil {
				ctx.String(http.StatusBadRequest, err.Error())
				return
			}
		}
		if maxSizeQueryStr != "" {
			var err error
			maxSize, err = strconv.ParseInt(maxSizeQueryStr, 10, 64)
			if err != nil {
				ctx.String(http.StatusBadRequest, err.Error())
				return
			}
		}

		result, err := currentTask.Result()
		if err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}
		inspectionDataReader, err := result.ResultStore.GetRangeReader(rangeStart, maxSize)
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}
		defer inspectionDataReader.Close()
		fileSize, err := result.ResultStore.GetInspectionResultSizeInBytes()
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}
		ctx.DataFromReader(http.StatusOK, min(maxSize, int64(fileSize)-rangeStart), "application/octet-stream", inspectionDataReader, map[string]string{})
	})

	return basePathWithoutTrailingSlash, router
}

// CreateKHIServer creates and sets up standard KHI routes on engine, returning the engine for chaining.
func CreateKHIServer(engine *gin.Engine, inspectionServer *coreinspection.InspectionTaskServer, serverConfig *ServerConfig) *gin.Engine {
	SetupKHIServerRoutes(engine, inspectionServer, serverConfig)
	return engine
}
