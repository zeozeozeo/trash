// Trash - a stupid, simple website compiler.
// Licensing information at the bottom of this file.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"maps"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	text_template "text/template"
	"time"

	_ "embed"

	d2 "github.com/FurqanSoftware/goldmark-d2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/davecgh/go-spew/spew"
	"github.com/expr-lang/expr"
	"github.com/fatih/color"
	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
	pikchr "github.com/jchenry/goldmark-pikchr"
	figure "github.com/mangoumbrella/goldmark-figure"
	"github.com/pelletier/go-toml/v2"
	enclave "github.com/quailyquaily/goldmark-enclave"
	enclaveCallout "github.com/quailyquaily/goldmark-enclave/callout"
	enclaveCore "github.com/quailyquaily/goldmark-enclave/core"
	fences "github.com/stefanfritsch/goldmark-fences"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	minifyHtml "github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	minify_json "github.com/tdewolff/minify/v2/json"
	"github.com/tdewolff/minify/v2/svg"
	"github.com/tdewolff/minify/v2/xml"
	treeblood "github.com/wyatt915/goldmark-treeblood"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/goldmark/anchor"
	"go.abhg.dev/goldmark/frontmatter"
	"go.abhg.dev/goldmark/mermaid"
	"go.abhg.dev/goldmark/mermaid/mermaidcdp"
)

var (
	//go:embed mermaid.min.js
	mermaidJSSource string

	mermaidCompiler *mermaidcdp.Compiler
)

func usage() {
	programName := getProgramName()
	fmt.Printf("Usage: %s <command> [directory]\n\n", programName)
	fmt.Println("A stupid, simple website compiler.")
	fmt.Println("\nCommands:")
	fmt.Println("  init     Initialize a new site in the directory (default: current).")
	fmt.Println("  build    Build the site.")
	fmt.Println("  watch    Watch for changes and rebuild.")
	fmt.Println("  serve    Serve the site with live reload.")
	fmt.Println("  help     Show this help message.")
}

func printerr(format string, a ...any) {
	fmt.Print(color.HiRedString("error"))
	fmt.Print(": ")
	fmt.Printf(format, a...)
	fmt.Print("\n")
}

func printwarn(format string, a ...any) {
	fmt.Print(color.YellowString("warn"))
	fmt.Print(": ")
	fmt.Printf(format, a...)
	fmt.Print("\n")
}

func main() {
	if len(os.Args) == 2 &&
		(os.Args[1] == "help" ||
			os.Args[1] == "-h" ||
			os.Args[1] == "--help" ||
			os.Args[1] == "?") {
		usage()
		return
	}

	defer func() {
		if mermaidCompiler != nil {
			mermaidCompiler.Close()
		}
	}()

	var cmd string
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	switch cmd {
	case "", "build":
		build(false, true)
	case "init":
		initCmd()
	case "watch":
		watchCmd()
	case "serve":
		serveCmd()
	default:
		printerr("No such command `%s`.\n", cmd)
		usage()
		os.Exit(1)
	}
}

// -- util --

func checkAllDirsExist(dirs ...string) bool {
	for _, dir := range dirs {
		if stat, err := os.Stat(dir); err != nil || !stat.IsDir() {
			return false
		}
	}
	return true
}

func isEmptyDir() bool {
	entries, err := os.ReadDir(".")
	if err != nil {
		return false
	}
	return len(entries) == 0
}

func ask(prompt string) bool {
	for {
		fmt.Print(prompt)
		var resp string
		_, err := fmt.Scanln(&resp)
		if err != nil {
			printerr("%v", err)
			os.Exit(1)
		}
		switch strings.TrimSpace(resp) {
		case "", "y", "Y":
			return true
		case "n", "N":
			return false
		default:
			fmt.Println("enter y or n.")
		}
	}
}

func getProgramName() string {
	basename := filepath.Base(os.Args[0])
	return strings.TrimSuffix(basename, filepath.Ext(basename))
}

func writefile(content string, elem ...string) {
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	path := filepath.Join(elem...)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		_ = os.WriteFile(path, []byte(content), 0o644)
		fmt.Printf("%s file: %s\n", color.HiGreenString("Created"), filepath.ToSlash(path))
	}
}

