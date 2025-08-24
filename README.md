# Trash

Trash - a stupid, simple website compiler.

> [!CAUTION]
> The demos are quite ugly and unfinished, but the compiler itself is already pretty usable.

## Features

- $LaTeX$ expressions (no client-side JS!)
- [D2](https://d2lang.com/) diagram rendering (no JS still!)
- [Mermaid](https://mermaid.js.org/) diagram rendering (yeah, still no client-side JS)
- [Pikchr](https://pikchr.org/home/doc/trunk/homepage.md) diagram rendering (you guessed it)
- Painless embedding of YouTube videos, HTML5 audio, and more in native Markdown
- Syntax highlighting
- Various Markdown extensions such as image `<figure>`s, image sizing, callouts, Pandoc-style fences, `:emojis:`, and more
- YAML `---` and TOML `+++` frontmatter parsing support
- Automatic anchor placement
- Under 700 lines of Go code [in a single file](./main.go)

## Installation

Install [Go](https://go.dev/dl/) if you haven't yet.

```console
$ go install github.com/zeozeozeo/trash@latest
```

## Usage

```console
$ trash help
Usage: trash <command> [directory]

A stupid, simple website compiler.

Commands:
  init     Initialize a new site in the directory (default: current).
  build    Build the site.
  watch    Watch for changes and rebuild.
  serve    Serve the site with live reload.
  help     Show this help message.
```
