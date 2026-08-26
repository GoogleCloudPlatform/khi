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
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	"github.com/GoogleCloudPlatform/khi/pkg/server"
	"github.com/GoogleCloudPlatform/khi/pkg/server/upload"
	"github.com/gin-gonic/gin"
)

var (
	// UploadStoreKey stores the UploadFileStore instance.
	UploadStoreKey = typedmap.NewTypedKey[*upload.UploadFileStore]("khi.google.com/init/upload-store")

	// GinRouterKey stores the base gin.IRouter instance.
	GinRouterKey = typedmap.NewTypedKey[gin.IRouter]("khi.google.com/init/gin-router")

	// BasePathKey stores the normalized server base path.
	BasePathKey = typedmap.NewTypedKey[string]("khi.google.com/init/base-path")
)

// InitializerIDGinServer mounts default REST endpoints and static files onto Gin engine.
const InitializerIDGinServer coreinit.InitializerID = "khi.default/gin-server"

// GinServerInitializer mounts standard KHI HTTP routes, frontend assets, and store configs onto gin.Engine.
var GinServerInitializer = &coreinit.Initializer{
	ID: InitializerIDGinServer,
	Dependencies: []coreinit.InitializerID{
		InitializerIDGinEngine,
	},
	Before: []coreinit.InitializerID{
		InitializerIDServerRunner,
	},
	Init: func(ctx *coreinit.InitContext) error {
		jobParams := coreinit.MustGet(ctx, JobParametersKey)
		if *jobParams.JobMode {
			return nil
		}
		commonParams := coreinit.MustGet(ctx, CommonParametersKey)
		serverParams := coreinit.MustGet(ctx, ServerParametersKey)

		uploadFileStoreFolder := "/tmp"
		if commonParams.UploadFileStoreFolder != nil {
			uploadFileStoreFolder = *commonParams.UploadFileStoreFolder
		}
		uploadFileStore := upload.NewUploadFileStore(upload.NewLocalUploadFileStoreProvider(uploadFileStoreFolder))
		upload.DefaultUploadFileStore = uploadFileStore
		coreinit.Set(ctx, UploadStoreKey, uploadFileStore)

		engine := coreinit.MustGet(ctx, GinEngineKey)
		basePath := strings.TrimSuffix(*serverParams.BasePath, "/")
		var router gin.IRouter = engine.Group(basePath)

		coreinit.Set(ctx, GinRouterKey, router)
		coreinit.Set(ctx, BasePathKey, basePath)

		server.SetupFrontendMiddleware(engine, basePath, *serverParams.FrontendAssetFolder)
		server.SetupFrontendRoutes(router, *serverParams.FrontendAssetFolder)
		return nil
	},
}

func init() {
	coreinit.RegisterInitializer(GinServerInitializer)
}
