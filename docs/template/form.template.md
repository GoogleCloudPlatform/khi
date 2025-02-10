{{define "form-template"}}
{{range $index,$form := .Forms }}
<!-- BEGIN GENERATED PART: form-element-header-{{$form.ID}} -->
## {{$form.Label}}

{{$form.Description}}
<!-- END GENERATED PART: form-element-header-{{$form.ID}} -->
{{end}}
{{end}}