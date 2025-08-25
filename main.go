// Trash - a stupid, simple website compiler.
// Licensing information at the bottom of this file.
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	text_template "text/template"
	"time"

	_ "embed"

	d2 "github.com/FurqanSoftware/goldmark-d2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/fatih/color"
	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
	pikchr "github.com/jchenry/goldmark-pikchr"
	figure "github.com/mangoumbrella/goldmark-figure"
	enclave "github.com/quailyquaily/goldmark-enclave"
	enclaveCallout "github.com/quailyquaily/goldmark-enclave/callout"
	enclaveCore "github.com/quailyquaily/goldmark-enclave/core"
	fences "github.com/stefanfritsch/goldmark-fences"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	minifyHtml "github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"github.com/tdewolff/minify/v2/json"
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
		buildCmd(false, true)
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
			fmt.Println("enter 'y' or 'n'.")
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

// -- init --

const (
	pagesDir    = "pages"
	staticDir   = "static"
	templateDir = "templates"
	outputDir   = "out"
)

func initCmd() {
	if !isEmptyDir() {
		if !ask("The current directory is not empty. Are you sure you want to continue? (Y/n): ") {
			fmt.Println("Aborting.")
			return
		}
	}

	makedirs(pagesDir+"/posts", staticDir, templateDir)

	// templates/base.html
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
</html>`, "templates", "base.html")

	// static/style.css
	writefile(`body {
  font-family: sans-serif;
  color: #333;
}

.container {
  max-width: 800px;
  margin: 2rem auto;
  padding: 0 1rem;
}`, "static", "style.css")

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

	writefile("/out", ".gitignore")

	programName := getProgramName()
	fmt.Printf("You can now do %s to build your site, %s to rebuild on file changes, or %s to start a server with live reloading.\n", color.HiBlueString(programName), color.HiBlueString(programName+" watch"), color.HiBlueString(programName+" serve"))
}

// -- build --

type Page struct {
	// SourcePath is the path to the original .md file relative to the project root.
	SourcePath string
	// Permalink is the final URL path for the page.
	Permalink string
	// Markdown is the raw markdown content, without front matter.
	Markdown string
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
	IsServing bool
}

func maybeInitMermaidCDP() {
	if mermaidCompiler != nil {
		return
	}
	var err error
	mermaidCompiler, err = mermaidcdp.New(&mermaidcdp.Config{
		JSSource: mermaidJSSource,
	})
	if err != nil {
		printerr("Failed to initialize Mermaid with CDP: %v; falling back to clientside JS", err)
	}
}

func ptr[T any](v T) *T {
	return &v
}

func newMinifier() *minify.M {
	m := minify.New()
	m.AddFunc("text/css", css.Minify)
	m.AddFunc("text/html", minifyHtml.Minify)
	m.AddFunc("image/svg+xml", svg.Minify)
	m.AddFuncRegexp(regexp.MustCompile("^(application|text)/(x-)?(java|ecma)script$"), js.Minify)
	m.AddFuncRegexp(regexp.MustCompile("[/+]json$"), json.Minify)
	m.AddFuncRegexp(regexp.MustCompile("[/+]xml$"), xml.Minify)

	m.AddFunc("importmap", json.Minify)
	m.AddFunc("speculationrules", json.Minify)

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

	writer := bufio.NewWriter(outFile)

	if mediaType, ok := minifyTypes[ext]; ok {
		// a minifier is registered for the extension
		if err := m.Minify(mediaType, writer, srcFile); err != nil {
			return err
		}
		if err := writer.Flush(); err != nil {
			return err
		}
	} else {
		// just copy
		if _, err := io.Copy(writer, srcFile); err != nil {
			return err
		}
		if err := writer.Flush(); err != nil {
			return err
		}
	}

	return os.Chmod(dstPath, info.Mode())
}

func buildCmd(isServing, copyStatic bool) {
	if !checkAllDirsExist(pagesDir, templateDir) {
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

	// pass 1: discover markdown files
	var allPages []*Page
	var hasMermaid bool
	err := filepath.Walk(pagesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		fileContent, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		page := &Page{
			SourcePath: path,
			Metadata:   make(map[string]any),
			Markdown:   string(fileContent),
		}

		hasMermaid = hasMermaid || strings.Contains(page.Markdown, "```mermaid")

		// generate permalink
		relPath, _ := filepath.Rel(pagesDir, path)
		page.Permalink = "/" + strings.TrimSuffix(filepath.ToSlash(relPath), ".md") + ".html"
		if filepath.Base(relPath) == "index.md" {
			dir := filepath.ToSlash(filepath.Dir(relPath))
			if dir == "." {
				dir = ""
			}
			page.Permalink = "/" + dir + "/"
		}

		allPages = append(allPages, page)
		return nil
	})
	if err != nil {
		printerr("Failed to walk pages directory: %v", err)
		os.Exit(1)
	}

	site := Site{Pages: allPages}

	// pass 2: parse all frontmatter
	{
		mdFrontmatter := goldmark.New(
			goldmark.WithExtensions(&frontmatter.Extender{Mode: frontmatter.SetMetadata}),
		)
		for _, page := range allPages {
			root := mdFrontmatter.Parser().Parse(text.NewReader([]byte(page.Markdown)))
			doc := root.OwnerDocument()
			page.Metadata = doc.Meta()

			if _, ok := page.Metadata["title"]; !ok {
				page.Metadata["title"] = "Untitled"
			}
		}
	}

	funcMap := text_template.FuncMap{
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
		"sortBy": func(key string, order string, pages []*Page) []*Page {
			sort.SliceStable(pages, func(i, j int) bool {
				valI, okI := pages[i].Metadata[key]
				valJ, okJ := pages[j].Metadata[key]
				if !okI {
					return false
				}
				if !okJ {
					return true
				}

				strI := fmt.Sprintf("%v", valI)
				strJ := fmt.Sprintf("%v", valJ)

				if strings.ToLower(order) == "desc" {
					return strI > strJ
				}
				return strI < strJ
			})
			return pages
		},
	}

	if hasMermaid {
		maybeInitMermaidCDP()
	}

	mdContent := goldmark.New(
		goldmark.WithRendererOptions(html.WithUnsafe()),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithExtensions(
			extension.GFM,
			extension.DefinitionList,
			extension.Footnote,
			emoji.Emoji,
			treeblood.MathML(),
			&frontmatter.Extender{},
			&d2.Extender{Sketch: true, ThemeID: ptr(int64(200))}, // see https://d2lang.com/tour/themes/
			&mermaid.Extender{
				Compiler: mermaidCompiler,
			},
			&pikchr.Extender{},
			enclave.New(&enclaveCore.Config{}),
			enclaveCallout.New(),
			highlighting.NewHighlighting(
				highlighting.WithFormatOptions(
					chromahtml.ClassPrefix("highlight-"),
					chromahtml.WithClasses(true),
					chromahtml.WithAllClasses(true),
					chromahtml.LineNumbersInTable(false), // TODO
					chromahtml.WithLineNumbers(false),    // TODO
				),
			),
			&fences.Extender{},
			figure.Figure,
			&anchor.Extender{},
		),
	)

	layout, err := template.ParseFiles(filepath.Join(templateDir, "base.html"))
	if err != nil {
		printerr("Could not parse base layout template: %v", err)
		os.Exit(1)
	}

	// pass 3: process markdown and render HTML
	m := newMinifier()
	for _, page := range allPages {
		tmpl, err := text_template.New(page.SourcePath).Funcs(funcMap).Parse(page.Markdown)
		if err != nil {
			printerr("Failed to parse markdown template for %s: %v", page.SourcePath, err)
			continue
		}

		var processedMd bytes.Buffer
		if err := tmpl.Execute(&processedMd, TemplateData{Site: site, Page: page}); err != nil {
			printerr("Failed to execute markdown template for %s: %v", page.SourcePath, err)
			continue
		}

		var finalHtml bytes.Buffer
		if err := mdContent.Convert(processedMd.Bytes(), &finalHtml); err != nil {
			printerr("Failed to convert markdown for %s: %v", page.SourcePath, err)
			continue
		}
		page.Content = template.HTML(finalHtml.String())

		outputPath := filepath.Join(outputDir, strings.TrimPrefix(page.Permalink, "/"))
		if strings.HasSuffix(page.Permalink, "/") {
			outputPath = filepath.Join(outputPath, "index.html")
		}

		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			printerr("Failed to create directory for %s: %v", outputPath, err)
			continue
		}

		var finalBuf bytes.Buffer
		if err := layout.Execute(&finalBuf, TemplateData{Site: site, Page: page, IsServing: isServing}); err != nil {
			printerr("Failed to render final HTML for %s: %v", page.SourcePath, err)
			continue
		}

		file, err := os.Create(outputPath)
		if err != nil {
			printerr("Failed to create file %s: %v", outputPath, err)
			continue
		}
		defer file.Close()

		minifiedBuf := minifyBuf(m, finalBuf, "text/html")
		io.Copy(file, &minifiedBuf)
	}

	// copy static files
	if copyStatic {
		filepath.Walk(staticDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			relPath, _ := filepath.Rel(staticDir, path)
			dstPath := filepath.Join(outputDir, relPath)

			return minifyStaticFile(m, path, dstPath, info)
		})
	}

	fmt.Printf("%s in %s.\n", color.HiGreenString("Build complete"), time.Since(start))
}

// -- watch --

func watchAndRebuild(onRebuild func()) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		printerr("Failed to create file watcher: %v", err)
		os.Exit(1)
	}
	defer watcher.Close()

	for _, path := range []string{pagesDir, staticDir, templateDir} {
		filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				if err := watcher.Add(path); err != nil {
					printerr("Failed to watch directory %s: %v", path, err)
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
					buildCmd(true, strings.HasPrefix(event.Name, staticDir+string(filepath.Separator)))
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
	buildCmd(false, true)

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
	buildCmd(true, true)

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
