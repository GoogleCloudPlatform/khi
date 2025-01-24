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

package httpclient_test

import (
	"context"
	"net/http"

	"github.com/GoogleCloudPlatform/khi/pkg/common/httpclient"
)

type HttpClientSpyResponse[T any] struct {
	Response T
	Error    error
}

type HttpClientSpy[T any] struct {
	Results  []*HttpClientSpyResponse[T]
	Requests []*http.Request
}

func NewHttpClientSpyResponse[T any](response T, err error) *HttpClientSpyResponse[T] {
	return &HttpClientSpyResponse[T]{
		Response: response,
		Error:    err,
	}
}

func NewHttpClientSpy[T any](responses ...*HttpClientSpyResponse[T]) *HttpClientSpy[T] {
	return &HttpClientSpy[T]{
		Results:  responses,
		Requests: make([]*http.Request, 0),
	}
}

// DoWithContext implements httpclient.HttpClient.
func (h *HttpClientSpy[T]) DoWithContext(ctx context.Context, request *http.Request) (T, error) {
	h.Requests = append(h.Requests, request)
	callIndex := len(h.Requests) - 1
	if callIndex >= len(h.Results) {
		return *new(T), nil
	}
	return h.Results[callIndex].Response, h.Results[callIndex].Error
}

var _ httpclient.HTTPClient[any] = (*HttpClientSpy[any])(nil)
