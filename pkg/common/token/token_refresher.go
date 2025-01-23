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
)

type TokenRefresher interface {
	Refresh(ctx context.Context) error
}

type NopTokenRefresher struct {
}

// Refresh implements TokenRefresher.
func (n *NopTokenRefresher) Refresh(ctx context.Context) error {
	return nil
}

var _ TokenRefresher = (*NopTokenRefresher)(nil)

// MultiTokenStoreRefresher implements the TokenRefresher to refresh tokens in multiple stores.
// This rotate the next token store to be refreshed but it will ignore if the token is assured to be alive from the expiry time.
type MultiTokenStoreRefresher struct {
	tokenStores             []TokenStore
	nextStoreIndexToRefresh int
}

// Refresh implements TokenRefresher.
func (m *MultiTokenStoreRefresher) Refresh(ctx context.Context) error {
	for i := 0; i < len(m.tokenStores); i++ {
		store := m.tokenStores[m.nextStoreIndexToRefresh]
		m.nextStoreIndexToRefresh = (m.nextStoreIndexToRefresh + 1) % len(m.tokenStores)
		if store.IsTokenValidityAssured(ctx) {
			slog.DebugContext(ctx, fmt.Sprintf("Token for %s shouldn't be expired yet. ignoring token refresh.", store.GetType()))
			continue
		}
		err := store.RefreshToken(ctx)
		if err == nil {
			return nil
		}
		slog.DebugContext(ctx, fmt.Sprintf("token store %s couldn't refresh the new token. Skipping refresh this token store", m.tokenStores[i].GetType()))
	}
	return fmt.Errorf("there were no token store to refresh its token")
}

func NewMultiTokenStoreRefresher(tokenStores ...TokenStore) *MultiTokenStoreRefresher {
	return &MultiTokenStoreRefresher{
		tokenStores: tokenStores,
	}
}

var _ TokenRefresher = (*MultiTokenStoreRefresher)(nil)
