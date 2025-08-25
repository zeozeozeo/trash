---
title: "Trash Demo"
date: "2025-08-24"
---

Hello, world! This is a demo of the Trash website compiler. :wastebasket: <- `:wastebasket:`

To view it properly, you can ask Trash to serve it with live-reloading:

```bash
$ cd examples/blog
$ trash serve
Build complete in 324.5177ms.
Server starting on http://localhost:8080
Watching for changes...
```

All of the following elements can be compiled fully on the server side, serving **zero client-side JS**!

### $LaTeX$ expressions

Given the radius $r$ of a circle, the area $A$ is:

$$
A = \pi \times r^2
$$

And the circumference $C$ is:

$$
C = 2 \pi r
$$

_The page will live-reload if you change any of this!_

### D2 diagram:

```d2
dogs -> cats -> mice: chase
replica 1 <-> replica 2
a -> b: To err is human, to moo bovine {
  source-arrowhead: 1
  target-arrowhead: * {
    shape: diamond
  }
}
```

### Mermaid Mindmap:

```mermaid
mindmap
  root((Problem))
    Category A
      Cause A
        Cause C
    Category B
      Cause B
        Cause D
        Cause E
    Category C
      Usual Cause A
      Usual Cause B
    Category D
      Usual Cause C
      Usual Cause D
```

### Pikchr diagram:

```pikchr
arrow right 200% "Markdown" "Source"
box rad 10px "Markdown" "Formatter" "(markdown.c)" fit
arrow right 200% "HTML+SVG" "Output"
arrow <-> down 70% from last box.s
box same "Pikchr" "Formatter" "(pikchr.c)" fit
```

### Embed YouTube videos & audio files

...with native Markdown syntax!

![](https://www.youtube.com/watch?v=dQw4w9WgXcQ)

![](https://archive.org/download/tvtunes_26154/My%20Little%20Pony%20-%20Friendship%20is%20Magic%20-%20Babs%20Seed.mp3)

### Syntax highlighting

[Go](https://go.dev/):

```go
func main() {
    fmt.Println("ok")
}
```

JavaScript:

```js
"b" + "a" + +"a" + "a"; // -> 'baNaNa'
```

:::{.blue}

### Life Inside Fences

This paragraph is inside a fenced block.

:::{#insideme .red data="important"}
You can nest and assign custom IDs to them.
:::
:::

### Callouts

Trash also supports [GitHub style callouts](https://github.com/orgs/community/discussions/16925):

> [!NOTE]  
> Highlights information that users should take into account, even when skimming.

TIP: Optional information to help a user be more successful.

IMPORTANT
Crucial information necessary for users to succeed.

> [!WARNING]  
> Critical content demanding immediate user attention due to potential risks.

> [!CAUTION]
> Negative potential consequences of an action.

### Image figures

This is an extension of Markdown that allows you to place `<figure>` elements by typing text below an image:

<div style="display:flex; gap:20px; align-items:center;">
  <div>

![](/rainbow.webp?h=100px)
Rainbow Dash

  </div>
  <div>

![](/rarity.webp?h=100px)
Rarity

  </div>
</div>

Note how we're appending `?w=100px` after the image URL and Trash automatically knows to make it 100px, even though the host doesn't support it:

```markdown
![](https://your-image.com/image.png?w=100px)
![alt text](https://example.com/image.png|200 "title")
![alt text|200x300](https://example.com/image.png "title")
![alt text|200px](https://example.com/image.png "title")
![alt text|50%](https://example.com/image.png "title")
![alt text|50%](https://example.com/image.png?align=left "title")
```

(you might remember this from [Obsidian](https://obsidian.md/))
