{{define "feature-template"}}
{{range $index,$feature := .Features }}
<!-- BEGIN GENERATED PART: feature-element-header-{{$feature.ID}} -->
## [{{$feature.Name}}](#{{$feature.ID}})

{{$feature.Description}}

<!-- END GENERATED PART: feature-element-header-{{$feature.ID}} -->
{{with $feature.Queries}}
<!-- BEGIN GENERATED PART: feature-element-depending-query-header-{{$feature.ID}} -->
### Depending Queries

Following log queries are used with this feature.
<!-- END GENERATED PART: feature-element-depending-query-header-{{$feature.ID}} -->
{{range $index,$query := $feature.Queries}}
<!-- BEGIN GENERATED PART: feature-element-depending-query-{{$feature.ID}} -->
#### ![{{$query.LogTypeColorCode}}](https://placehold.co/15x15/{{$query.LogTypeColorCode}}/{{$query.LogTypeColorCode}}.png){{$query.LogTypeLabel}}

**Sample used query**

```
{{$query.SampleQuery}}
```
<!-- END GENERATED PART: feature-element-depending-query-{{$feature.ID}} -->
{{end}}
{{end}}
{{end}}
{{end}}