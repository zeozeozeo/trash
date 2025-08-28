---
title: "Source Code"
description: "The entire source code of Trash"
date: "2025-08-22"
---

{{ $sourcePath := "../../main.go" }} <!-- relative to examples/blog -->

{{ if pathExists $sourcePath }}

```go
{{ readFile $sourcePath }}
```

{{ else }}
Couldn't read file `{{ $sourcePath }}`, make sure to clone the entire project
{{ end }}
