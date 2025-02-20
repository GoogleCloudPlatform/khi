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
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/common/httpclient"
	"github.com/GoogleCloudPlatform/khi/pkg/testutil"
	"github.com/google/go-cmp/cmp"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

type mockMDSResponseHttpClient struct {
	response string
	err      error
}

func newMockMDSResponseHttpClient(response string, err error) *mockMDSResponseHttpClient {
	return &mockMDSResponseHttpClient{
		response: response,
		err:      err,
	}
}

// DoWithContext implements httpclient.HttpClient.
func (m *mockMDSResponseHttpClient) DoWithContext(ctx context.Context, request *http.Request) (*http.Response, error) {
	if m.err != nil {
		return testutil.ResponseFromString(http.StatusOK, ""), m.err
	}
	return testutil.ResponseFromString(http.StatusOK, m.response), nil
}

var _ httpclient.HTTPClient[*http.Response] = (*mockMDSResponseHttpClient)(nil)

func TestMDSTokenResolver(t *testing.T) {
	tests := []struct {
		name             string
		client           *httpclient.JSONReponseHttpClient[MDSResponse]
		wantExpireAround time.Time
		want             string
		wantErr          bool
		expiredToken     map[string]interface{}
	}{
		{
			name:             "MDSTokenResolver should return the token from the metadata server",
			client:           httpclient.NewJsonResponseHttpClient[MDSResponse](newMockMDSResponseHttpClient("{ \"access_token\": \"test-fake-token\",\"expires_in\": 180}", nil)),
			want:             "test-fake-token",
			wantErr:          false,
			wantExpireAround: time.Now().Add(time.Minute * 3),
		},
		{
			name:    "MDSTokenResolver should return error if client returns error",
			client:  httpclient.NewJsonResponseHttpClient[MDSResponse](newMockMDSResponseHttpClient("", errors.New("test-error"))),
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMetadataServerAccessTokenResolver(tt.client)
			got, err := m.Resolve(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("got %v, want %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if diff := cmp.Diff(got.RawToken, tt.want); diff != "" {
					t.Errorf("token mismatch (-got +want):\n%s", diff)
				}
				if !got.ValidAtLeastUntil.Before(tt.wantExpireAround.Add(time.Second*10)) || !got.ValidAtLeastUntil.After(tt.wantExpireAround.Add(-time.Second*10)) {
					t.Errorf("expire time far different from the expected time,got %v, want around %v", got.ValidAtLeastUntil, tt.wantExpireAround)
				}
			}
		})
	}
}
