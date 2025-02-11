{{define "inspection-type-template"}}
{{range $index,$type := .InspectionTypes }}
<!-- BEGIN GENERATED PART: inspection-type-element-header-{{$type.ID}} -->
## [{{$type.Name}}](#{{$type.ID}})

### Features

<!-- END GENERATED PART: inspection-type-element-header-{{$type.ID}} -->

<!-- BEGIN GENERATED PART: inspection-type-element-header-features-{{$type.ID}} -->
{{range $feature := $type.SupportedFeatures}}
* [{{$feature.Name}}](./features.md#{{$feature.Name | anchor }})
{{- end}}
<!-- END GENERATED PART: inspection-type-element-header-features-{{$type.ID}} -->
{{end}}
{{end}}