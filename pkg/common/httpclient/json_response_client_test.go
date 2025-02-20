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
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

type mockHttpClient struct {
	Response *http.Response
	Error    error
}

// DoWithContext implements HttpClient.
func (m *mockHttpClient) DoWithContext(ctx context.Context, request *http.Request) (*http.Response, error) {
	if m.Error == nil {
		return m.Response, nil
	} else {
		return nil, m.Error
	}
}

var _ HTTPClient[*http.Response] = (*mockHttpClient)(nil)

func TestDoWithContext(t *testing.T) {
	type testJsonType struct {
		Foo string `json:"foo"`
		Bar string `json:"bar"`
	}
	jsonClient := NewJsonResponseHttpClient[testJsonType](&mockHttpClient{
		Response: &http.Response{
			Body: io.NopCloser(strings.NewReader(`{
  "foo":"foo-val",
  "bar":"bar-val"
}`)),
		},
	})
	result, _, err := jsonClient.DoWithContext(context.Background(), &http.Request{})
	if err != nil {
		t.Errorf("unexpected err:%s", err.Error())
	}
	if diff := cmp.Diff(&testJsonType{
		Foo: "foo-val",
		Bar: "bar-val",
	}, result); diff != "" {
		t.Errorf("response is not matching with the expected value\n%s", diff)
	}
}
