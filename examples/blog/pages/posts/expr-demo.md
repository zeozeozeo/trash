---
title: "Parsing CSV with Expr"
description: "Parse CSV files directly in a template"
date: "2025-08-25"
---

This example parses the [products.csv](static/products.csv) file directly in the template by using the [Expr](https://expr-lang.org/) expression language.

<!-- parse the CSV file -->

{{ $csvData := readFile "static/products.csv" }}
{{ $products := expr `
    let lines = split(csvData, "\n");
    let header = map(split(trim(lines[0]), ","), { trim(#) });
    let dataLines = filter(lines[1:], { trim(#) != "" });

    map(dataLines, {
        let fields = split(trim(#), ",");
        let pairs = map(header, {
            let value = fields[#index];
            let isInt = value matches '^-?[0-9]+$';
            let isFloat = value matches '^-?[0-9]+\\.[0-9]+$';
            [#, isInt ? int(value) : (isFloat ? float(value) : value)]
        });
        fromPairs(pairs)
    })
` (dict "csvData" $csvData) }}

<!-- make it a table -->

| Name | Price | Category | Stock |
| ---- | ----- | -------- | ----- |

{{- range $products }}
| {{ .name }} | {{ .price }} | {{ .category }} | {{ .stock }} |
{{- end }}

```rs
// the above is parsed with
let lines = split(csvData, "\n");
let header = map(split(trim(lines[0]), ","), { trim(#) });
let dataLines = filter(lines[1:], { trim(#) != "" });
map(dataLines, {
    let fields = split(trim(#), ",");
    let pairs = map(header, {
        let value = fields[#index];
        let isInt = value matches '^-?[0-9]+$';
        let isFloat = value matches '^-?[0-9]+\\.[0-9]+$';
        [#, isInt ? int(value) : (isFloat ? float(value) : value)]
    });
    fromPairs(pairs)
})
```

See the [Expr documentation](https://expr-lang.org/docs/getting-started) and [language definition](https://expr-lang.org/docs/language-definition) for more about the language.

## Filtered products: electronics only

This section only displays products with the category set to "Electronics".

<ul>
{{- $electronics := expr `filter(products, {.category == "Electronics"})` (dict "products" $products) }}
{{- range $electronics }}
    <li>{{ .name }} - In stock: {{ .stock }}</li>
{{- end }}
</ul>
