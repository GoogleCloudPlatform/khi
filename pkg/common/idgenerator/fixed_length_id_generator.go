// Copyright 2025 Google LLC
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

package idgenerator

import (
	"math/rand/v2"
)

const (
	defaultCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

// fixedLengthIDGenerator is a thread-safe ID generator that creates IDs with a fixed length.
type fixedLengthIDGenerator struct {
	length  int
	charset string
}

// NewFixedLengthIDGenerator creates a new FixedLengthIDGenerator.
func NewFixedLengthIDGenerator(length int) IDGenerator {
	return &fixedLengthIDGenerator{
		length:  length,
		charset: defaultCharset,
	}
}

// Generate returns a new unique ID.
func (g *fixedLengthIDGenerator) Generate() string {
	b := make([]byte, g.length)
	for i := range b {
		b[i] = g.charset[rand.N(len(g.charset))]
	}
	return string(b)
}
