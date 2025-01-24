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

package accesstoken

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/GoogleCloudPlatform/khi/pkg/common/httpclient"
	"github.com/GoogleCloudPlatform/khi/pkg/common/token"
)

// GCPAccessTokenHeaderProvider is an implementation of HTTPHeaderProvider for access token.
type GCPAccessTokenHeaderProvider struct {
	AccessToken token.TokenStore
}

func NewHeaderProvider(tokenStore token.TokenStore) *GCPAccessTokenHeaderProvider {
	return &GCPAccessTokenHeaderProvider{
		AccessToken: tokenStore,
	}
}

// AddHeader implements httpclient.HTTPHeaderProvider.
func (a *GCPAccessTokenHeaderProvider) AddHeader(req *http.Request) error {
	token, err := a.AccessToken.GetToken(req.Context())
	if err != nil {
		return err
	}
	if token == nil {
		return errors.New("access token is empty")
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.RawToken))
	return nil
}

var _ httpclient.HTTPHeaderProvider = (*GCPAccessTokenHeaderProvider)(nil)
