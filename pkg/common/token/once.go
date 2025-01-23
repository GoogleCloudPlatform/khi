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
)

// OnceTokenResolver resolves the token from somewhere only onetime. This is mainly used for resolving token from CLI arguments or environment variables.
type OnceTokenResolver struct {
	// resolver can be called twice if the resolver can't return token at the previous call.
	resolver func() string
	// tokenResolved is set to true once this resolver returns a token.
	tokenResolved bool
}

func NewOnceTokenResolver(resolver func() string) *OnceTokenResolver {
	return &OnceTokenResolver{
		resolver:      resolver,
		tokenResolved: false,
	}
}

// Resolve implements TokenResolver.
func (e *OnceTokenResolver) Resolve(ctx context.Context) (*Token, error) {
	if !e.tokenResolved {
		token := e.resolver()
		if token != "" {
			e.tokenResolved = true
			return New(token), nil
		}
	}
	if e.tokenResolved {
		return nil, ErrNoNewTokenResolved
	}
	return nil, ErrNoValidTokenResolved
}

var _ TokenResolver = (*OnceTokenResolver)(nil)
