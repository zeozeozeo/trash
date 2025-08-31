# Trash

Trash - a stupid, simple website compiler.

This readme is more of a reference; if you want pretty pictures, go see the ones in [examples](/examples/) (or see [hosted version](https://zeozeozeo.github.io/trash/)).

## Features

- $LaTeX$ expressions (no client-side JS!)
- [D2](https://d2lang.com/) diagram rendering (no JS still!)
- [Mermaid](https://mermaid.js.org/) diagram rendering (yeah, still no client-side JS)
- [Pikchr](https://pikchr.org/home/doc/trunk/homepage.md) diagram rendering (you guessed it)
- Painless embedding of YouTube videos, HTML5 audio, and more in native Markdown
- Syntax highlighting
- Various Markdown extensions such as image `<figure>`s, image sizing, callouts, Pandoc-style fences, `:emojis:`, ==text highlighting==, and more
- YAML and TOML frontmatter parsing support
- Automatic table of contents (TOC) generation and anchor placement
- Automatially minifies output HTML, CSS, JS, JSON, SVG and XML for smallest builds
- Lots of built-in template functions including an integration with the [Expr expression language](https://expr-lang.org/)
- Built-in webserver with live-reloading (`trash serve`)
- Under 1600 lines of Go code [in a single file](./main.go)

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

## Template cheatsheet

Trash uses the [Go template syntax](https://pkg.go.dev/text/template) for templates, extending it with some handy built-in functions. Below is a reference of all extra functions defined by Trash.

You should still refer to the [source code](./main.go) instead of this if possible, this mostly serves as a general overview.

#### File system

- `readDir "path"`: Get all pages within a directory
  ```go-template
  {{ $posts := readDir "posts" }}
  ```
- `listDir "path"`: List directory entries, use `.Name` and `.IsDir` on returned values

  used in [personal example](./examples/personal/layouts/base.html)

- `readFile "path"`: Read a file from the project root
  ```go-template
  <style>{{ readFile "static/style.css" }}</style>
  ```
- `pathExists "path"`: Return true if a file, directory or symlink exists under `path`

#### Dict operations

- `dict "key1" val1 "key2" val2`: Create a dict, usually paired with other functions
- `sortBy "key" "order" $dict`: Sort by a key path. `order` can be `"asc"` or `"desc"`
  ```go-template
  {{ $posts := readDir "posts" | sortBy "date" "desc" }}
  {{ $users := $data.users | sortBy "age" "asc" }}
  ```
- `where "key" value $dict`: Filter where a key path matches a value
  ```go-template
  {{ $featured := where "featured" true .Site.Pages }}
  {{ $activeUsers := where "active" true $data.users }}
  ```
- `groupBy "key" $dict`: Group by a key path
  ```go-template
  {{ $postsByYear := groupBy "date.year" .Site.Pages }}
  {{ $usersByDept := groupBy "department" $data.users }}
  ```
- `select "key" $dict`: Extract values from a key path across many dicts
  ```go-template
  {{ $allTags := select "tags" .Site.Pages }}
  {{ $allNames := select "name" $data.users }}
  ```
- `has "key" $dict`: Check if a dict has a certain key path
  ```go-template
  {{ if has "image" .Page }} ... {{ end }}
  {{ if has "email" $user }} ... {{ end }}
  ```

All collection functions support dot notation for nested keys:

```go-template
{{ where "author.name" "Alice" .Site.Pages }}
{{ groupBy "metadata.tags.primary" .Site.Pages }}
{{ select "contact.email.work" $users }}
```

Passing a `.Page` will decay into its frontmatter (`.Page.Metadata`):

```go-template
{{/* These are equivalent - both access the page's frontmatter */}}
{{ if has "title" .Page.Metadata }} ... {{ end }}
{{ if has "title" .Page }} ... {{ end }}
```

#### Datetime

- `now`: Return the current UTC time
- `formatTime "format" date`: Format time

  ```go-template
  {{ .Metadata.date | formatTime "DateOnly" }}
  ```

  Supported formats:

  | Format        | Output (example)                    |
  | ------------- | ----------------------------------- |
  | `Layout`      | 01/02 03:04:05PM '06 -0700          |
  | `ANSIC`       | Mon Jan \_2 15:04:05 2006           |
  | `UnixDate`    | Mon Jan \_2 15:04:05 MST 2006       |
  | `RubyDate`    | Mon Jan 02 15:04:05 -0700 2006      |
  | `RFC822`      | 02 Jan 06 15:04 MST                 |
  | `RFC822Z`     | 02 Jan 06 15:04 -0700               |
  | `RFC850`      | Monday, 02-Jan-06 15:04:05 MST      |
  | `RFC1123`     | Mon, 02 Jan 2006 15:04:05 MST       |
  | `RFC1123Z`    | Mon, 02 Jan 2006 15:04:05 -0700     |
  | `RFC3339`     | 2006-01-02T15:04:05Z07:00           |
  | `RFC3339Nano` | 2006-01-02T15:04:05.999999999Z07:00 |
  | `Kitchen`     | 3:04PM                              |
  | `Stamp`       | Jan \_2 15:04:05                    |
  | `StampMilli`  | Jan \_2 15:04:05.000                |
  | `StampMicro`  | Jan \_2 15:04:05.000000             |
  | `StampNano`   | Jan \_2 15:04:05.000000000          |
  | `DateTime`    | 2006-01-02 15:04:05                 |
  | `DateOnly`    | 2006-01-02                          |
  | `TimeOnly`    | 15:04:05                            |

  Or vice-versa (passing `"15:04:05"` will have the same effect as passing `"TimeOnly"`)

Time is always returned and formatted in the UTC timezone, no matter what your local timezone is.

#### Strings and URLs

- `concatURL "base" "path1" "path2" ...`: Join URL parts together
  ```go-template
  <img src="{{ concatURL .Config.site.url .Page.Metadata.image }}">
  ```
- `truncate length "string"`: Shorten a string to a max length by adding `…`
  ```go-template
  <p>{{ .Content | truncate 150 }}</p>
  ```
- `pluralize count "singular" "plural"`: Return the singular or plural form based on the count
  ```go-template
  {{ len $posts | pluralize "post" "posts" }}
  ```
- `markdownify "string"`: Render a string of Markdown as HTML
  ```go-template
  {{ .Page.Metadata.bio | markdownify }}
  ```
- `replace "old" "new" str`: Replace every occurence of `old` with `new` in string `str`
- `contains "substring str"`: Check if a string contains a substring
- `startsWith "prefix str"`: Check if a string contains a prefix
- `endsWith "suffix str"`: Check if a string contains a suffix
- `repeat count str`: Repeat the string `count` times
- `toUpper "string"`: Make a string uppercase
- `toLower "string"`: Make a string lowercase
- `title "string"`: Make all words start with a capital letter, e.g. `title "hello world"` -> `"Hello World"`
- `strip "string"`: Remove all leading and trailing whitespace in a string
- `split "sep" str`: Slice `str` into all substrings separated by `sep`
- `fields "string"`: Split `str` around each instance of one or more consecutive whitespace characters
- `count "substr" str`: Return the number of non-overlapping instances of `substr` in `str`

#### Conditionals

- `default "fallback" value`: Return the fallback if the value is empty
  ```go-template
  <img alt="{{ default "A cool image" .Page.Metadata.altText }}">
  ```
- `ternary condition trueVal falseVal`: if/else
  ```go-template
  <body class="{{ ternary (.Page.Metadata.isHome) "home" "page" }}">
  ```

#### Math and random

- `add`, `subtract`, `multiply`, `divide`, `max`, `min`: Operations on integers and floats
- `rand min max`: A random integer or float in range
- `choice item1 item2 ...`: Randomly select one item from the list of arguments provided
  ```go-template
  <p>Today's greeting: {{ choice "Hello" "Welcome" "Greetings" "Howdy" }}!</p>
  ```
- `shuffle $slice`: Randomly shuffle a slice (returns a copy)

#### Slice utilities

- `first $slice`: Get the first item of a slice
- `last $slice`: Get the last item of a slice
- `reverse $slice`: Return a new slice with the order of elements reversed
- `contains $slice item`: Check if a slice contains an item
  ```go-template
  {{ if contains .Page.Metadata.tags "featured" }}
    <span class="featured-badge">Featured</span>
  {{ end }}
  ```

#### Casts

- `toString value`: Convert any value to a string
- `toInt value`: Convert any value (e.g. float, string) to an integer
- `toFloat value`: Convert any value (e.g. int, string) to a float

#### Debugging

- `print value`: Print a value during the build

#### Utility

- `toc`: Render the automatically generated table of contents of the current document
- `toJSON $data`: Convert a value to a JSON string
- `fromJSON $data`: Parse a JSON string
- `sprint "format" values...`: Return a formatted string, similar to `printf`
  ```go-template
  {{ $message := sprint "Processed page %s" .Page.Permalink }}
  ```
- `expr "code" environ`: Execute an [Expr](https://expr-lang.org/) block, see the [blog example](./examples/blog/pages/posts/expr-demo.md) for more

## Config format

Running `trash init` will create a `Trash.toml` as one of the files in your current directory. The structure of this file is not forced upon you whatsoever, however there are a few optional settings you can toggle:

```toml
[mermaid]
theme = "dark" # see available themes at https://mermaid.js.org/config/theming.html

[d2]
sketch = true # see https://d2lang.com/tour/sketch/
theme = 200   # see available themes at https://d2lang.com/tour/themes/

[pikchr]
dark = true # change pikchr colors to assume a dark-mode theme

[anchor]
text = "#"          # change the ¶ character in auto-anchors to something else
position = "before" # default is "after", where to place the anchor

[highlight]
enabled = true        # whether to enable code highlighting (default: true)
prefix = "highlight-" # CSS class prefix to use, this is the default

[highlight.gutter]
enabled = true # whether to show line numbers (default: false)
table = true   # whether to separate code and line numbers using a <td>, copy-paste friendly (default: false)
```

Aside from this, you can add your own fields, and access them in templates:

```toml
[site]
url = "https://example.com/"
```

Let's say you're making an RSS feed for your blog:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
<channel>
    <title>My Blog</title>
    <link>{{ .Config.site.url }}</link> <!-- this is the `url` from Trash.toml! -->
    ...
    <item>
        ...
    </item>
</channel>
```

## Template context

All templates under the `layouts` directory are created in the same context, so you can include other templates within a template:

```html
<!-- layouts/base.html (this is the default layout) --->
<!DOCTYPE html>
<html lang="en">
  {{ template "boilerplate.html" }}
  <body>
    {{ .Page.Content }}
  </body>
</html>
```

```html
<!-- layouts/boilerplate.html --->
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>My Blog</title>
</head>
```

This is not the case for `.md` filles inside the `pages` directory.

## Hosting on GitHub Pages

Since Trash generates a static website, you can use [GitHub Pages](https://docs.github.com/en/pages) to host it for free directly from your GitHub repo. Here's a tutorial:

1. Create a new repository
2. Go to <kbd>Settings</kbd> > <kbd>Pages</kbd> (in sidebar) > <kbd>Source</kbd> > select "GitHub Actions" in the dropdown (change from "Deploy from a branch")
3. Create `.github/workflows/build.yml`:

   ```yml
   name: Build and Deploy Page

   on:
     push:
       branches:
         - main # change to the branch you want to deploy from
       # you can use `paths:` to only run on changes in set path
     workflow_dispatch:
     pull_request:
       branches:
         - main # change to the branch you want to deploy from

   permissions:
     contents: read
     pages: write
     id-token: write

   concurrency:
     group: ${{ github.workflow }}-${{ github.ref }}
     cancel-in-progress: true

   jobs:
     build:
       runs-on: ubuntu-latest
       steps:
         - name: Checkout
           uses: actions/checkout@v4

         - name: Setup Go
           uses: actions/setup-go@v4
           with:
             go-version: "^1.25.0"

         - name: Setup Trash
           run: go install github.com/zeozeozeo/trash@latest

         - name: Build Site
           run: |
             TRASH_NO_SANDBOX=1 ./trash build .

         - name: Setup Pages
           uses: actions/configure-pages@v4

         - name: Upload artifact
           uses: actions/upload-pages-artifact@v4
           with:
             path: ./out

     deploy:
       if: github.ref == 'refs/heads/main' # change "main" to the branch you want to deploy from
       environment:
         name: github-pages
         url: ${{ steps.deployment.outputs.page_url }}
       runs-on: ubuntu-latest
       needs: build
       steps:
         - name: Deploy to GitHub Pages
           id: deployment
           uses: actions/deploy-pages@v4
   ```

4. After the `deploy` step finishes, the page should be live at `https://<your_github_username>.github.io/<your_repo_name>`

> [!NOTE]
> If you are using Mermaid diagrams, CI can be a little bit flaky. If you are getting the following errors when building the site:
>
> ```
> error: Failed to initialize Mermaid with CDP: set up headless browser: websocket url timeout reached; falling back to clientside JS
> error: Failed to process page pages/posts/trash-demo.md: failed to convert markdown: generate svg: mmdc: exec: "mmdc": executable file not found in $PATH
> ```
>
> that means that it hasn't fully compiled and some pages will be missing. In that case, just re-run the build (for some reason Chrome is particularly slow in the runners). If that doesn't fix it, you can add a build step to install Node.js and `npm install -g @mermaid-js/mermaid-cli` to install `mmdc`. It shouldn't fail, but CDP is still preferred since `mmdc` generates SVGs with an opaque background.

By deploying the site directly from GitHub Actions, we get rid of the need to host the `out` directory in the repository.
