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
)

var ErrNoValidTokenResolved = errors.New("no valid token returned")

type MultiTokenResolver struct {
	resolvers []TokenResolver
}

func NewMultiTokenResolver(resolvers ...TokenResolver) *MultiTokenResolver {
	return &MultiTokenResolver{
		resolvers: resolvers,
	}
}

// Resolve implements TokenResolver.
func (m *MultiTokenResolver) Resolve(ctx context.Context) (*Token, error) {
	resultErrors := []error{
		ErrNoValidTokenResolved,
	}
	for _, resolver := range m.resolvers {
		token, err := resolver.Resolve(ctx)
		if err != nil {
			resultErrors = append(resultErrors, err)
			continue
		}
		return token, nil
	}
	return nil, errors.Join(resultErrors...)
}

var _ TokenResolver = &MultiTokenResolver{}
