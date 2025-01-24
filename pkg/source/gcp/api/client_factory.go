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

package api

import (
	"github.com/GoogleCloudPlatform/khi/pkg/common/httpclient"
	"github.com/GoogleCloudPlatform/khi/pkg/common/token"
	"github.com/GoogleCloudPlatform/khi/pkg/source/gcp/api/accesstoken"
)

type GCPClientFactory struct {
	HeaderProviders []httpclient.HTTPHeaderProvider
	TokenStores     []token.TokenStore
}

func NewGCPClientFactory() *GCPClientFactory {
	return &GCPClientFactory{
		TokenStores:     []token.TokenStore{},
		HeaderProviders: []httpclient.HTTPHeaderProvider{},
	}
}

// NewClient instanciate a new GCPClient from current factory config.
func (f *GCPClientFactory) NewClient() (GCPClient, error) {
	return NewGCPClient(token.NewMultiTokenStoreRefresher(f.TokenStores...), f.HeaderProviders)
}

// RegisterHeaderProvider adds a new HeaderProvider on factory config.
func (f *GCPClientFactory) RegisterHeaderProvider(provider httpclient.HTTPHeaderProvider) {
	f.HeaderProviders = append(f.HeaderProviders, provider)
}

// RegisterRefreshableTokenStore adds a refreshable token store. The token will be refreshed when permission related error happens.
func (f *GCPClientFactory) RegisterRefreshableTokenStore(store token.TokenStore) {
	f.TokenStores = append(f.TokenStores, store)
}

var DefaultGCPClientFactory *GCPClientFactory = NewGCPClientFactory()

// set the default header providers on the default factory.
func init() {
	DefaultGCPClientFactory.RegisterHeaderProvider(accesstoken.NewHeaderProvider(accesstoken.DefaultAccessTokenStore))
	DefaultGCPClientFactory.RegisterRefreshableTokenStore(accesstoken.DefaultAccessTokenStore)
}