func makedirs(dirs ...string) {
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			printerr("Mkdir %s: %v", d, err)
			os.Exit(1)
		}
		fmt.Printf("%s directory: %s\n", color.HiGreenString("Created"), d)
	}
}

func queryMap(m map[string]any, path ...string) (any, bool) {
	for i, key := range path {
		if m == nil {
			return nil, false
		}
		if i == len(path)-1 {
			val, exists := m[key]
			return val, exists
		}
		if next, ok := m[key]; ok {
			if nextMap, ok := next.(map[string]any); ok {
				m = nextMap
			} else {
				return nil, false
			}
		} else {
			return nil, false
		}
	}
	return nil, false
}

func queryMapOrDefault[T any](m map[string]any, fallback T, path ...string) T {
	val, exists := queryMap(m, path...)
	if !exists {
		return fallback
	}
	if converted, ok := val.(T); ok {
		return converted
	}
	return fallback
}

// -- init --

const (
	pagesDir            = "pages"
	staticDir           = "static"
	layoutsDir          = "layouts"
	outputDir           = "out"
	trashConfigFilename = "Trash.toml"
)

func initCmd() {
	if !isEmptyDir() {
		if !ask("The current directory is not empty. Are you sure you want to continue? (Y/n): ") {
			fmt.Println("Aborting.")
			return
		}
	}

	makedirs(pagesDir+"/posts", staticDir, layoutsDir)

	// layouts/base.html
	writefile(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{ .Page.Metadata.title }}</title>
    <link rel="stylesheet" href="/style.css">
</head>
<body>
    <div class="container">
        <h1>{{ .Page.Metadata.title }}</h1>
        <main>
            {{ .Page.Content }}
        </main>
    </div>

    {{ if .IsServing }}
    <script>
        const socket = new WebSocket("ws://" + window.location.host + "/ws");
        socket.addEventListener("message", (event) => {
            if (event.data === "reload") {
                window.location.reload();
            }
        });
        socket.addEventListener("close", () => {
            console.log("Live reload connection lost. Please refresh manually.");
        });
    </script>
    {{ end }}
</body>
</html>`, layoutsDir, "base.html")

	// static/style.css
	writefile(`body {
  font-family: sans-serif;
  color: #333;
}

.container {
  max-width: 800px;
  margin: 2rem auto;
  padding: 0 1rem;
}`, staticDir, "style.css")

	// pages/posts/first-post.md
	writefile(`---
title: "Welcome!"
---

This is the home page.

## Blog Posts

{{ $posts := readDir "posts" | sortBy "date" "desc" }}

<ul>
{{- range $posts }}
    <li><a href="{{ .Permalink }}">{{ .Metadata.title }}</a> - {{ .Metadata.date }}</li>
{{- end }}
</ul>`, pagesDir, "index.md")

	writefile(`---
title: "My First Post"
date: "2025-08-24"
---

Hello, world! This is my first post.`, pagesDir, "posts", "first-post.md")

	// .gitignore
	writefile("/out", ".gitignore")

	// Trash.toml
	writefile(`# The structure of this config file is not forced upon you, it is just useful to
# have the permalink/other configuration stored somewhere so you can access it in templates

[site]
url = "https://example.com/" # Access this like {{ .Config.site.url }}`, trashConfigFilename)

	programName := getProgramName()
	fmt.Printf("You can now do %s to build your site, %s to rebuild on file changes, or %s to start a server with live reloading.\n", color.HiBlueString(programName), color.HiBlueString(programName+" watch"), color.HiBlueString(programName+" serve"))
}

// -- build --

type Page struct {
	// SourcePath is the path to the original .md file relative to the project root.
	SourcePath string
	// IsMarkdown is true when the file is a markdown file.
	IsMarkdown bool
	// Permalink is the final URL path for the page.
	Permalink string
	// RawContent is the raw file content.
	RawContent string
	// Content is the final HTML content after all processing.
	Content template.HTML
	// Metadata is the parsed YAML/TOML front matter.
	Metadata map[string]any
}

type Site struct {
	Pages []*Page
}

type TemplateData struct {
	Site      Site
	Page      *Page
	Config    map[string]any
	IsServing bool
}

func initMermaidCDP() {
	var err error
	mermaidCompiler, err = mermaidcdp.New(&mermaidcdp.Config{
		JSSource: mermaidJSSource,
	})
	if err != nil {
		printerr("Failed to initialize Mermaid with CDP: %v; falling back to clientside JS", err)
	}
}

