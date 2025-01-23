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

type SpyTokenResolver struct {
	tokenResponse      []*Token
	callCount          int
	delayInMillisecond int
}

func NewSpyTokenResolverWithDelay(delayInMillisecond int, tokenResponse ...*Token) *SpyTokenResolver {
	return &SpyTokenResolver{
		tokenResponse:      tokenResponse,
		delayInMillisecond: delayInMillisecond,
	}
}

func NewSpyTokenResolver(tokenResponse ...*Token) *SpyTokenResolver {
	return &SpyTokenResolver{
		tokenResponse: tokenResponse,
	}
}

func (m *SpyTokenResolver) Resolve(ctx context.Context) (*Token, error) {
	if m.callCount < len(m.tokenResponse) {
		m.callCount++
		time.Sleep(time.Duration(m.delayInMillisecond * int(time.Millisecond)))
		return m.tokenResponse[m.callCount-1], nil
	} else {
		return nil, fmt.Errorf("no expected token response supplied")
	}
}

var _ TokenResolver = &SpyTokenResolver{}

type MockErrorTokenResolver struct{}

func NewMockErrorTokenResolver() *MockErrorTokenResolver {
	return &MockErrorTokenResolver{}
}

// Resolve implements TokenResolver.
func (m *MockErrorTokenResolver) Resolve(ctx context.Context) (*Token, error) {
	return nil, errors.New("test error")
}

var _ TokenResolver = &MockErrorTokenResolver{}
