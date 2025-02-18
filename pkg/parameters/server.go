// Copyright 2024 Google LLC
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

package parameters

import "github.com/GoogleCloudPlatform/khi/pkg/common/flag"

var Server *ServerParameters = &ServerParameters{}

type ServerParameters struct {
	// ViewerMode limits the KHI feature to query logs with the backend. When it is true, KHI is only serve the frontend to open KHI files.
	ViewerMode *bool
	// Port is the port number where KHI server listens.
	Port *int
	// Host is the host address where KHI server serves.
	Host *string
	// BasePath specifies the base address of API endpoints. This path always ends with `/`.
	BasePath *string
	// FrontendResourceBasePath is another base address only for frontend assets. If this value is not set, this uses the BasePath value by default.
	FrontendResourceBasePath *string
	// FrontendAssetFolder is the root folder of the assets used in frontend including index.html.
	FrontendAssetFolder *string
	// MaxUploadFileSizeInBytes is the maximum limit of uploaded file. Server returns 400 when the request exceeds it.
	MaxUploadFileSizeInBytes *int
}

// PostProcess implements ParameterStore.
func (s *ServerParameters) PostProcess() error {
	ensureEndsWithSlash(s.BasePath)
	if *s.FrontendResourceBasePath == "" {
		*s.FrontendResourceBasePath = *s.BasePath
	}
	ensureEndsWithSlash(s.FrontendResourceBasePath)
	return nil
}

// Prepare implements ParameterStore.
func (s *ServerParameters) Prepare() error {
	s.ViewerMode = flag.Bool("viewer-mode", false, "Limits the KHI feature to query logs with the backend. When it is true, KHI is only serve the frontend to open KHI files.", "KHI_VIEWER_MODE")
	s.Port = flag.Int("port", 8080, "The port number where KHI server listens.", "PORT")
	s.Host = flag.String("host", "localhost", "The host address where KHI server serves.", "HOST")
	s.BasePath = flag.String("base-path", "/", "The base address of API endpoints.", "KHI_BASE_PATH")
	s.FrontendResourceBasePath = flag.String("frontend-resource-base-path", "", "Another base address only for frontend assets. If this value is not set, this uses `--base-path` value by default.", "KHI_FRONTEND_RESOURCE_PATH")
	s.FrontendAssetFolder = flag.String("frontend-asset-folder", "./web", "The root folder of the assets used in frontend including index.html.", "KHI_FRONTEND_ASSET_FOLDER")
	s.MaxUploadFileSizeInBytes = flag.Int("max-upload-file-size-in-bytes", 1024*1024*1024, "The maximum limit of uploaded file. Server returns 400 when the request exceeds it.", "")
	return nil
}

func ensureEndsWithSlash(v *string) {
	if *v != "" && (*v)[len(*v)-1] != '/' {
		*v += "/"
	}
}

var _ ParameterStore = (*ServerParameters)(nil)