func newMinifier() *minify.M {
	m := minify.New()
	m.AddFunc("text/css", css.Minify)
	m.AddFunc("text/html", minifyHtml.Minify)
	m.AddFunc("image/svg+xml", svg.Minify)
	m.AddFuncRegexp(regexp.MustCompile("^(application|text)/(x-)?(java|ecma)script$"), js.Minify)
	m.AddFuncRegexp(regexp.MustCompile("[/+]json$"), minify_json.Minify)
	m.AddFuncRegexp(regexp.MustCompile("[/+]xml$"), xml.Minify)

	m.AddFunc("importmap", minify_json.Minify)
	m.AddFunc("speculationrules", minify_json.Minify)

	aspMinifier := &minifyHtml.Minifier{}
	aspMinifier.TemplateDelims = [2]string{"<%", "%>"}
	m.Add("text/asp", aspMinifier)
	m.Add("text/x-ejs-template", aspMinifier)

	return m
}

func minifyBuf(m *minify.M, text bytes.Buffer, mime string) bytes.Buffer {
	var buf bytes.Buffer
	err := m.Minify(mime, &buf, &text)
	if err != nil {
		printerr("Failed to minify %s: %v", mime, err)
		return text
	}
	return buf
}

var minifyTypes = map[string]string{
	".css":              "text/css",
	".html":             "text/html",
	".htm":              "text/html",
	".svg":              "image/svg+xml",
	".js":               "application/javascript",
	".mjs":              "application/javascript",
	".cjs":              "application/javascript",
	".json":             "application/json",
	".xml":              "application/xml",
	".importmap":        "importmap",
	".speculationrules": "speculationrules",
	".asp":              "text/asp",
	".ejs":              "text/x-ejs-template",
}

func minifyStaticFile(m *minify.M, srcPath, dstPath string, info os.FileInfo) error {
	ext := strings.ToLower(filepath.Ext(srcPath))
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return err
	}

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	outFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	if mediaType, ok := minifyTypes[ext]; ok {
		// a minifier is registered for the extension
		writer := bufio.NewWriter(outFile)
		if err := m.Minify(mediaType, writer, srcFile); err != nil {
			return err
		}
		if err := writer.Flush(); err != nil {
			return err
		}
	} else {
		// just copy
		if _, err := io.Copy(outFile, srcFile); err != nil {
			return err
		}
	}

	return os.Chmod(dstPath, info.Mode())
}

var timeFormats = map[string]string{
	"Layout":      time.Layout,
	"ANSIC":       time.ANSIC,
	"UnixDate":    time.UnixDate,
	"RubyDate":    time.RubyDate,
	"RFC822":      time.RFC822,
	"RFC822Z":     time.RFC822Z,
	"RFC850":      time.RFC850,
	"RFC1123":     time.RFC1123,
	"RFC1123Z":    time.RFC1123Z,
	"RFC3339":     time.RFC3339,
	"RFC3339Nano": time.RFC3339Nano,
	"Kitchen":     time.Kitchen,
	"Stamp":       time.Stamp,
	"StampMilli":  time.StampMilli,
	"StampMicro":  time.StampMicro,
	"StampNano":   time.StampNano,
	"DateTime":    time.DateTime,
	"DateOnly":    time.DateOnly,
	"TimeOnly":    time.TimeOnly,
}

func getValueByPath(item any, path string) any {
	if item == nil {
		return nil
	}

	// handle map
	if m, ok := item.(map[string]any); ok {
		return queryMapOrDefault[any](m, nil, strings.Split(path, ".")...)
	}

	// handle *Page
	if page, ok := item.(*Page); ok {
		return queryMapOrDefault[any](page.Metadata, nil, strings.Split(path, ".")...)
	}

	// handle struct
	val := reflect.ValueOf(item)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil
	}

	parts := strings.Split(path, ".")
	current := val
	for _, part := range parts {
		if current.Kind() == reflect.Ptr {
			current = current.Elem()
		}
		if current.Kind() != reflect.Struct {
			return nil
		}

		field := current.FieldByName(strings.Title(part))
		if !field.IsValid() {
			return nil
		}
		current = field
	}

	if current.IsValid() {
		return current.Interface()
	}
	return nil
}

