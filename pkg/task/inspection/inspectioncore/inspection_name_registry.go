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

package inspectioncore

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

var (
	// ErrInspectionNameEmpty is returned when an inspection name is empty or whitespace only.
	ErrInspectionNameEmpty = errors.New("inspection name must not be empty")
	// ErrInspectionNameAlreadyInUse is returned when an inspection name is already reserved by another inspection.
	ErrInspectionNameAlreadyInUse = errors.New("inspection name is already in use")
)

// InspectionNameRegistry manages unique inspection names and their reservations across inspections.
type InspectionNameRegistry interface {
	// ResolveUniqueName returns baseName if it is not reserved by another inspection,
	// or appends a sequential suffix like (1), (2), etc. to return an unused name.
	// It does not modify any reservations.
	ResolveUniqueName(inspectionID string, baseName string) string

	// ReserveUniqueName resolves a unique name in the same way as ResolveUniqueName and reserves it
	// for inspectionID in one atomic step, so that concurrent callers never receive the same name.
	// It releases any previous reservation held by inspectionID and returns the reserved name.
	ReserveUniqueName(inspectionID string, baseName string) string

	// ReserveName atomically validates that name is non-empty and not reserved by another inspection,
	// and reserves it for inspectionID, releasing any previous reservation held by inspectionID.
	ReserveName(inspectionID string, name string) error

	// NameOf returns the name reserved for inspectionID and true, or empty string and false if no name is reserved.
	NameOf(inspectionID string) (string, bool)

	// IsNameAvailable returns true if the trimmed name is non-empty and is either not reserved or reserved by ownerID.
	// An empty ownerID treats every reservation as taken, which suits checks made before the inspection exists.
	IsNameAvailable(ownerID string, name string) bool
}

// InMemoryInspectionNameRegistry is a thread-safe in-memory implementation of InspectionNameRegistry.
type InMemoryInspectionNameRegistry struct {
	mu       sync.RWMutex
	nameByID map[string]string
	idByName map[string]string
}

// NewInMemoryInspectionNameRegistry creates a new InMemoryInspectionNameRegistry.
func NewInMemoryInspectionNameRegistry() *InMemoryInspectionNameRegistry {
	return &InMemoryInspectionNameRegistry{
		nameByID: map[string]string{},
		idByName: map[string]string{},
	}
}

// ResolveUniqueName returns baseName if it is not reserved by another inspection,
// or appends a sequential suffix like (1), (2), etc. to return an unused name.
func (r *InMemoryInspectionNameRegistry) ResolveUniqueName(inspectionID string, baseName string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.resolveUniqueNameLocked(inspectionID, baseName)
}

// ReserveUniqueName resolves a unique name from baseName and reserves it for inspectionID under a single lock.
func (r *InMemoryInspectionNameRegistry) ReserveUniqueName(inspectionID string, baseName string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	name := r.resolveUniqueNameLocked(inspectionID, baseName)
	r.reserveLocked(inspectionID, name)
	return name
}

// resolveUniqueNameLocked implements ResolveUniqueName. The caller must hold r.mu.
func (r *InMemoryInspectionNameRegistry) resolveUniqueNameLocked(inspectionID string, baseName string) string {
	trimmedBase := strings.TrimSpace(baseName)
	if trimmedBase == "" {
		trimmedBase = "Inspection"
	}

	if ownerID, exists := r.idByName[trimmedBase]; !exists || ownerID == inspectionID {
		return trimmedBase
	}
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s(%d)", trimmedBase, i)
		if ownerID, exists := r.idByName[candidate]; !exists || ownerID == inspectionID {
			return candidate
		}
	}
}

// ReserveName atomically validates that name is non-empty and not reserved by another inspection,
// and reserves it for inspectionID, releasing any previous reservation held by inspectionID.
func (r *InMemoryInspectionNameRegistry) ReserveName(inspectionID string, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ErrInspectionNameEmpty
	}

	if ownerID, exists := r.idByName[trimmed]; exists && ownerID != inspectionID {
		return ErrInspectionNameAlreadyInUse
	}

	r.reserveLocked(inspectionID, trimmed)
	return nil
}

// reserveLocked assigns name to inspectionID and releases its previous name. The caller must hold r.mu
// for writing and must have verified that name is not reserved by another inspection.
func (r *InMemoryInspectionNameRegistry) reserveLocked(inspectionID string, name string) {
	if prevName, hasPrev := r.nameByID[inspectionID]; hasPrev && prevName != name {
		delete(r.idByName, prevName)
	}
	r.nameByID[inspectionID] = name
	r.idByName[name] = inspectionID
}

// NameOf returns the name reserved for inspectionID and true, or empty string and false if no name is reserved.
func (r *InMemoryInspectionNameRegistry) NameOf(inspectionID string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	name, exists := r.nameByID[inspectionID]
	return name, exists
}

// IsNameAvailable returns true if the trimmed name is non-empty and is either not reserved or reserved by ownerID.
// An empty ownerID treats every reservation as taken.
func (r *InMemoryInspectionNameRegistry) IsNameAvailable(ownerID string, name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return false
	}
	reservedBy, exists := r.idByName[trimmed]
	return !exists || reservedBy == ownerID
}

var _ InspectionNameRegistry = (*InMemoryInspectionNameRegistry)(nil)
