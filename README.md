# gowiki

A minimal static site generator written in Go: Markdown pages with YAML
front matter are compiled to plain HTML using your own templates. Can be
used as a command-line tool or as a library.

## Features

- Builds a static site from Markdown files (GFM: tables, task lists, strikethrough)
- YAML front matter per page: `title`, `date`, `draft`
- Pages with `draft: true` are skipped entirely
- Index page lists all pages, newest first (undated pages last, sorted by title)
- Your own `html/template` templates — index, layout, page, and partials
  (layout and partials apply to pages only; the index template stands alone)
- Static assets are copied as-is
- Built-in dev server with optional rebuild
- Atomic builds: the site is assembled in a temp directory and swapped in with a rename

## Installation

```sh
go install github.com/pepetka/gowiki/cmd/gowiki@latest
```

Or build from source:

```sh
git clone https://github.com/pepetka/gowiki.git
cd gowiki
go build -o gowiki ./cmd/gowiki
```

## Usage

```sh
gowiki <command> [arguments] [flags]
```

### build

Build the site into `public/`:

```sh
gowiki build path/to/site
```

### serve

Serve the built site over HTTP; use `--build` to rebuild first:

```sh
gowiki serve path/to/site
gowiki serve path/to/site --build --port 3000
```

| Flag      | Default | Description          |
| --------- | ------- | -------------------- |
| `--port`  | `8080`  | Server port          |
| `--build` | `false` | Build before serving |

### new

Create a new page (a draft Markdown file with front matter) in `content/`:

```sh
gowiki new my-first-post path/to/site
```

The slug must match `^[a-z0-9-]+$`. The file is created atomically and
fails if it already exists.

## Site structure

```text
site/
├── site.yaml       # site title and description
├── content/        # Markdown pages
├── templates/      # index.html, layout.html, page.html, partials
├── static/         # copied to public/static as-is (optional)
└── public/         # build output
```

`site.yaml`:

```yaml
title: "My Wiki"
description: "Notes on everything"
```

A page in `content/`:

```markdown
---
title: "Hello, world"
date: 2026-08-20T10:00:00+03:00
draft: false
---

Regular **Markdown** here.
```

Templates use `html/template`. `index.html` gets an `IndexDocument`
(`SiteTitle`, `SiteDescription`, `Pages`); `page.html` — a `PageDocument`
(plus `Title`, `Date`, `Content`, `Slug`). See the `example/` directory
for a complete working site.

## Library usage

```sh
go get github.com/pepetka/gowiki
```

```go
package main

import (
	"log"

	"github.com/pepetka/gowiki"
)

func main() {
	if err := gowiki.Create("my-first-post", "site"); err != nil {
		log.Fatal(err)
	}
	if err := gowiki.Build("site"); err != nil {
		log.Fatal(err)
	}
	if err := gowiki.Serve("site", ":8080"); err != nil {
		log.Fatal(err)
	}
}
```

## License

[MIT](LICENSE)