type DirEntry struct {
	Name  string
	IsDir bool
}

func (ctx *buildContext) stdFuncMap(allPages []*Page) text_template.FuncMap {
	return text_template.FuncMap{
		// FS utilities
		"readDir": func(dir string) []*Page {
			var results []*Page
			for _, p := range allPages {
				relPath, _ := filepath.Rel(pagesDir, p.SourcePath)
				if strings.HasPrefix(filepath.ToSlash(relPath), dir+"/") {
					results = append(results, p)
				}
			}
			return results
		},
		"listDir": func(dir string) ([]DirEntry, error) {
			entries, err := os.ReadDir(dir)
			if err != nil {
				return nil, err
			}
			results := make([]DirEntry, 0, len(entries))
			for _, entry := range entries {
				results = append(results, DirEntry{
					Name:  entry.Name(),
					IsDir: entry.IsDir(),
				})
			}
			return results, nil
		},
		"readFile": func(path string) (string, error) {
			cleanPath := filepath.Clean(path)
			if strings.HasPrefix(cleanPath, "..") {
				return "", fmt.Errorf("path cannot be outside the project directory")
			}
			content, err := os.ReadFile(cleanPath)
			return string(content), err
		},

		// time utilities
		"formatTime": func(format string, v any) string {
			var t time.Time
			var err error

			switch val := v.(type) {
			case time.Time:
				t = val
			case string:
				for _, layout := range timeFormats {
					t, err = time.Parse(layout, val)
					if err == nil {
						break
					}
				}
			default:
				return fmt.Sprintf("%v", v) // IDK the type, return the original
			}

			if err != nil {
				return fmt.Sprintf("%v", v)
			}

			realFormat, ok := timeFormats[format]
			if !ok {
				realFormat = format
			}

			return t.UTC().Format(realFormat)
		},
		"now": func() time.Time {
			return time.Now().UTC()
		},

		// random utilities
		"randint": func(min, max int) int {
			return rand.IntN(max-min+1) + min
		},
		"randfloat": func(min, max float64) float64 {
			return min + rand.Float64()*(max-min)
		},
		"choice": func(choices ...any) any {
			if len(choices) == 0 {
				return nil
			}
			return choices[rand.IntN(len(choices))]
		},
		"shuffle": func(slice []any) []any {
			shuffled := make([]any, len(slice))
			copy(shuffled, slice)
			rand.Shuffle(len(shuffled), func(i, j int) {
				shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
			})
			return shuffled
		},

		// string & URL utilities
		"concatURL": func(base string, elements ...string) string {
			u, err := url.Parse(base)
			if err != nil {
				allSegments := make([]string, 1, 1+len(elements))
				allSegments[0] = base
				allSegments = append(allSegments, elements...)
				return strings.Join(allSegments, "/")
			}
			u = u.JoinPath(elements...)
			return u.String()
		},
		"joinPath": func(elem ...string) string {
			return filepath.Join(elem...)
		},
		"truncate": func(s string, maxLength int) string {
			if len(s) <= maxLength {
				return s
			}
			return s[:maxLength] + "…"
		},
		"pluralize": func(count int, singular, plural string) string {
			if count == 1 {
				return fmt.Sprintf("%d %s", count, singular)
			}
			return fmt.Sprintf("%d %s", count, plural)
		},
		"markdownify": func(s string) (string, error) {
			var buf bytes.Buffer
			if err := ctx.MarkdownParser.Convert([]byte(s), &buf); err != nil {
				return "", err
			}
			return buf.String(), nil
		},

		// math utilities
		"add":      func(a, b int) int { return a + b },
		"subtract": func(a, b int) int { return a - b },
		"multiply": func(a, b int) int { return a * b },
		"divide": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"max": func(a, b int) int { return max(a, b) },
		"min": func(a, b int) int { return min(a, b) },

		// array utilities
		"contains": func(slice []any, item any) bool { return slices.Contains(slice, item) },
		"first": func(slice []any) any {
			if len(slice) == 0 {
				return nil
			}
			return slice[0]
		},
		"last": func(slice []any) any {
			if len(slice) == 0 {
				return nil
			}
			return slice[len(slice)-1]
		},
		"reverse": func(slice []any) []any {
			result := make([]any, len(slice))
			for i, v := range slice {
				result[len(slice)-1-i] = v
			}
			return result
		},

		// type conversion utilities
		"toString": func(v any) string {
			return fmt.Sprintf("%v", v)
		},
		"toInt": func(v any) int {
			switch val := v.(type) {
			case int:
				return val
			case float64:
				return int(val)
			case string:
				if i, err := strconv.Atoi(val); err == nil {
					return i
				}
			}
			return 0
		},
		"toFloat": func(v any) float64 {
			switch val := v.(type) {
			case float64:
				return val
			case int:
				return float64(val)
			case string:
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					return f
				}
			}
			return 0
		},

		// conditional utilities
		"default": func(def, val any) any {
			if val == nil || val == "" || val == false {
				return def
			}
			return val
		},
		"ternary": func(condition bool, trueVal, falseVal any) any {
			if condition {
				return trueVal
			}
			return falseVal
		},

		// json utilities
		"toJSON": func(v any) string {
			b, err := json.Marshal(v)
			if err != nil {
				return fmt.Sprintf("error: %v", err)
			}
			return string(b)
		},
		"fromJSON": func(s string) any {
			var result any
			if err := json.Unmarshal([]byte(s), &result); err != nil {
				return nil
			}
			return result
		},

		// query utilities
		"sortBy": func(key string, order string, collection any) any {
			val := reflect.ValueOf(collection)
			if val.Kind() != reflect.Slice {
				return collection
			}

			length := val.Len()
			items := make([]any, length)
			for i := range length {
				items[i] = val.Index(i).Interface()
			}

			sort.SliceStable(items, func(i, j int) bool {
				valI := getValueByPath(items[i], key)
				valJ := getValueByPath(items[j], key)

				strI := fmt.Sprintf("%v", valI)
				strJ := fmt.Sprintf("%v", valJ)

				if strings.ToLower(order) == "desc" {
					return strI > strJ
				}
				return strI < strJ
			})

			return items
		},
		"where": func(key string, value any, collection any) any {
			val := reflect.ValueOf(collection)
			if val.Kind() != reflect.Slice {
				return collection
			}

			length := val.Len()
			var result []any
			for i := range length {
				item := val.Index(i).Interface()
				if itemVal := getValueByPath(item, key); itemVal == value {
					result = append(result, item)
				}
			}
			return result
		},
		"groupBy": func(key string, collection any) map[string][]any {
			val := reflect.ValueOf(collection)
			if val.Kind() != reflect.Slice {
				return map[string][]any{"": {collection}}
			}

			groups := make(map[string][]any)
			length := val.Len()
			for i := range length {
				item := val.Index(i).Interface()
				groupKey := fmt.Sprintf("%v", getValueByPath(item, key))
				groups[groupKey] = append(groups[groupKey], item)
			}
			return groups
		},
		"select": func(key string, collection any) []any {
			val := reflect.ValueOf(collection)
			if val.Kind() != reflect.Slice {
				return []any{getValueByPath(collection, key)}
			}

			var result []any
			length := val.Len()
			for i := range length {
				item := val.Index(i).Interface()
				if val := getValueByPath(item, key); val != nil {
					result = append(result, val)
				}
			}
			return result
		},
		"has": func(key string, item any) bool {
			return getValueByPath(item, key) != nil
		},

		// print & formatting utilities
		"print": func(a ...any) string {
			s := spew.Sdump(a)
			fmt.Println(s)
			return s
		},
		"sprint": func(format string, a ...any) string {
			return spew.Sprintf(format, a)
		},

		// expression evaluation
		"expr": func(expression string, data ...any) (any, error) {
			env := map[string]any{
				"Site":   ctx.Site,
				"Config": ctx.Config,
			}
			if len(data) > 0 {
				if userEnv, ok := data[0].(map[string]any); ok {
					maps.Copy(env, userEnv)
				}
			}

			program, err := expr.Compile(expression, expr.Env(env))
			if err != nil {
				return nil, err
			}

			result, err := expr.Run(program, env)
			if err != nil {
				return nil, err
			}

			return result, nil
		},

		// other
		"dict": func(values ...any) (map[string]any, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("dict expects an even number of arguments")
			}
			m := make(map[string]any)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}
				m[key] = values[i+1]
			}
			return m, nil
		},
	}
}

