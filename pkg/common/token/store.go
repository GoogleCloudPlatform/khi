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
	"log/slog"
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/khi/pkg/task"
)

var ErrNoNewTokenResolved = errors.New("no new token resolved")

type TokenStore interface {
	task.CachableDependency
	GetType() string
	// GetToken returns the current token. It can come from cache or newly resolved from TokenResolver.
	GetToken(ctx context.Context) (*Token, error)
	// IsTokenValidityAssured returns true if this token is assured to be valid now.
	// When a permission error happens, refresher may attempt to refresh tokens. The refresh is only needed when the token can be expired(when we don't know the expiration time) or the token is actually expired.
	IsTokenValidityAssured(ctx context.Context) bool
	// RefreshToken refreshes the token returned from GetToken()
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
}

// Digest implements TokenStore.
func (b *BasicTokenStore) Digest() string {
	return b.tokenType
}

func NewBasicTokenStore(tokenType string, resolver TokenResolver) *BasicTokenStore {
	return &BasicTokenStore{
		tokenType: tokenType,
		resolver:  resolver,
	}
}

func (b *BasicTokenStore) GetType() string {
	return b.tokenType
}

// GetToken implements TokenStore.
func (b *BasicTokenStore) GetToken(ctx context.Context) (*Token, error) {
	defer b.tokenLock.RUnlock()
	b.tokenLock.RLock()
	if b.lastToken == nil {
		b.tokenLock.RUnlock()
		b.RefreshToken(ctx)
		b.tokenLock.RLock()
	}
	return b.lastToken, b.lastTokenRefreshError
}

func (b *BasicTokenStore) IsTokenValidityAssured(ctx context.Context) bool {
	return b.lastToken != nil && b.lastToken.ValidAtLeastUntil.After(time.Now())
}

// RefreshToken implements TokenStore.
// A token store can be referenced from multiple http client and these can call RefreshToken multiple times in parallel just after token expiration.
// RefreshToken() will ignore the refresh request when it can't acquire the write lock and wait until the lock to be acquired.
func (b *BasicTokenStore) RefreshToken(ctx context.Context) error {
	defer b.tokenLock.Unlock()
	if b.tokenLock.TryLock() {
		// Only the thread acquiring the lock will request the actual token refresh.
		return b.refreshTokenWithoutLock(ctx)
	} else {
		b.tokenLock.Lock()
	}
	return nil
}

func (b *BasicTokenStore) refreshTokenWithoutLock(ctx context.Context) error {
	slog.DebugContext(ctx, fmt.Sprintf("Current token for %s is expired. Refreshing a new token", b.tokenType))
	token, err := b.resolver.Resolve(ctx)
	if err != nil {
		b.lastToken = nil
		b.lastTokenRefreshError = err
		return err
	}
	b.lastToken = token
	b.lastTokenRefreshError = nil
	return nil
}

var _ TokenStore = &BasicTokenStore{}
