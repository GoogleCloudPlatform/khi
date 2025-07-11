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
	"fmt"
	"sync/atomic"
)

// prefixIDGenerator is a thread-safe ID generator that creates IDs with a prefix.
type prefixIDGenerator struct {
	prefix  string
	counter uint64
}

// NewPrefixIDGenerator creates a new PrefixIDGenerator.
func NewPrefixIDGenerator(prefix string) IDGenerator {
	return &prefixIDGenerator{prefix: prefix}
}

// Generate returns a new unique ID.
func (g *prefixIDGenerator) Generate() string {
	id := atomic.AddUint64(&g.counter, 1)
	return fmt.Sprintf("%s%d", g.prefix, id)
}