type buildContext struct {
	Config         map[string]any
	Templates      *template.Template
	MarkdownParser goldmark.Markdown
	Minifier       *minify.M
	Site           Site
	IsServing      bool
}

func build(isServing, copyStatic bool) {
	if !checkAllDirsExist(pagesDir, layoutsDir) {
		printerr("No project files in current directory\n")
		usage()
		os.Exit(1)
	}

	if !isServing {
		_ = os.RemoveAll(outputDir)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		printerr("Failed to create output directory: %v", err)
		os.Exit(1)
	}

	start := time.Now()

	ctx := &buildContext{
		Config:    parseConfig(),
		IsServing: isServing,
		Minifier:  newMinifier(),
	}

	if err := ctx.discoverAndParsePages(); err != nil {
		printerr("Failed to discover pages: %v", err)
	}

	if err := ctx.loadTemplates(); err != nil {
		printerr("Failed to load templates: %v", err)
	}

	for _, page := range ctx.Site.Pages {
		if err := ctx.compileAndRenderPage(page); err != nil {
			printerr("Failed to process page %s: %v", page.SourcePath, err)
		}
	}

	// copy static files
	if copyStatic {
		if err := ctx.copyStaticFiles(); err != nil {
			printerr("Failed to copy static files: %v", err)
		}
	}

	fmt.Printf("%s in %s.\n", color.HiGreenString("Build complete"), time.Since(start))
}

