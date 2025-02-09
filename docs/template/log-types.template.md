{{define "log-type-template"}}
{{range $index,$type := .LogTypes }}
<!-- BEGIN GENERATED PART: log-type-element-header-{{$type.ID}} -->
## [![#{{$type.ColorCode}}](https://placehold.co/15x15/{{$type.ColorCode}}/{{$type.ColorCode}}.png) {{$type.Name}}](#{{$type.ID}})
<!-- END GENERATED PART: log-type-element-header-{{$type.ID}} -->
{{end}}
{{end}}