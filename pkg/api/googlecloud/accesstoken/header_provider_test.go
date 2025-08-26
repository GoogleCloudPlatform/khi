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
	"net/http"
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/common/token"
)

func TestGCPAccessTokenHeaderProvider(t *testing.T) {
	testCases := []struct {
		name              string
		tokenStore        token.TokenStore
		wantAuthorization string
		wantError         bool
	}{
		{
			name:              "success",
			tokenStore:        token.NewBasicTokenStore("test", token.NewSpyTokenResolver(token.New("test-token"))),
			wantAuthorization: "Bearer test-token",
			wantError:         false,
		},
		{
			name:              "empty token",
			tokenStore:        token.NewBasicTokenStore("test", token.NewMockErrorTokenResolver()),
			wantAuthorization: "",
			wantError:         true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			p := NewHeaderProvider(tt.tokenStore)
			req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			err = p.AddHeader(req)
			if (err != nil) != tt.wantError {
				t.Errorf("GCPAccessTokenHeaderProvider.AddHeader() error = %v, wantError %v", err, tt.wantError)
			}
			if !tt.wantError {
				gotAuthorization := req.Header.Get("Authorization")
				if gotAuthorization != tt.wantAuthorization {
					t.Errorf("GCPAccessTokenHeaderProvider.AddHeader() = %v, want %v", gotAuthorization, tt.wantAuthorization)
				}
			}
		})
	}

}
