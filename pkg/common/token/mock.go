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
	"time"
)

var (
	_ TokenResolver = &SpyTokenResolver{}
	_ TokenResolver = &MockErrorTokenResolver{}
)

// SpyTokenResolver is a mock TokenResolver that returns predefined tokens with an optional delay.
type SpyTokenResolver struct {
	tokenResponse      []*Token
	callCount          int
	delayInMillisecond int
}

// MockErrorTokenResolver is a mock TokenResolver that always returns an error.
type MockErrorTokenResolver struct{}

// NewSpyTokenResolverWithDelay creates a SpyTokenResolver that sleeps for delayInMillisecond before returning each token.
func NewSpyTokenResolverWithDelay(delayInMillisecond int, tokenResponse ...*Token) *SpyTokenResolver {
	return &SpyTokenResolver{
		tokenResponse:      tokenResponse,
		delayInMillisecond: delayInMillisecond,
	}
}

// NewSpyTokenResolver creates a SpyTokenResolver that returns the given tokenResponse sequence.
func NewSpyTokenResolver(tokenResponse ...*Token) *SpyTokenResolver {
	return &SpyTokenResolver{
		tokenResponse: tokenResponse,
	}
}

// Resolve implements TokenResolver.
func (m *SpyTokenResolver) Resolve(ctx context.Context) (*Token, error) {
	if m.callCount < len(m.tokenResponse) {
		m.callCount++
		time.Sleep(time.Duration(m.delayInMillisecond * int(time.Millisecond)))
		return m.tokenResponse[m.callCount-1], nil
	} else {
		return nil, fmt.Errorf("no expected token response supplied")
	}
}

// NewMockErrorTokenResolver creates a MockErrorTokenResolver.
func NewMockErrorTokenResolver() *MockErrorTokenResolver {
	return &MockErrorTokenResolver{}
}

// Resolve implements TokenResolver.
func (m *MockErrorTokenResolver) Resolve(ctx context.Context) (*Token, error) {
	return nil, errors.New("test error")
}
