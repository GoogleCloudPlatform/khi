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
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

type blockingTokenResolver struct {
	started   chan struct{}
	release   chan struct{}
	token     *Token
	err       error
	callCount atomic.Int32
}

func (r *blockingTokenResolver) Resolve(ctx context.Context) (*Token, error) {
	if r.callCount.Add(1) == 1 && r.started != nil {
		close(r.started)
	}
	if r.release != nil {
		<-r.release
	}
	if r.err != nil {
		return nil, r.err
	}
	return r.token, nil
}

type funcTokenResolver struct {
	resolve func(ctx context.Context) (*Token, error)
}

func (r *funcTokenResolver) Resolve(ctx context.Context) (*Token, error) {
	return r.resolve(ctx)
}

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

// TestBasicTokenStore_RefreshTokenCallsResolverOnceInParallel deterministically verifies singleflight coalescing, follow-up refreshes, error propagation, and follower context cancellation.
func TestBasicTokenStore_RefreshTokenCallsResolverOnceInParallel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const concurrency = 100
		resolver := &blockingTokenResolver{
			started: make(chan struct{}),
			release: make(chan struct{}),
			token:   New("token-1"),
		}
		store := NewBasicTokenStore("foo", resolver)

		var completed atomic.Int32
		var wg sync.WaitGroup
		for range concurrency {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := store.RefreshToken(t.Context()); err != nil {
					t.Errorf("RefreshToken() unexpected error: %v", err)
				}
				completed.Add(1)
			}()
		}

		<-resolver.started
		synctest.Wait()

		if got := resolver.callCount.Load(); got != 1 {
			t.Errorf("in-flight resolver calls = %d, want 1", got)
		}
		if got := completed.Load(); got != 0 {
			t.Errorf("completed callers before release = %d, want 0", got)
		}

		close(resolver.release)
		wg.Wait()

		if got := resolver.callCount.Load(); got != 1 {
			t.Errorf("total resolver calls after first wave = %d, want 1", got)
		}
		if got := completed.Load(); got != concurrency {
			t.Errorf("completed callers = %d, want %d", got, concurrency)
		}

		// A follow-up refresh after the in-flight refresh completes invokes the resolver again.
		resolver.token = New("token-2")
		if err := store.RefreshToken(t.Context()); err != nil {
			t.Fatalf("second RefreshToken() unexpected error: %v", err)
		}
		if got := resolver.callCount.Load(); got != 2 {
			t.Errorf("resolver calls after second refresh = %d, want 2", got)
		}
	})
	verifyRefreshErrorPropagation(t)
	verifyRefreshFollowerCancel(t)
	verifyRefreshPreCanceledContext(t)
	verifyRefreshPanicSignalsFollowers(t)
}

func verifyRefreshPreCanceledContext(t *testing.T) {
	t.Helper()
	var calls atomic.Int32
	resolver := &funcTokenResolver{
		resolve: func(context.Context) (*Token, error) {
			calls.Add(1)
			return New("token"), nil
		},
	}
	store := NewBasicTokenStore("foo", resolver)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := store.RefreshToken(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("RefreshToken(canceled) error = %v, want context.Canceled", err)
	}
	if got := calls.Load(); got != 0 {
		t.Errorf("resolver calls on pre-canceled ctx = %d, want 0", got)
	}
}

func verifyRefreshPanicSignalsFollowers(t *testing.T) {
	t.Helper()
	synctest.Test(t, func(t *testing.T) {
		started := make(chan struct{})
		release := make(chan struct{})
		resolver := &funcTokenResolver{
			resolve: func(context.Context) (*Token, error) {
				close(started)
				<-release
				panic("boom")
			},
		}
		store := NewBasicTokenStore("foo", resolver)

		panicked := make(chan any, 1)
		go func() {
			defer func() {
				panicked <- recover()
			}()
			_ = store.RefreshToken(t.Context())
		}()
		<-started

		followerErrCh := make(chan error, 1)
		go func() {
			followerErrCh <- store.RefreshToken(t.Context())
		}()
		synctest.Wait()

		close(release)
		if r := <-panicked; r != "boom" {
			t.Errorf("leader panic = %v, want %q", r, "boom")
		}
		if err := <-followerErrCh; !errors.Is(err, errTokenRefreshAborted) {
			t.Errorf("follower error after leader panic = %v, want %v", err, errTokenRefreshAborted)
		}
	})
}

func verifyRefreshErrorPropagation(t *testing.T) {
	t.Helper()
	synctest.Test(t, func(t *testing.T) {
		wantErr := errors.New("resolver failed")
		resolver := &blockingTokenResolver{
			started: make(chan struct{}),
			release: make(chan struct{}),
			err:     wantErr,
		}
		store := NewBasicTokenStore("foo", resolver)

		errCh := make(chan error, 2)
		for range 2 {
			go func() {
				errCh <- store.RefreshToken(t.Context())
			}()
		}

		<-resolver.started
		synctest.Wait()
		close(resolver.release)

		for i := range 2 {
			if err := <-errCh; !errors.Is(err, wantErr) {
				t.Errorf("caller %d error = %v, want %v", i, err, wantErr)
			}
		}
		if got := resolver.callCount.Load(); got != 1 {
			t.Errorf("resolver calls = %d, want 1", got)
		}
	})
}

