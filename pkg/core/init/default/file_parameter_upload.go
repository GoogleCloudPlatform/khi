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
	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	"github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1/apiv1connect"
	serverapiv1 "github.com/GoogleCloudPlatform/khi/pkg/server/api/v1"
	"github.com/GoogleCloudPlatform/khi/pkg/server/chunkedupload"
	"github.com/GoogleCloudPlatform/khi/pkg/server/upload"
)

// InitializerIDFileParameterUpload mounts Connect-RPC FileParameterUploadService onto Gin engine.
const InitializerIDFileParameterUpload coreinit.InitializerID = "khi.default/file-parameter-upload"

// FileParameterUploadInitializer initializes and registers the FileParameterUploadService handlers.
var FileParameterUploadInitializer = &coreinit.Initializer{
	ID: InitializerIDFileParameterUpload,
	Dependencies: []coreinit.InitializerID{
		InitializerIDGinServer,
	},
	Before: []coreinit.InitializerID{
		InitializerIDServerRunner,
	},
	Init: func(ctx *coreinit.InitContext) error {
		jobParams := coreinit.MustGet(ctx, JobParametersKey)
		if *jobParams.JobMode {
			return nil
		}

		uploadStore := coreinit.MustGet(ctx, UploadStoreKey)
		router := coreinit.MustGet(ctx, GinRouterKey)
		basePath := coreinit.MustGet(ctx, BasePathKey)
		commonParams := coreinit.MustGet(ctx, CommonParametersKey)

		uploadFolder := "/tmp"
		if commonParams.UploadFileStoreFolder != nil {
			uploadFolder = *commonParams.UploadFileStoreFolder
		}

		chunkManager := chunkedupload.NewChunkSessionManager(uploadFolder)
		manager := upload.NewFileParameterUploadManager(uploadStore, chunkManager)
		fileUploadServer := serverapiv1.NewFileParameterUploadServiceServer(manager)
		fileUploadPath, fileUploadHandler := apiv1connect.NewFileParameterUploadServiceHandler(fileUploadServer)
		coreinit.RegisterConnectServiceHandler(router, basePath, fileUploadPath, fileUploadHandler)

		return nil
	},
}

func init() {
	coreinit.RegisterInitializer(FileParameterUploadInitializer)
}
