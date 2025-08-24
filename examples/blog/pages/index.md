---
title: "Welcome!"
---

This is the home page.

## Blog Posts

{{ $posts := readDir "posts" | sortBy "date" "desc" }}

<ul>
{{- range $posts }}
    <li><a href="{{ .Permalink }}">{{ .Metadata.title }}</a> - {{ .Metadata.date }}</li>
{{- end }}
</ul>