func verifyRefreshFollowerCancel(t *testing.T) {
	t.Helper()
	synctest.Test(t, func(t *testing.T) {
		resolver := &blockingTokenResolver{
			started: make(chan struct{}),
			release: make(chan struct{}),
			token:   New("token"),
		}
		store := NewBasicTokenStore("foo", resolver)

		leaderErrCh := make(chan error, 1)
		go func() {
			leaderErrCh <- store.RefreshToken(t.Context())
		}()
		<-resolver.started

		followerCtx, cancel := context.WithCancel(t.Context())
		followerErrCh := make(chan error, 1)
		go func() {
			followerErrCh <- store.RefreshToken(followerCtx)
		}()
		synctest.Wait()

		cancel()
		if err := <-followerErrCh; !errors.Is(err, context.Canceled) {
			t.Errorf("canceled follower error = %v, want context.Canceled", err)
		}

		close(resolver.release)
		if err := <-leaderErrCh; err != nil {
			t.Errorf("leader RefreshToken() unexpected error: %v", err)
		}
	})
}

// TestBasicTokenStore_GetTokenCallsResolverOnceInParallel deterministically verifies concurrent initial GetToken coalescing and self-healing retry after a transient error.
func TestBasicTokenStore_GetTokenCallsResolverOnceInParallel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const concurrency = 100
		resolver := &blockingTokenResolver{
			started: make(chan struct{}),
			release: make(chan struct{}),
			token:   New("token"),
		}
		store := NewBasicTokenStore("foo", resolver)

		var wg sync.WaitGroup
		for range concurrency {
			wg.Add(1)
			go func() {
				defer wg.Done()
				token, err := store.GetToken(t.Context())
				if err != nil {
					t.Errorf("GetToken() unexpected error: %v", err)
					return
				}
				if token.RawToken != "token" {
					t.Errorf("GetToken().RawToken = %q, want %q", token.RawToken, "token")
				}
			}()
		}

		<-resolver.started
		synctest.Wait()
		if got := resolver.callCount.Load(); got != 1 {
			t.Errorf("in-flight resolver calls = %d, want 1", got)
		}

		close(resolver.release)
		wg.Wait()

		if got := resolver.callCount.Load(); got != 1 {
			t.Errorf("resolver calls = %d, want 1", got)
		}
		// Double-checked resolveToken returns cached token without re-invoking the resolver.
		tok, err := store.resolveToken(t.Context(), true)
		if err != nil || tok == nil || tok.RawToken != "token" || resolver.callCount.Load() != 1 {
			t.Errorf("resolveToken(onlyIfMissing=true) = (%v, %v, calls=%d), want (token, nil, calls=1)", tok, err, resolver.callCount.Load())
		}
	})

	// Initial resolution error is not permanently cached; a subsequent GetToken retries and succeeds.
	wantErr := errors.New("transient failure")
	var attempt atomic.Int32
	retryingStore := NewBasicTokenStore("foo", &funcTokenResolver{
		resolve: func(context.Context) (*Token, error) {
			if attempt.Add(1) == 1 {
				return nil, wantErr
			}
			return New("recovered"), nil
		},
	})
	if tok, err := retryingStore.GetToken(context.Background()); !errors.Is(err, wantErr) || tok != nil {
		t.Fatalf("first GetToken() = (%v, %v), want (nil, %v)", tok, err, wantErr)
	}
	if tok, err := retryingStore.GetToken(context.Background()); err != nil || tok == nil || tok.RawToken != "recovered" {
		t.Errorf("retry GetToken() = (%v, %v), want (recovered, nil)", tok, err)
	}
}

// TestBasicTokenStore_ConcurrentAccess exercises GetToken, RefreshToken, and IsTokenValidityAssured concurrently under -race without wall-clock assumptions.
func TestBasicTokenStore_ConcurrentAccess(t *testing.T) {
	var seq atomic.Int64
	resolver := &funcTokenResolver{
		resolve: func(context.Context) (*Token, error) {
			n := seq.Add(1)
			return NewWithExpiry(fmt.Sprintf("tok-%d", n), time.Now().Add(time.Hour)), nil
		},
	}
	store := NewBasicTokenStore("foo", resolver)

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for range 50 {
				tok, err := store.GetToken(context.Background())
				if err != nil || tok == nil || tok.RawToken == "" {
					t.Errorf("GetToken() = (%v, %v), want non-empty token and nil err", tok, err)
					return
				}
				_ = store.IsTokenValidityAssured(context.Background())
			}
		}()
		go func() {
			defer wg.Done()
			for range 50 {
				if err := store.RefreshToken(context.Background()); err != nil {
					t.Errorf("RefreshToken() unexpected error: %v", err)
					return
				}
			}
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