func (ctx *buildContext) loadTemplates() error {
	t := template.New("").Funcs(ctx.stdFuncMap(ctx.Site.Pages))

	var layoutFiles []string
	err := filepath.Walk(layoutsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".html") {
			layoutFiles = append(layoutFiles, path)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("could not walk layouts directory: %w", err)
	}

	if len(layoutFiles) > 0 {
		t, err = t.ParseFiles(layoutFiles...)
		if err != nil {
			return fmt.Errorf("could not parse layout files: %w", err)
		}
	}

	ctx.Templates = t
	return nil
}

func (ctx *buildContext) discoverAndParsePages() error {
	var allPages []*Page
	err := filepath.Walk(pagesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		fileContent, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		page := &Page{
			SourcePath: path,
			IsMarkdown: strings.HasSuffix(path, ".md"),
			RawContent: string(fileContent),
			Metadata:   make(map[string]any),
		}

		mdFrontmatter := goldmark.New(goldmark.WithExtensions(&frontmatter.Extender{Mode: frontmatter.SetMetadata}))
		root := mdFrontmatter.Parser().Parse(text.NewReader([]byte(page.RawContent)))
		page.Metadata = root.OwnerDocument().Meta()

		if _, ok := page.Metadata["title"]; !ok && page.IsMarkdown {
			page.Metadata["title"] = "Untitled"
		}

		relPath, _ := filepath.Rel(pagesDir, path)
		if page.IsMarkdown {
			if strings.HasSuffix(relPath, "index.md") {
				dir := filepath.ToSlash(filepath.Dir(relPath))
				if dir == "." {
					dir = ""
				}
				page.Permalink = "/" + dir + "/"
			} else {
				outPath := strings.TrimSuffix(relPath, filepath.Ext(relPath))
				page.Permalink = "/" + filepath.ToSlash(outPath) + ".html"
			}
		} else {
			page.Permalink = "/" + filepath.ToSlash(relPath)
		}

		if mermaidCompiler == nil && strings.Contains(page.RawContent, "```mermaid") {
			initMermaidCDP()
		}

		allPages = append(allPages, page)
		return nil
	})

	if err != nil {
		return fmt.Errorf("error during page discovery: %w", err)
	}

	ctx.Site.Pages = allPages
	return nil
}

