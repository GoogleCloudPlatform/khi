{{define "feature-template"}}
{{range $index,$feature := .Features }}
<!-- BEGIN GENERATED PART: feature-element-header-{{$feature.ID}} -->
## {{$feature.Name}}

{{$feature.Description}}

<!-- END GENERATED PART: feature-element-header-{{$feature.ID}} -->
{{with $feature.Forms}}
<!-- BEGIN GENERATED PART: feature-element-depending-form-header-{{$feature.ID}} -->
### Parameters

|Parameter name|Description|
|:-:|---|
{{- range $index,$form := $feature.Forms}}
|[{{$form.Label}}](./forms.md#{{$form.Label | anchor}})|{{$form.Description}}|
{{- end}}
<!-- END GENERATED PART: feature-element-depending-form-header-{{$feature.ID}} -->
{{end}}

<!-- BEGIN GENERATED PART: feature-element-output-timelines-{{$feature.ID}} -->
### Output timelines

This feature can generates following timeline relationship of timelines. 

|Timeline relationships|Short name on chip|Description|
|:-:|:-:|:-:|
{{- range $index,$timeline := $feature.OutputTimelines}}
|![{{$timeline.RelationshipColorCode}}](https://placehold.co/15x15/{{$timeline.RelationshipColorCode}}/{{$timeline.RelationshipColorCode}}.png)[{{$timeline.LongName}}](./relationships.md#{{$timeline.LongName | anchor}})|{{$timeline.Label}}|{{$timeline.Description}}|
{{- end}}

<!-- END GENERATED PART: feature-element-output-timelines-{{$feature.ID}} -->
<!-- BEGIN GENERATED PART: feature-element-target-query-{{$feature.ID}} -->
### Target log type

**![{{$feature.TargetQueryDependency.LogTypeColorCode}}](https://placehold.co/15x15/{{$feature.TargetQueryDependency.LogTypeColorCode}}/{{$feature.TargetQueryDependency.LogTypeColorCode}}.png){{$feature.TargetQueryDependency.LogTypeLabel}}**

Sample query:

```ada {{/* "ada" syntax highlighting is good for Cloud Logging filter */}}
{{$feature.TargetQueryDependency.SampleQuery}}
```

<!-- END GENERATED PART: feature-element-target-query-{{$feature.ID}} -->
{{with $feature.IndirectQueryDependency}}
<!-- BEGIN GENERATED PART: feature-element-depending-indirect-query-header-{{$feature.ID}} -->
### Dependent queries

Following log queries are used with this feature.
{{range $index,$query := $feature.IndirectQueryDependency}}
* ![{{$query.LogTypeColorCode}}](https://placehold.co/15x15/{{$query.LogTypeColorCode}}/{{$query.LogTypeColorCode}}.png){{$query.LogTypeLabel}}
{{- end}}
<!-- END GENERATED PART: feature-element-depending-indirect-query-header-{{$feature.ID}} -->
{{end}}

{{with $feature.AvailableInspectionTypes}}
<!-- BEGIN GENERATED PART: feature-element-available-inspection-type-{{$feature.ID}} -->
### Inspection types

This feature is supported in the following inspection types.
{{range $index,$type := $feature.AvailableInspectionTypes}}
* [{{$type.Name}}](./inspection-type.md#{{$type.Name | anchor}})
{{- end}}
<!-- END GENERATED PART: feature-element-available-inspection-type-{{$feature.ID}} -->
{{end}}
{{end}}
{{end}}
