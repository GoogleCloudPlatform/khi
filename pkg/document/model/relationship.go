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

package model

import (
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

// RelationshipDocumentModel is a model type for generating document docs/en/reference/relationships.md.
type RelationshipDocumentModel struct {
	// Relationships is a list of relationship document elements.
	Relationships []RelationshipDocumentElement
}

// RelationshipDocumentElement represents a relationship element in the document.
type RelationshipDocumentElement struct {
	// ID is the unique identifier of the relationship.
	ID string
	// HasVisibleChip indicates whether the relationship has a visible chip on the left side of timeline name.
	HasVisibleChip bool
	// Label is the short label for the relationship.
	Label string
	// LongName is the descriptive name of the relationship.
	LongName string
	// ColorCode is the hexadecimal color code for the relationship.
	ColorCode string

	// GeneratableEvents is the list of the generatable events on the timeline of this relationship.
	GeneratableEvents []RelationshipGeneratableEvent
	// GeneratableRevisions is the list of the generatable revisions on the timeline of this relationship.
	GeneratableRevisions []RelationshipGeneratableRevisions
	// GeneratableAliases is the list of the generatable aliases on the timeline of this relationship.
	GeneratableAliases []RelationshipGeneratableAliases
}

// RelationshipGeneratableEvent represents a generatable event on the timeline of this relationship.
type RelationshipGeneratableEvent struct {
	// ID is the unique identifier of the event.
	ID string
	// SourceLogTypeLabel is the label of the source log type.
	SourceLogTypeLabel string
	// ColorCode is the hexadecimal color code for the event without `#` prefix.
	ColorCode string
	// Description describes the event.
	Description string
}

// RelationshipGeneratableRevisions represents generatable revision states on the timeline of this relationship.
type RelationshipGeneratableRevisions struct {
	// ID is the unique identifier of the revision state.
	ID string
	// SourceLogTypeLabel is the label of the source log type.
	SourceLogTypeLabel string
	// SourceLogTypeColorCode is the hexadecimal color code for the source log type without `#` prefix.
	SourceLogTypeColorCode string
	// RevisionStateColorCode is the hexadecimal color code for the revision state without `#` prefix.
	RevisionStateColorCode string
	// RevisionStateLabel is the label of the revision state.
	RevisionStateLabel string
	// Description describes the revision state.
	Description string
}

// RelationshipGeneratableAliases represents generatable aliases on the timeline of this relationship.
type RelationshipGeneratableAliases struct {
	// ID is the unique identifier of the alias.
	ID string
	// AliasedTimelineRelationshipLabel is the label of the aliased timeline relationship.
	AliasedTimelineRelationshipLabel string
	// AliasedTimelineRelationshipColorCode is the hexadecimal color code for the aliased timeline relationship.
	AliasedTimelineRelationshipColorCode string
	// SourceLogTypeLabel is the label of the source log type.
	SourceLogTypeLabel string
	// SourceLogTypeColorCode is the hexadecimal color code for the source log type  without `#` prefix.
	SourceLogTypeColorCode string
	// Description describes the alias.
	Description string
}

// GetRelationshipDocumentModel returns the document model for relationships.
func GetRelationshipDocumentModel() RelationshipDocumentModel {
	relationships := []RelationshipDocumentElement{}
	for i := 0; i < int(enum.EnumParentRelationshipLength); i++ {
		relationshipKey := enum.ParentRelationship(i)
		relationship := enum.ParentRelationships[relationshipKey]
		relationships = append(relationships, RelationshipDocumentElement{
			ID:             relationship.EnumKeyName,
			HasVisibleChip: relationship.Visible,
			Label:          relationship.Label,
			LongName:       relationship.LongName,
			ColorCode:      strings.TrimLeft(relationship.LabelBackgroundColor, "#"),

			GeneratableEvents:    getRelationshipGeneratableEvents(relationshipKey),
			GeneratableRevisions: getRelationshipGeneratableRevisions(relationshipKey),
			GeneratableAliases:   getRelationshipGeneratableAliases(relationshipKey),
		})
	}

	return RelationshipDocumentModel{
		Relationships: relationships,
	}
}

// getRelationshipGeneratableEvents retrieves generatable events for a given relationship.
func getRelationshipGeneratableEvents(reltionship enum.ParentRelationship) []RelationshipGeneratableEvent {
	result := []RelationshipGeneratableEvent{}
	relationship := enum.ParentRelationships[reltionship]
	for _, event := range relationship.GeneratableEvents {
		logType := enum.LogTypes[event.SourceLogType]
		result = append(result, RelationshipGeneratableEvent{
			ID:                 logType.EnumKeyName,
			SourceLogTypeLabel: logType.Label,
			ColorCode:          strings.TrimLeft(logType.LabelBackgroundColor, "#"),
			Description:        event.Description,
		})
	}
	return result
}

// getRelationshipGeneratableRevisions retrieves generatable revisions for a given relationship.
func getRelationshipGeneratableRevisions(reltionship enum.ParentRelationship) []RelationshipGeneratableRevisions {
	result := []RelationshipGeneratableRevisions{}
	relationship := enum.ParentRelationships[reltionship]
	for _, revision := range relationship.GeneratableRevisions {
		logType := enum.LogTypes[revision.SourceLogType]
		revisionState := enum.RevisionStates[revision.State]
		result = append(result, RelationshipGeneratableRevisions{
			ID:                     logType.EnumKeyName,
			SourceLogTypeLabel:     logType.Label,
			SourceLogTypeColorCode: strings.TrimLeft(logType.LabelBackgroundColor, "#"),
			RevisionStateColorCode: strings.TrimLeft(revisionState.BackgroundColor, "#"),
			RevisionStateLabel:     revisionState.Label,
			Description:            revision.Description,
		})
	}
	return result
}

// getRelationshipGeneratableAliases retrieves generatable aliases for a given relationship.
func getRelationshipGeneratableAliases(reltionship enum.ParentRelationship) []RelationshipGeneratableAliases {
	result := []RelationshipGeneratableAliases{}
	relationship := enum.ParentRelationships[reltionship]
	for _, alias := range relationship.GeneratableAliasTimelineInfo {
		aliasedRelationship := enum.ParentRelationships[alias.AliasedTimelineRelationship]
		logType := enum.LogTypes[alias.SourceLogType]
		result = append(result, RelationshipGeneratableAliases{
			ID:                                   logType.EnumKeyName,
			AliasedTimelineRelationshipLabel:     aliasedRelationship.Label,
			AliasedTimelineRelationshipColorCode: strings.TrimLeft(aliasedRelationship.LabelBackgroundColor, "#"),
			SourceLogTypeLabel:                   logType.Label,
			SourceLogTypeColorCode:               strings.TrimLeft(logType.LabelBackgroundColor, "#"),
			Description:                          alias.Description,
		})
	}
	return result
}
