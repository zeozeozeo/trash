---
title: "Parsing CSV with Expr"
description: "Parse CSV files directly in a template"
---

This example parses the [products.csv](/products.csv) file directly in the template by using the [Expr](https://expr-lang.org/) expression language.

<!-- parse the CSV file -->

{{ $csvData := readFile "static/products.csv" }}
{{ $products := expr `
    let lines = split(csvData, "\n");
    let header = map(split(trim(lines[0]), ","), { trim(#) });
    let data_lines = filter(lines[1:], { trim(#) != "" });
    map(data_lines, {
        let fields = split(trim(#), ",");
        {
            (header[0]): fields[0],
            (header[1]): int(fields[1]),
            (header[2]): fields[2],
            (header[3]): int(fields[3])
        }
    })
` (dict "csvData" $csvData) }}

<!-- make it a table -->

| Name | Price | Category | Stock |
| ---- | ----- | -------- | ----- |

{{- range $products }}
| {{ .name }} | {{ .price }} | {{ .category }} | {{ .stock }} |
{{- end }}

## Filtered products: electronics only

This section only displays products with the category set to "Electronics".

<ul>
{{- $electronics := expr `filter(products, {.category == "Electronics"})` (dict "products" $products) }}
{{- range $electronics }}
    <li>{{ .name }} - In stock: {{ .stock }}</li>
{{- end }}
</ul>
