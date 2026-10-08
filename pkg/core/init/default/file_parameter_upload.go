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
	"net"
	"strconv"

	"github.com/GoogleCloudPlatform/khi/pkg/common/typedmap"
	coreinit "github.com/GoogleCloudPlatform/khi/pkg/core/init"
	"github.com/GoogleCloudPlatform/khi/pkg/generated/api/v1/apiv1connect"
	serverapiv1 "github.com/GoogleCloudPlatform/khi/pkg/server/api/v1"
	"github.com/GoogleCloudPlatform/khi/pkg/server/chunkedupload"
	"github.com/GoogleCloudPlatform/khi/pkg/server/upload"
	"github.com/GoogleCloudPlatform/khi/pkg/server/uploadurl"
	"github.com/gin-gonic/gin"
)

var (
	// UploadURLIssuerKey stores the uploadurl.Issuer shared by the upload URL route and the MCP server.
	UploadURLIssuerKey = typedmap.NewTypedKey[*uploadurl.Issuer]("khi.google.com/init/upload-url-issuer")
)

// uploadURLPath is the route of upload URLs under the base path. The token follows it.
const uploadURLPath = "/api/v1/file-upload/"

// InitializerIDFileParameterUpload mounts Connect-RPC FileParameterUploadService onto Gin engine.
const InitializerIDFileParameterUpload coreinit.InitializerID = "khi.default/file-parameter-upload"

// FileParameterUploadInitializer initializes and registers the FileParameterUploadService handlers and the upload URL route.
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
		serverParams := coreinit.MustGet(ctx, ServerParametersKey)
		connectOpts, _ := coreinit.Get(ctx, ConnectHandlerOptionsKey)

		uploadFolder := "/tmp"
		if commonParams.UploadFileStoreFolder != nil {
			uploadFolder = *commonParams.UploadFileStoreFolder
		}

		chunkManager := chunkedupload.NewChunkSessionManager(uploadFolder)
		manager := upload.NewFileParameterUploadManager(uploadStore, chunkManager)
		fileUploadServer := serverapiv1.NewFileParameterUploadServiceServer(manager)
		fileUploadPath, fileUploadHandler := apiv1connect.NewFileParameterUploadServiceHandler(fileUploadServer, connectOpts...)
		coreinit.RegisterConnectServiceHandler(router, basePath, fileUploadPath, fileUploadHandler)

		issuer := uploadurl.NewIssuer(
			uploadURLBase(*serverParams.Host, *serverParams.Port, basePath),
			int64(*serverParams.MaxUploadFileSizeInBytes),
			uploadurl.DefaultTTL,
		)
		uploadURLHandler := uploadurl.NewHandler(issuer, manager)
		router.PUT(uploadURLPath+":token", func(c *gin.Context) {
			uploadURLHandler.ServeUpload(c.Writer, c.Request, c.Param("token"))
		})
		coreinit.Set(ctx, UploadURLIssuerKey, issuer)

		return nil
	},
}

// uploadURLBase returns the absolute URL that upload URL tokens are appended to.
// Upload URLs are used by clients on the same machine as the server, so wildcard listen addresses,
// which clients cannot connect to, are replaced with the loopback address.
func uploadURLBase(host string, port int, basePath string) string {
	switch host {
	case "", "0.0.0.0", "::":
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(port)) + basePath + uploadURLPath
}

func init() {
	coreinit.RegisterInitializer(FileParameterUploadInitializer)
}
