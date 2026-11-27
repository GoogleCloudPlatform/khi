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
	"fmt"
	"log/slog"
	"sync"
	"time"
)

var _ TokenStore = &BasicTokenStore{}

// TokenStore manages caching and refreshing of authentication tokens.
type TokenStore interface {
	// GetType returns the token type identifier.
	GetType() string
	// GetToken returns the current token. It can come from cache or newly resolved from TokenResolver.
	GetToken(ctx context.Context) (*Token, error)
	// IsTokenValidityAssured returns true if this token is assured to be valid now.
	// When a permission error happens, refresher may attempt to refresh tokens. The refresh is only needed when the token can be expired(when we don't know the expiration time) or the token is actually expired.
	IsTokenValidityAssured(ctx context.Context) bool
	// RefreshToken refreshes the token returned from GetToken().
	RefreshToken(ctx context.Context) error
}

// BasicTokenStore provides feature to refresh token and return cached token.
// BasicTokenStore memory the expired tokens and it calls resolvers in order to get new token after MarkTokenExpired called.
type BasicTokenStore struct {
	tokenType             string
	resolver              TokenResolver
	tokenLock             sync.RWMutex
	lastToken             *Token
	lastTokenRefreshError error
	initialRefreshOnce    sync.Once

	refreshMu       sync.Mutex
	inFlightRefresh *refreshFlight
}

type refreshFlight struct {
	done chan struct{}
	err  error
}

// NewBasicTokenStore creates a BasicTokenStore for the given tokenType and resolver.
func NewBasicTokenStore(tokenType string, resolver TokenResolver) *BasicTokenStore {
	return &BasicTokenStore{
		tokenType: tokenType,
		resolver:  resolver,
	}
}

// GetType implements TokenStore.
func (b *BasicTokenStore) GetType() string {
	return b.tokenType
}

// GetToken implements TokenStore.
func (b *BasicTokenStore) GetToken(ctx context.Context) (*Token, error) {
	b.initialRefreshOnce.Do(func() {
		_ = b.resolveAndStoreToken(ctx)
	})
	b.tokenLock.RLock()
	defer b.tokenLock.RUnlock()
	return b.lastToken, b.lastTokenRefreshError
}

// IsTokenValidityAssured implements TokenStore.
func (b *BasicTokenStore) IsTokenValidityAssured(ctx context.Context) bool {
	b.tokenLock.RLock()
	defer b.tokenLock.RUnlock()
	return b.lastToken != nil && b.lastToken.ValidAtLeastUntil.After(time.Now())
}

// RefreshToken implements TokenStore.
// Concurrent RefreshToken calls share a single in-flight resolver call and wait until it completes or ctx is canceled.
func (b *BasicTokenStore) RefreshToken(ctx context.Context) error {
	b.refreshMu.Lock()
	if flight := b.inFlightRefresh; flight != nil {
		b.refreshMu.Unlock()
		select {
		case <-flight.done:
			return flight.err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	flight := &refreshFlight{done: make(chan struct{})}
	b.inFlightRefresh = flight
	b.refreshMu.Unlock()

	defer func() {
		b.refreshMu.Lock()
		b.inFlightRefresh = nil
		close(flight.done)
		b.refreshMu.Unlock()
	}()

	flight.err = b.resolveAndStoreToken(ctx)
	return flight.err
}

func (b *BasicTokenStore) resolveAndStoreToken(ctx context.Context) error {
	slog.DebugContext(ctx, fmt.Sprintf("Current token for %s is expired. Refreshing a new token", b.tokenType))
	token, err := b.resolver.Resolve(ctx)
	b.tokenLock.Lock()
	defer b.tokenLock.Unlock()
	if err != nil {
		b.lastToken = nil
		b.lastTokenRefreshError = err
		return err
	}
	b.lastToken = token
	b.lastTokenRefreshError = nil
	return nil
}
