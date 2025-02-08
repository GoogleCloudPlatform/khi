{{define "inspection-type-template"}}
<!-- BEGIN GENERATED PART: inspection-type-header -->
# Inspection types

Inspection type is ...

<!-- END GENERATED PART: inspection-type-header -->
{{range $index,$type := .InspectionTypes }}
<!-- BEGIN GENERATED PART: inspection-type-element-header-{{$type.ID}} -->
## [{{$type.Name}}](#{{$type.ID}})

### Supported features

<!-- END GENERATED PART: inspection-type-element-header-{{$type.ID}} -->

{{range $feature := $type.SupportedFeatures}}
<!-- BEGIN GENERATED PART: inspection-type-element-header-{{$type.ID}}-{{$feature.ID}} -->
* {{$feature.Name}}
<!-- END GENERATED PART: inspection-type-element-header-{{$type.ID}}-{{$feature.ID}} -->
{{end}}
{{end}}
{{end}}