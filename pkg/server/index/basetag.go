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

package index

import (
	"fmt"

	"github.com/GoogleCloudPlatform/khi/pkg/parameters"
)

// BaseTagGenerator generates the <base> tag on index.html.
// This allow backend to change the base path of api requests from frontend to allow KHI placed under specified path.
type BaseTagGenerator struct {
}

// GenerateTags implements IndexTagGenerator.
func (b *BaseTagGenerator) GenerateTags() []string {
	basePath := "/"
	if parameters.Server.FrontendResourceBasePath != nil {
		basePath = *parameters.Server.FrontendResourceBasePath
	}
	return []string{fmt.Sprintf(`<base href="%s">`, basePath)}
}

var _ IndexTagGenerator = (*BaseTagGenerator)(nil)

// ServerBaseMetaTagGenerator generates a <meta> tag and that specify the base path for API client used in frontend.
// The path should be usually same as the base tag generated with BaseTagGenerator, but this can be different especially when Angular dev server needs to host the frontend files whereas Golang server works on another port.
type ServerBaseMetaTagGenerator struct {
}

// GenerateTags implements IndexTagGenerator.
func (s *ServerBaseMetaTagGenerator) GenerateTags() []string {
	basePath := ""
	if parameters.Server.BasePath != nil {
		basePath = *parameters.Server.BasePath
	}
	return []string{fmt.Sprintf(`<meta id="server-base-path" content="%s">`, basePath)}

}

var _ IndexTagGenerator = (*ServerBaseMetaTagGenerator)(nil)
