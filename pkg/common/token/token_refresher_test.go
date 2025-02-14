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
	"time"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestNewMultiTokenStoreRefresher(t *testing.T) {
	refresher := NewMultiTokenStoreRefresher(NewBasicTokenStore("foo", NewSpyTokenResolver()), NewBasicTokenStore("bar", NewSpyTokenResolver()))

	if refresher.nextStoreIndexToRefresh != 0 {
		t.Errorf("got %d, want %d", refresher.nextStoreIndexToRefresh, 0)
	}
}

func TestMultiTokenStoreRefresher_Refresh(t *testing.T) {
	expireOnFuture := time.Date(2300, time.January, 1, 0, 0, 0, 0, time.UTC)
	testCase := []struct {
		name                            string
		stores                          []TokenStore
		nextStoreIndexToRefreshFirst    int
		nextStoreIndexToRefreshExpected int
		expectedRawTokens               []string
		wantErr                         bool
	}{
		{
			name: "refreshing the first token",
			stores: []TokenStore{
				NewBasicTokenStore("store1", NewSpyTokenResolver(New("foo"), New("bar"))),
				NewBasicTokenStore("store2", NewSpyTokenResolver(New("foo"), New("bar"))),
			},
			nextStoreIndexToRefreshFirst:    0,
			nextStoreIndexToRefreshExpected: 1,
			wantErr:                         false,
			expectedRawTokens: []string{
				"bar", "foo",
			},
		},
		{
			name: "refreshing the last token",
			stores: []TokenStore{
				NewBasicTokenStore("store1", NewSpyTokenResolver(New("foo"), New("bar"))),
				NewBasicTokenStore("store2", NewSpyTokenResolver(New("foo"), New("bar"))),
			},
			nextStoreIndexToRefreshFirst:    1,
			nextStoreIndexToRefreshExpected: 0,
			wantErr:                         false,
			expectedRawTokens: []string{
				"foo", "bar",
			},
		},
		{
			name: "refreshing the last token because the first token is not yet expired",
			stores: []TokenStore{
				NewBasicTokenStore("store1", NewSpyTokenResolver(NewWithExpiry("foo", expireOnFuture), New("bar"))),
				NewBasicTokenStore("store2", NewSpyTokenResolver(New("foo"), New("bar"))),
			},
			nextStoreIndexToRefreshFirst:    0,
			nextStoreIndexToRefreshExpected: 0,
			wantErr:                         false,
			expectedRawTokens: []string{
				"foo", "bar",
			},
		},
		{
			name: "return error when all store has valid tokens",
			stores: []TokenStore{
				NewBasicTokenStore("store1", NewSpyTokenResolver(NewWithExpiry("foo", expireOnFuture))),
				NewBasicTokenStore("store1", NewSpyTokenResolver(NewWithExpiry("foo", expireOnFuture))),
			},
			wantErr:                         true,
			nextStoreIndexToRefreshFirst:    0,
			nextStoreIndexToRefreshExpected: 0,
			expectedRawTokens:               []string{},
		},
	}
	for _, tt := range testCase {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMultiTokenStoreRefresher(tt.stores...)
			store.nextStoreIndexToRefresh = tt.nextStoreIndexToRefreshFirst
			for _, store := range tt.stores {
				_, _ = store.GetToken(context.Background())
			}

			err := store.Refresh(context.Background())
			if tt.wantErr {
				if err == nil {
					t.Error("got nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("got %v, want nil", err)
				}
				if tt.nextStoreIndexToRefreshExpected != store.nextStoreIndexToRefresh {
					t.Errorf("got %d, want %d", store.nextStoreIndexToRefresh, tt.nextStoreIndexToRefreshExpected)
				}
				if len(tt.expectedRawTokens) != len(tt.stores) {
					t.Errorf("got %d, want %d", len(tt.expectedRawTokens), len(tt.stores))
				}

				for i, store := range tt.stores {
					token, err := store.GetToken(context.Background())

					if err != nil {
						t.Errorf("got %v, want nil", err)
					}
					if tt.expectedRawTokens[i] != token.RawToken {
						t.Errorf("got %q, want %q", token.RawToken, tt.expectedRawTokens[i])
					}
				}
			}
		})
	}
}
