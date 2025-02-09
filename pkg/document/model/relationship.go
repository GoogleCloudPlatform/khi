package model

import (
	"strings"

	"github.com/GoogleCloudPlatform/khi/pkg/model/enum"
)

type RelationshipDocumentModel struct {
	Relationships []RelationshipDocumentElement
}

type RelationshipDocumentElement struct {
	ID             string
	HasVisibleChip bool
	Label          string
	LongName       string
	ColorCode      string

	GeneratableEvents    []RelationshipGeneratableEvent
	GeneratableRevisions []RelationshipGeneratableRevisions
	GeneratableAliases   []RelationshipGeneratableAliases
}

type RelationshipGeneratableEvent struct {
	ID                 string
	SourceLogTypeLabel string
	ColorCode          string
	Description        string
}

type RelationshipGeneratableRevisions struct {
	ID                     string
	SourceLogTypeLabel     string
	SourceLogTypeColorCode string
	RevisionStateColorCode string
	RevisionStateLabel     string
	Description            string
}

type RelationshipGeneratableAliases struct {
	ID                                   string
	AliasedTimelineRelationshipLabel     string
	AliasedTimelineRelationshipColorCode string
	SourceLogTypeLabel                   string
	SourceLogTypeColorCode               string
	Description                          string
}

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