func (ctx *buildContext) compileAndRenderPage(page *Page) error {
	if ctx.MarkdownParser == nil {
		var anchorText *string
		if text, exists := queryMap(ctx.Config, "anchor", "text"); exists {
			if str, ok := text.(string); ok {
				anchorText = &str
			} else {
				anchorText = nil
			}
		} else {
			anchorText = nil
		}

		anchorPosition := anchor.After
		if queryMapOrDefault(ctx.Config, "", "anchor", "position") == "before" {
			anchorPosition = anchor.Before
		}

		ctx.MarkdownParser = createMarkdownParser(
			queryMapOrDefault(ctx.Config, "", "mermaid", "theme"),
			queryMapOrDefault(ctx.Config, true, "d2", "sketch"),
			queryMapOrDefault[int64](ctx.Config, -1, "d2", "theme"),
			queryMapOrDefault(ctx.Config, false, "pikchr", "dark"),
			anchorText,
			anchorPosition,
		)
	}

	tmpl, err := text_template.New(page.SourcePath).Funcs(ctx.stdFuncMap(ctx.Site.Pages)).Parse(page.RawContent)
	if err != nil {
		return fmt.Errorf("failed to parse markdown template: %w", err)
	}

	var processedContent bytes.Buffer
	templateData := TemplateData{Site: ctx.Site, Page: page, Config: ctx.Config, IsServing: ctx.IsServing}
	if err := tmpl.Execute(&processedContent, templateData); err != nil {
		return fmt.Errorf("failed to execute markdown template: %w", err)
	}

	if page.IsMarkdown {
		var finalContent bytes.Buffer
		if err := ctx.MarkdownParser.Convert(processedContent.Bytes(), &finalContent); err != nil {
			return fmt.Errorf("failed to convert markdown: %w", err)
		}
		page.Content = template.HTML(finalContent.String())
	} else {
		page.Content = template.HTML(processedContent.String())
	}

	outputPath := filepath.Join(outputDir, strings.TrimPrefix(page.Permalink, "/"))
	if strings.HasSuffix(page.Permalink, "/") {
		outputPath = filepath.Join(outputPath, "index.html")
	}

	return ctx.renderWithLayout(page, outputPath)
}

func (ctx *buildContext) renderWithLayout(page *Page, outputPath string) error {
	layoutName := "base.html"
	useLayout := true

	if customLayout, ok := page.Metadata["layout"].(string); ok {
		if customLayout == "none" {
			useLayout = false
		} else {
			layoutName = customLayout + ".html"
		}
	}

	var finalBuf bytes.Buffer
	if useLayout && page.IsMarkdown {
		templateData := TemplateData{Site: ctx.Site, Page: page, Config: ctx.Config, IsServing: ctx.IsServing}

		err := ctx.Templates.ExecuteTemplate(&finalBuf, layoutName, templateData)
		if err != nil {
			if ctx.Templates.Lookup(layoutName) == nil {
				printwarn("Layout `%s` not found, falling back to base.html", layoutName)
				err = ctx.Templates.ExecuteTemplate(&finalBuf, "base.html", templateData)
			}
			if err != nil {
				return fmt.Errorf("failed to render layout for %s: %w", page.SourcePath, err)
			}
		}
	} else {
		finalBuf.WriteString(string(page.Content))
	}

	// write output
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", outputPath, err)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", outputPath, err)
	}
	defer file.Close()

	// minify based on file extension
	ext := filepath.Ext(outputPath)
	if mime, ok := minifyTypes[ext]; ok {
		minified := minifyBuf(ctx.Minifier, finalBuf, mime)
		_, err = io.Copy(file, &minified)
	} else {
		_, err = io.Copy(file, &finalBuf)
	}

	return err
}

func (ctx *buildContext) copyStaticFiles() error {
	return filepath.Walk(staticDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		dstPath := filepath.Join(outputDir, path)
		return minifyStaticFile(ctx.Minifier, path, dstPath, info)
	})
}

func parseConfig() map[string]any {
	cfg := make(map[string]any)
	cfgFile, err := os.Open(trashConfigFilename)
	if err != nil {
		printwarn("No `%s` config file found", trashConfigFilename)
		return cfg
	}
	defer cfgFile.Close()

	if err := toml.NewDecoder(cfgFile).Decode(&cfg); err != nil {
		printerr("Error parsing config file: %v", err)
	}
	return cfg
}

type anchorTexter struct {
	text []byte
}

func (a *anchorTexter) AnchorText(*anchor.HeaderInfo) []byte {
	if a == nil {
		return nil
	}
	return a.text
}

