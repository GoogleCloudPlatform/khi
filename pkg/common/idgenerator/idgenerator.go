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

	"github.com/google/uuid"
)

// IDGenerator defines the interface for generating unique string IDs.
type IDGenerator interface {
	Generate() string
}

// UUIDGenerator generates IDs using the UUID standard.
// It is safe for concurrent use.
type UUIDGenerator struct{}

// NewUUIDGenerator creates a new UUIDGenerator.
func NewUUIDGenerator() *UUIDGenerator {
	return &UUIDGenerator{}
}

// Generate returns a new UUID string.
func (g *UUIDGenerator) Generate() string {
	return uuid.Must(uuid.NewUUID()).String()
}

var _ IDGenerator = (*UUIDGenerator)(nil)

// SequentialGenerator generates sequential, predictable IDs for testing.
// It is safe for concurrent use by multiple goroutines.
type SequentialGenerator struct {
	counter atomic.Uint64
	prefix  string
}

// NewSequentialGenerator creates a new thread-safe sequential ID generator.
func NewSequentialGenerator(prefix string) *SequentialGenerator {
	return &SequentialGenerator{prefix: prefix}
}

// Generate returns a new, unique sequential ID.
// This method uses a pointer receiver (*SequentialGenerator) to ensure that
// all goroutines operate on the same counter instance, not a copy.
func (g *SequentialGenerator) Generate() string {
	// Add atomically increments the counter and returns the new value.
	// This operation is guaranteed to be safe across multiple goroutines.
	count := g.counter.Add(1)
	return fmt.Sprintf("%s%d", g.prefix, count)
}

var _ IDGenerator = (*SequentialGenerator)(nil)
