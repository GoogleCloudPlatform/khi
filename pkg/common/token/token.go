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

import "time"

// Token contains the token string like access token.
type Token struct {
	// RawToken is the actual token in string representation.
	RawToken string
	// ValidAtLeastUntil holds the expiration time when the information is available.
	// The token refreshers won't refresh token if this value exists and later than now.
	// The default value will be `January 1, year 1, 00:00:00.000000000 UTC` and it is earlier than the possible time.Now().
	// The default value naturally means the token is expired.
	ValidAtLeastUntil time.Time
}

// NewWithExpiry instanciate a new Token from the raw token string and expiration time.
func NewWithExpiry(rawToken string, validAtLeastUntil time.Time) *Token {
	return &Token{
		RawToken:          rawToken,
		ValidAtLeastUntil: validAtLeastUntil,
	}
}

// New instanciate a new Token without expiration time.
func New(rawToken string) *Token {
	return &Token{
		RawToken: rawToken,
	}
}
