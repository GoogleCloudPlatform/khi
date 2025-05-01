{{define "form-template"}}
{{range $index,$form := .Forms }}
<!-- BEGIN GENERATED PART: form-element-header-{{$form.ID}} -->
## {{$form.Label}}

{{$form.Description}}
<!-- END GENERATED PART: form-element-header-{{$form.ID}} -->

{{with $form.UsedFeatures}}
<!-- BEGIN GENERATED PART: form-used-feature-{{$form.ID}} -->
### Features using this parameter

Following feature tasks are depending on this parameter:

{{range $index,$feature := $form.UsedFeatures }}

* [{{$feature.Name}}](./features.md#{{$feature.Name | anchor}})
{{- end}}
<!-- END GENERATED PART: form-used-feature-{{$form.ID}} -->
{{end}}

{{end}}
{{end}}