func createMarkdownParser(mermaidTheme string, d2Sketch bool, d2Theme int64, pikchrDarkMode bool, anchorText *string, anchorPosition anchor.Position) goldmark.Markdown {
	var d2ThemeId *int64
	if d2Theme >= 0 {
		d2ThemeId = &d2Theme
	}
	var texter anchor.Texter
	if anchorText != nil {
		texter = &anchorTexter{text: []byte(*anchorText)}
	}
	return goldmark.New(
		goldmark.WithRendererOptions(html.WithUnsafe()),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithAttribute(),
		),
		goldmark.WithExtensions(
			extension.GFM,
			extension.DefinitionList,
			extension.Footnote,
			extension.Typographer,
			extension.CJK,
			emoji.Emoji,
			treeblood.MathML(),
			&frontmatter.Extender{},
			&d2.Extender{Sketch: d2Sketch, ThemeID: d2ThemeId},
			&mermaid.Extender{
				Compiler: mermaidCompiler,
				Theme:    mermaidTheme,
			},
			&pikchr.Extender{DarkMode: pikchrDarkMode},
			enclave.New(&enclaveCore.Config{}),
			enclaveCallout.New(),
			highlighting.NewHighlighting(
				highlighting.WithFormatOptions(
					chromahtml.ClassPrefix("highlight-"),
					chromahtml.WithClasses(true),
					chromahtml.WithAllClasses(true),
					chromahtml.LineNumbersInTable(false),
					chromahtml.WithLineNumbers(false),
				),
			),
			&fences.Extender{},
			figure.Figure,
			&anchor.Extender{
				Texter:   texter,
				Position: anchorPosition,
			},
		),
	)
}

// -- watch --

func watchAndRebuild(onRebuild func()) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		printerr("Failed to create file watcher: %v", err)
		os.Exit(1)
	}
	defer watcher.Close()

	for _, path := range []string{pagesDir, staticDir, layoutsDir, trashConfigFilename} {
		filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() || filepath.Base(path) == trashConfigFilename {
				if err := watcher.Add(path); err != nil {
					printerr("Failed to watch directory `%s`: %v", path, err)
				}
			}
			return nil
		})
	}

	var (
		rebuildTimer *time.Timer
		mu           sync.Mutex
	)
	debounceDuration := 250 * time.Millisecond

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				mu.Lock()
				if rebuildTimer != nil {
					rebuildTimer.Stop()
				}
				rebuildTimer = time.AfterFunc(debounceDuration, func() {
					build(true, strings.HasPrefix(event.Name, staticDir+string(filepath.Separator)))
					onRebuild()
				})
				mu.Unlock()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			printerr("Watcher error: %v", err)
		}
	}
}

func watchCmd() {
	build(false, true)

	fmt.Println("Watching for changes...")
	watchAndRebuild(func() {})
}

// -- serve --

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Hub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan []byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.Mutex
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

func (h *Hub) run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.clients[conn] = true
			h.mu.Unlock()
		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				conn.Close()
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.Lock()
			for conn := range h.clients {
				if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
					printerr("Error broadcasting to client: %v", err)
				}
			}
			h.mu.Unlock()
		}
	}
}

func serveCmd() {
	// initial build
	build(true, true)

	hub := newHub()
	go hub.run()

	go watchAndRebuild(func() {
		hub.broadcast <- []byte("reload")
	})

	// live reload websocket handler
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			printerr("Failed to upgrade WebSocket connection: %v", err)
			return
		}
		hub.register <- conn

		go func() {
			defer func() {
				hub.unregister <- conn
			}()
			for {
				if _, _, err := conn.NextReader(); err != nil {
					break
				}
			}
		}()
	})

	fs := http.FileServer(http.Dir("out"))
	http.Handle("/", fs)

	port := "8080"
	fmt.Printf("%s on http://localhost:%s\n", color.HiCyanString("Server starting"), port)
	fmt.Println("Watching for changes...")
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		printerr("Failed to start server: %v", err)
		os.Exit(1)
	}
}

/*
This is free and unencumbered software released into the public domain.

Anyone is free to copy, modify, publish, use, compile, sell, or
distribute this software, either in source code form or as a compiled
binary, for any purpose, commercial or non-commercial, and by any
means.

In jurisdictions that recognize copyright laws, the author or authors
of this software dedicate any and all copyright interest in the
software to the public domain. We make this dedication for the benefit
of the public at large and to the detriment of our heirs and
successors. We intend this dedication to be an overt act of
relinquishment in perpetuity of all present and future rights to this
software under copyright law.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS BE LIABLE FOR ANY CLAIM, DAMAGES OR
OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
OTHER DEALINGS IN THE SOFTWARE.

For more information, please refer to <https://unlicense.org/>
*/
