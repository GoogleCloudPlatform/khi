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

package quotaproject

import (
	"net/http"

	"github.com/GoogleCloudPlatform/khi/pkg/common/httpclient"
)

// GCPQuotaProjectHeaderProvider is an implementation of HTTPHeaderProvider for setting quota project.
type GCPQuotaProjectHeaderProvider struct {
	QuotaProject string
}

func NewHeaderProvider(quotaProject string) *GCPQuotaProjectHeaderProvider {
	return &GCPQuotaProjectHeaderProvider{
		QuotaProject: quotaProject,
	}
}

// AddHeader implements httpclient.HTTPHeaderProvider.
func (g *GCPQuotaProjectHeaderProvider) AddHeader(req *http.Request) error {
	if g.QuotaProject == "" {
		return nil
	}
	req.Header.Set("X-Goog-User-Project", g.QuotaProject)
	return nil
}

var _ httpclient.HTTPHeaderProvider = (*GCPQuotaProjectHeaderProvider)(nil)
