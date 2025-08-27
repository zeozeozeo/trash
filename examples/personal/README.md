# examples/personal

A dummy personal homepage.

To view this example properly, [install Trash](/README.md) and do:

```console
$ cd examples/personal
$ trash serve
Build complete in 9.6595ms.
Server starting on http://localhost:8080
Watching for changes...
```

![screenshot of the personal example](./screenshot.png)

The most notable feature of this example is the use of the `listDir` and `readDir` functions to define the navbar using a directory structure (see [pages](./pages)). The pages under "socials" [are also Markdown files](./pages/socials/youtube.md), but their frontmatter defines their order and redirect link in the navbar.

Page layout and style inpired by https://msx.horse/
