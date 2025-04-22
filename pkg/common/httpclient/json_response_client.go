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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type JSONReponseHttpClient[T any] struct {
	client HTTPClient[*http.Response]
}

func NewJsonResponseHttpClient[T any](client HTTPClient[*http.Response]) *JSONReponseHttpClient[T] {
	return &JSONReponseHttpClient[T]{
		client: client,
	}
}

func (p *JSONReponseHttpClient[T]) DoWithContext(ctx context.Context, request *http.Request) (*T, *http.Response, error) {

	// In the current implementation, `JSONReponseHttpClient.DoWithContext` returns an `http.Response`,
	// so the client's user is expected to close the Response.Body (as required by `golangci-lint:bodyclose`).
	// But what is typically needed is the object deserialized from the JSON and the error (`err`).
	// TODO as a cleanup, refine the role and interface of this Client.

	response, err := p.client.DoWithContext(ctx, request)
	if err != nil {
		return nil, response, err
	}
	if response.StatusCode >= 400 {
		return nil, response, fmt.Errorf("%d:%s", response.StatusCode, response.Status)
	}
	responseData, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, response, err
	}
	result, err := p.parse(string(responseData))
	if err != nil {
		return nil, response, err
	}
	return result, response, err
}

func (p *JSONReponseHttpClient[T]) parse(body string) (*T, error) {
	if body == "" {
		return nil, fmt.Errorf("response is empty")
	}
	var typedResponse T
	err := json.Unmarshal([]byte(body), &typedResponse)
	if err != nil {
		return nil, err
	}
	return &typedResponse, nil
}
