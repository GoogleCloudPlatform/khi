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
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/httpclient"
)

type mockHttpClientForPageClient struct {
	CurrentCallCount int
	Requests         []*http.Request
	Response         []*http.Response
}

// DoWithContext implements HttpClient.
func (m *mockHttpClientForPageClient) DoWithContext(ctx context.Context, request *http.Request) (*http.Response, error) {
	response := m.Response[m.CurrentCallCount]
	m.Requests = append(m.Requests, request)
	m.CurrentCallCount++
	return response, nil
}

var _ httpclient.HTTPClient[*http.Response] = (*mockHttpClientForPageClient)(nil)

func TestPageClient(t *testing.T) {
	type TestResponseType struct {
		Value         string `json:"value"`
		NextPageToken string `json:"nextPageToken"`
	}
	mc := &mockHttpClientForPageClient{}
	pc := NewPageClient[TestResponseType](mc)
	mc.Response = append(mc.Response, &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("{\"value\": \"v1\", \"nextPageToken\": \"t1\"}")),
	})
	mc.Response = append(mc.Response, &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("{\"value\": \"v2\", \"nextPageToken\": \"t2\"}")),
	})
	mc.Response = append(mc.Response, &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("{\"value\": \"v3\"}")),
	})

	r, err := pc.GetAll(context.Background(), func(hasToken bool, pageToken string) (*http.Request, error) {
		if !hasToken {
			return http.NewRequest("GET", "https://example.com", nil)
		}
		return http.NewRequest("GET", fmt.Sprintf("https://example.com?nextPageToken=%s", pageToken), nil)
	}, func(response *TestResponseType) string {
		return response.NextPageToken
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(r) != 3 {
		t.Errorf("unexpected response count: %v", len(r))
	}
	if r[0].Value != "v1" {
		t.Errorf("unexpected response value: %v", r[0].Value)
	}
	if r[1].Value != "v2" {
		t.Errorf("unexpected response value: %v", r[1].Value)
	}
	if r[2].Value != "v3" {
		t.Errorf("unexpected response value: %v", r[2].Value)
	}
	if len(mc.Requests) != 3 {
		t.Errorf("unexpected request count: %v", len(mc.Requests))
	}
	if mc.Requests[0].URL.String() != "https://example.com" {
		t.Errorf("unexpected request url: %v", mc.Requests[0].URL.String())
	}
	if mc.Requests[1].URL.String() != "https://example.com?nextPageToken=t1" {
		t.Errorf("unexpected request url: %v", mc.Requests[1].URL.String())
	}
	if mc.Requests[2].URL.String() != "https://example.com?nextPageToken=t2" {
		t.Errorf("unexpected request url: %v", mc.Requests[2].URL.String())
	}
}
