// Copyright 2026 Google LLC
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

package id

import (
	"sync/atomic"

	"golang.org/x/sys/cpu"
)

// Namespace represents a namespace for generating unique IDs.
type Namespace uint32

// ServerStringIDBase is the base offset for server-only string IDs to prevent collisions with client string IDs.
const ServerStringIDBase uint32 = 0x80000000

const (
	// String is the namespace for string IDs.
	String Namespace = iota
	// ServerString is the namespace for server-only string IDs.
	ServerString
	// FieldSet is the namespace for field set IDs.
	FieldSet
	// Struct is the namespace for struct IDs.
	Struct
	// TimelinePath is the namespace for timeline path IDs.
	TimelinePath
	// TimelineItems is the namespace for timeline items IDs.
	TimelineItems
	// TemporaryLog is the namespace for parser-side temporary log IDs.
	TemporaryLog
	// Log is the namespace for finalized/serialized log IDs.
	Log

	// namespaceMax is the sentinel value for the maximum number of namespaces.
	namespaceMax
)

type nsCounter struct {
	value atomic.Uint32
	_     cpu.CacheLinePad
}

// Generator generates unique IDs for different namespaces.
// It is safe for concurrent use.
type Generator struct {
	counters [namespaceMax]nsCounter
}

// NewGenerator creates a new Generator initialized with starting counter values.
func NewGenerator() *Generator {
	g := &Generator{}
	g.counters[ServerString].value.Store(ServerStringIDBase)
	return g
}

// New allocates a fresh uint32 ID in the given namespace.
// IDs start from 1 (or ServerStringIDBase + 1 for ServerString).
func (g *Generator) New(ns Namespace) uint32 {
	if ns >= namespaceMax {
		panic("invalid namespace")
	}
	return g.counters[ns].value.Add(1)
}

// Set sets the current value of the ID counter for the given namespace.
func (g *Generator) Set(ns Namespace, value uint32) {
	if ns >= namespaceMax {
		panic("invalid namespace")
	}
	g.counters[ns].value.Store(value)
}
