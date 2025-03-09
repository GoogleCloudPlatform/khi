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
	"context"
	"net/http"

	"github.com/GoogleCloudPlatform/khi/pkg/common/httpclient"
)

type RequestGenerator = func(hasToken bool, nextPageToken string) (*http.Request, error)

// PageClient is utility to obtain all the resource from API returning page token.
type PageClient[T any] struct {
	client httpclient.HTTPClient[*http.Response]
}

func NewPageClient[T any](client httpclient.HTTPClient[*http.Response]) *PageClient[T] {
	return &PageClient[T]{
		client: client,
	}
}

func (p *PageClient[T]) GetAll(ctx context.Context, requestGenerator RequestGenerator, nextPageTokenMapper func(response *T) string) ([]*T, error) {
	result := make([]*T, 0)
	for nextPageToken := "-"; nextPageToken != ""; {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			request, err := requestGenerator(nextPageToken != "-", nextPageToken)
			if err != nil {
				return nil, err
			}
			client := httpclient.NewJsonResponseHttpClient[T](p.client)
			typedResponse, resp, err := client.DoWithContext(ctx, request)
			if err != nil {
				return nil, err
			}
			if resp != nil && resp.Body != nil {
				defer resp.Body.Close()
			}
			result = append(result, typedResponse)
			nextPageToken = nextPageTokenMapper(typedResponse)
		}
	}
	return result, nil
}
