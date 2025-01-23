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

package httpclient

import (
	"context"
	"net/http"
)

type BasicHttpClient struct {
	HeaderProvider []HTTPHeaderProvider
}

// BasicHttpClient implements HttpClient interface
var _ HTTPClient[*http.Response] = (*BasicHttpClient)(nil)

func NewBasicHttpClient() *BasicHttpClient {
	return &BasicHttpClient{
		HeaderProvider: []HTTPHeaderProvider{},
	}
}

// WithHeaderProvider creates a new BasicHttpClient with given header provider additionally.
func (b *BasicHttpClient) WithHeaderProvider(headerProvider ...HTTPHeaderProvider) *BasicHttpClient {
	client := NewBasicHttpClient()
	client.HeaderProvider = append(client.HeaderProvider, b.HeaderProvider...)
	b.HeaderProvider = append(b.HeaderProvider, headerProvider...)
	return b
}

// DoWithContext implements HttpClient.
func (b *BasicHttpClient) DoWithContext(ctx context.Context, request *http.Request) (*http.Response, error) {
	for _, headerProvider := range b.HeaderProvider {
		err := headerProvider.AddHeader(request)
		if err != nil {
			return nil, err
		}
	}
	req := request.WithContext(ctx)
	client := new(http.Client)
	return client.Do(req)
}
