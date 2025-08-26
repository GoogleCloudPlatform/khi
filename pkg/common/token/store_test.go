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
	"sync"
	"testing"
	"time"
)

func TestBasicTokenStore_GetType(t *testing.T) {
	store := NewBasicTokenStore("foo", NewSpyTokenResolver(New("token")))

	if store.GetType() != "foo" {
		t.Errorf("got %q, want %q", store.GetType(), "foo")
	}
}

func TestBasicTokenStore_GetTokenOnlyCallsResolverOnce(t *testing.T) {
	resolver := NewSpyTokenResolver(New("token"))
	store := NewBasicTokenStore("foo", resolver)

	token, err := store.GetToken(context.Background())

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if token.RawToken != "token" {
		t.Errorf("got %q, want %q", token.RawToken, "token")
	}
	if resolver.callCount != 1 {
		t.Errorf("got %d resolver calls, want 1", resolver.callCount)
	}

	token2, err := store.GetToken(context.Background())
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if token2.RawToken != "token" {
		t.Errorf("got %q, want %q", token2.RawToken, "token")
	}
	if resolver.callCount != 1 {
		t.Errorf("got %d resolver calls, want 1", resolver.callCount)
	}
}

func TestBasicTokenStore_RefreshTokenCallsResolverOnceInParallel(t *testing.T) {
	wg := sync.WaitGroup{}
	for attempt := 0; attempt < 1000; attempt++ {
		wg.Add(1)
		go func() {
			resolver := NewSpyTokenResolverWithDelay(1000, New("token"))
			store := NewBasicTokenStore("foo", resolver)

			refreshWg := sync.WaitGroup{}
			for i := 0; i < 100; i++ {
				refreshWg.Add(1)
				go func() {
					err := store.RefreshToken(context.Background())
					if err != nil {
						t.Errorf("Unexpected error: %v", err)
					}
					refreshWg.Done()
				}()
			}
			refreshWg.Wait()

			if resolver.callCount != 1 {
				t.Errorf("got %d resolver calls, want 1", resolver.callCount)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestBasicTokenStore_GetTokenCallsResolverOnceInParallel(t *testing.T) {
	wg := sync.WaitGroup{}
	for attempt := 0; attempt < 1000; attempt++ {
		wg.Add(1)
		go func() {
			resolver := NewSpyTokenResolverWithDelay(1000, New("token"))
			store := NewBasicTokenStore("foo", resolver)

			refreshWg := sync.WaitGroup{}
			for i := 0; i < 100; i++ {
				refreshWg.Add(1)
				go func() {
					token, err := store.GetToken(context.Background())
					if err != nil {
						t.Errorf("Unexpected error: %v", err)
					}
					refreshWg.Done()

					if token.RawToken != "token" {
						t.Errorf("got %q, want %q", token.RawToken, "token")
					}
				}()
			}
			refreshWg.Wait()

			if resolver.callCount != 1 {
				t.Errorf("got %d resolver calls, want 1", resolver.callCount)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestBasicTokenStore_GetTokenReturnsErrorWhenTokenResolutionFails(t *testing.T) {
	resolver := NewMockErrorTokenResolver()
	store := NewBasicTokenStore("foo", resolver)

	_, err := store.GetToken(context.Background())

	if err == nil {
		t.Error("got nil, want error")
	}
}

func TestBasicTokenStore_RefreshTokenCallsResolverAndSet(t *testing.T) {
	resolver := NewSpyTokenResolver(New("token1"), New("token2"))
	store := NewBasicTokenStore("foo", resolver)

	_, _ = store.GetToken(context.Background())
	err1 := store.RefreshToken(context.Background())
	token2, err2 := store.GetToken(context.Background())

	if err1 != nil {
		t.Errorf("Unexpected error: %v", err1)
	}
	if token2.RawToken != "token2" {
		t.Errorf("got %q, want %q", token2.RawToken, "token2")
	}
	if err2 != nil {
		t.Errorf("Unexpected error: %v", err2)
	}
}

func TestBasicTokenStore_RefreshTokenReturnsErrorWhenTokenResolutionFails(t *testing.T) {
	resolver := NewMockErrorTokenResolver()
	store := NewBasicTokenStore("foo", resolver)

	err := store.RefreshToken(context.Background())

	if err == nil {
		t.Error("got nil, want error")
	}
}

func TestBasicTokenStore_ISValidityAssured(t *testing.T) {
	testCases := []struct {
		name   string
		expect bool
		store  *BasicTokenStore
	}{
		{
			name:   "without the last token",
			expect: false,
			store: &BasicTokenStore{
				lastToken: nil,
			},
		},
		{
			name:   "with a token without expiry",
			expect: false,
			store: &BasicTokenStore{
				lastToken: New("foo"),
			},
		},
		{
			name:   "with a token with non expired expiry",
			expect: true,
			store: &BasicTokenStore{
				lastToken: NewWithExpiry("foo", time.Date(3000, time.January, 1, 0, 0, 0, 0, time.UTC)),
			},
		},
		{
			name:   "with a token with expired expiry",
			expect: false,
			store: &BasicTokenStore{
				lastToken: NewWithExpiry("foo", time.Date(1900, time.January, 1, 0, 0, 0, 0, time.UTC)),
			},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.store.IsTokenValidityAssured(context.Background())

			if actual != tt.expect {
				t.Errorf("got %t, want %t", actual, tt.expect)
			}
		})
	}
}
