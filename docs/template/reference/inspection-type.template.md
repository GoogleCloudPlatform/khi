{{define "inspection-type-template"}}
{{range $index,$type := .InspectionTypes }}
<!-- BEGIN GENERATED PART: inspection-type-element-header-{{$type.ID}} -->
## [{{$type.Name}}](#{{$type.ID}})

<!-- END GENERATED PART: inspection-type-element-header-{{$type.ID}} -->

{{with $type.SupportedFeatures}}
<!-- BEGIN GENERATED PART: inspection-type-element-header-features-{{$type.ID}} -->
### Features

| Feature task name | Description |
| --- | --- |
{{- range $feature := $type.SupportedFeatures}}
|[{{$feature.Name}}](./features.md#{{$feature.Name | anchor }})|{{$feature.Description}}|
{{- end}}
<!-- END GENERATED PART: inspection-type-element-header-features-{{$type.ID}} -->
{{end}}
{{end}}
{{end}}
