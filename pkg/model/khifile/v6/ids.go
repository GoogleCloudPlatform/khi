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

package khifilev6

import (
	"sync/atomic"

	"golang.org/x/sys/cpu"
)

// IDNamespace represents a namespace for generating unique IDs.
type IDNamespace uint32

const (
	// IDString is the namespace for string IDs.
	IDString IDNamespace = iota
	// IDFieldSet is the namespace for field set IDs.
	IDFieldSet

	// idNamespaceMax is the sentinel value for the maximum number of namespaces.
	idNamespaceMax
)

type nsCounter struct {
	value atomic.Uint32
	_     cpu.CacheLinePad
}

// IDGenerator generates unique IDs for different namespaces.
// It is safe for concurrent use.
// Note: This generates uint32 IDs instead of string IDs used in pkg/common/idgenerator
// to save memory and disk space in the file format.
type IDGenerator struct {
	counters [idNamespaceMax]nsCounter
}

// New allocates a fresh uint32 ID in the given namespace.
// IDs start from 1.
// Note: This method panics if the namespace is invalid.
func (g *IDGenerator) New(ns IDNamespace) uint32 {
	if ns >= idNamespaceMax {
		panic("invalid namespace")
	}
	return g.counters[ns].value.Add(1)
}

// Set sets the current value of the ID counter for the given namespace.
// Note: This method is safe from data races, but the outcome is non-deterministic
// if called concurrently with New. It is intended for initialization or single-threaded setup.
func (g *IDGenerator) Set(ns IDNamespace, value uint32) {
	if ns >= idNamespaceMax {
		panic("invalid namespace")
	}
	g.counters[ns].value.Store(value)
}
