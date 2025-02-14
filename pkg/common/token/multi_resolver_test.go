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

package token

import (
	"context"
	"testing"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestMultiTokenResolver_Resolve(t *testing.T) {
	testCases := []struct {
		name          string
		resolvers     []TokenResolver
		wantErr       bool
		expectedToken *Token
	}{
		{
			name:          "without any resolvers",
			resolvers:     make([]TokenResolver, 0),
			wantErr:       true,
			expectedToken: nil,
		},
		{
			name: "with the first successful resolver",
			resolvers: []TokenResolver{
				NewSpyTokenResolver(New("foo")),
			},
			wantErr:       false,
			expectedToken: New("foo"),
		},
		{
			name: "with the errornous resolver and successful resolver",
			resolvers: []TokenResolver{
				NewMockErrorTokenResolver(),
				NewSpyTokenResolver(New("foo")),
			},
			wantErr:       false,
			expectedToken: New("foo"),
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewMultiTokenResolver(tt.resolvers...)

			token, err := resolver.Resolve(context.Background())

			if tt.wantErr {
				if err == nil {
					t.Error("got nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("got %v, want nil", err)
				}
				if token.RawToken != tt.expectedToken.RawToken {
					t.Errorf("got raw token %s, want %s", token.RawToken, tt.expectedToken.RawToken)
				}
				if !token.ValidAtLeastUntil.Equal(tt.expectedToken.ValidAtLeastUntil) {
					t.Errorf("got expiry %v, want %v", token.ValidAtLeastUntil, tt.expectedToken.ValidAtLeastUntil)
				}
			}
		})
	}
}
