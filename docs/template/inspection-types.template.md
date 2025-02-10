{{define "inspection-type-template"}}
{{range $index,$type := .InspectionTypes }}
<!-- BEGIN GENERATED PART: inspection-type-element-header-{{$type.ID}} -->
## [{{$type.Name}}](#{{$type.ID}})

### Features

<!-- END GENERATED PART: inspection-type-element-header-{{$type.ID}} -->

{{range $feature := $type.SupportedFeatures}}
<!-- BEGIN GENERATED PART: inspection-type-element-header-{{$type.ID}}-{{$feature.ID}} -->
* [{{$feature.Name}}](./features.md#{{$feature.ID}})
<!-- END GENERATED PART: inspection-type-element-header-{{$type.ID}}-{{$feature.ID}} -->
{{end}}
{{end}}
{{end}}