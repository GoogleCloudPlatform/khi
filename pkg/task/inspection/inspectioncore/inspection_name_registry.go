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

	// ReserveName atomically validates that name is non-empty and not reserved by another inspection,
	// and reserves it for inspectionID, releasing any previous reservation held by inspectionID.
	ReserveName(inspectionID string, name string) error
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

	if prevName, hasPrev := r.nameByID[inspectionID]; hasPrev && prevName != trimmed {
		delete(r.idByName, prevName)
	}
	r.nameByID[inspectionID] = trimmed
	r.idByName[trimmed] = inspectionID
	return nil
}

var _ InspectionNameRegistry = (*InMemoryInspectionNameRegistry)(nil)
