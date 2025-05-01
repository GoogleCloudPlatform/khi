{{define "relationship-template"}}
{{range $index,$relationship := .Relationships }}
<!-- BEGIN GENERATED PART: relationship-element-header-{{$relationship.ID}} -->
## ![#{{$relationship.ColorCode}}](https://placehold.co/15x15/{{$relationship.ColorCode}}/{{$relationship.ColorCode}}.png){{$relationship.LongName}}

{{- with $relationship.HasVisibleChip}}

Timelines of this type have ![#{{$relationship.ColorCode}}](https://placehold.co/15x15/{{$relationship.ColorCode}}/{{$relationship.ColorCode}}.png)`{{$relationship.Label}}` chip on the left side of its timeline name.
{{end}}
<!-- END GENERATED PART: relationship-element-header-{{$relationship.ID}} -->
{{with $relationship.GeneratableRevisions}}
<!-- BEGIN GENERATED PART: relationship-element-header-{{$relationship.ID}}-revisions-header -->
### Revisions

This timeline can have the following revisions.
<!-- END GENERATED PART: relationship-element-header-{{$relationship.ID}}-revisions-header -->
<!-- BEGIN GENERATED PART: relationship-element-header-{{$relationship.ID}}-revisions-table -->
|State|Source log|Description|
|---|---|---|
{{range $index,$revision := $relationship.GeneratableRevisions}}|![#{{$revision.RevisionStateColorCode}}](https://placehold.co/15x15/{{$revision.RevisionStateColorCode}}/{{$revision.RevisionStateColorCode}}.png){{$revision.RevisionStateLabel}}|![#{{$revision.SourceLogTypeColorCode}}](https://placehold.co/15x15/{{$revision.SourceLogTypeColorCode}}/{{$revision.SourceLogTypeColorCode}}.png){{$revision.SourceLogTypeLabel}}|{{$revision.Description}}|
{{end}}
<!-- END GENERATED PART: relationship-element-header-{{$relationship.ID}}-revisions-table -->
{{end}}
{{with $relationship.GeneratableEvents}}
<!-- BEGIN GENERATED PART: relationship-element-header-{{$relationship.ID}}-events-header -->
### Events

This timeline can have the following events.
<!-- END GENERATED PART: relationship-element-header-{{$relationship.ID}}-events-header -->
<!-- BEGIN GENERATED PART: relationship-element-header-{{$relationship.ID}}-events-table -->
|Source log|Description|
|---|---|
{{range $index,$event := $relationship.GeneratableEvents}}|![#{{$event.ColorCode}}](https://placehold.co/15x15/{{$event.ColorCode}}/{{$event.ColorCode}}.png){{$event.SourceLogTypeLabel}}|{{$event.Description}}|
{{end}}
<!-- END GENERATED PART: relationship-element-header-{{$relationship.ID}}-events-table -->
{{end}}
{{with $relationship.GeneratableAliases}}
<!-- BEGIN GENERATED PART: relationship-element-header-{{$relationship.ID}}-aliases-header -->
### Aliases

This timeline can have the following aliases.
<!-- END GENERATED PART: relationship-element-header-{{$relationship.ID}}-aliases-header -->
<!-- BEGIN GENERATED PART: relationship-element-header-{{$relationship.ID}}-aliases-table -->
|Aliased timeline|Source log|Description|
|---|---|---|
{{range $index,$alias := $relationship.GeneratableAliases}}|![#{{$alias.AliasedTimelineRelationshipColorCode}}](https://placehold.co/15x15/{{$alias.AliasedTimelineRelationshipColorCode}}/{{$alias.AliasedTimelineRelationshipColorCode}}.png){{$alias.AliasedTimelineRelationshipLabel}}|![#{{$alias.SourceLogTypeColorCode}}](https://placehold.co/15x15/{{$alias.SourceLogTypeColorCode}}/{{$alias.SourceLogTypeColorCode}}.png){{$alias.SourceLogTypeLabel}}|{{$alias.Description}}|
{{end}}
<!-- END GENERATED PART: relationship-element-header-{{$relationship.ID}}-aliases-table -->
{{end}}

{{end}}
{{end}}
