// Package gowiki builds a static site from markdown files.
package gowiki

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/yuin/goldmark/v2/extension"
	"github.com/yuin/goldmark/v2/parser"
	"github.com/yuin/goldmark/v2/renderer/html"
)

// Site describes the whole site, loaded from site.yaml.
type Site struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
}

// Meta is the YAML front matter of a single content page.
// Pages with Draft set to true are skipped by Build.
type Meta struct {
	Title string    `yaml:"title"`
	Date  time.Time `yaml:"date"`
	Draft bool      `yaml:"draft"`
}

// IndexDocument is the data passed to the index template.
type IndexDocument struct {
	SiteTitle       string
	SiteDescription string

	Pages []Page
}

// PageDocument is the data passed to the page template.
type PageDocument struct {
	SiteTitle       string
	SiteDescription string

	Page
}

// Page is a single content page rendered to HTML.
type Page struct {
	Title string
	Date  time.Time

	Content template.HTML

	Slug string
}

var (
	// ErrAlreadyExists is returned by Create when the page file already exists.
	ErrAlreadyExists = os.ErrExist
	// ErrInvalidSlug is returned by Create when the slug does not match ^[a-z0-9-]+$.
	ErrInvalidSlug = errors.New("invalid slug")
)

var slugRe = regexp.MustCompile(`^[a-z0-9-]+$`)

var (
	mdParser = parser.New(
		parser.WithAutoHeadingID(),
		parser.WithExtensions(extension.GFMParser),
	)
	mdRenderer = html.New(
		html.WithExtensions(extension.GFMHTMLRenderer),
	)
)

// Build compiles the site in dir into dir/public. The site is assembled in a
// temporary directory and moved into place atomically. Build requires
// dir/site.yaml, dir/content, and dir/templates to exist; dir/static is
// copied to dir/public/static if present.
func Build(dir string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("gowiki: in %s: %w", dir, err)
		}
	}()

	if dir == "" {
		dir = "."
	}
	tmpDst, err := os.MkdirTemp(dir, ".gowiki-*")
	if err != nil {
		return err
	}
	defer func() {
		if removeErr := os.RemoveAll(tmpDst); removeErr != nil && err == nil {
			err = removeErr
		}
	}()

	siteSrc := filepath.Join(dir, "site.yaml")
	if !fileOrDirExists(siteSrc) {
		return errors.New("site.yaml not found")
	}
	contentSrc := filepath.Join(dir, "content")
	if !fileOrDirExists(contentSrc) {
		return errors.New("content directory not found")
	}
	templatesSrc := filepath.Join(dir, "templates")
	if !fileOrDirExists(templatesSrc) {
		return errors.New("templates directory not found")
	}

	site, err := parseSite(siteSrc)
	if err != nil {
		return err
	}
	contents, err := parseAllContent(contentSrc)
	if err != nil {
		return err
	}
	iTmpl, pTmpl, err := parseTemplates(templatesSrc)
	if err != nil {
		return err
	}

	data := IndexDocument{
		SiteTitle:       site.Title,
		SiteDescription: site.Description,
		Pages:           contents,
	}
	if saveErr := saveHTML(iTmpl, filepath.Join(tmpDst, "index.html"), data); saveErr != nil {
		return saveErr
	}

	for _, c := range contents {
		data := PageDocument{
			SiteTitle:       site.Title,
			SiteDescription: site.Description,
			Page:            c,
		}
		if saveErr := saveHTML(pTmpl, filepath.Join(tmpDst, c.Slug+".html"), data); saveErr != nil {
			return saveErr
		}
	}

	err = copyStatic(dir, tmpDst)
	if err != nil {
		return err
	}

	dst := filepath.Join(dir, "public")
	if removeErr := os.RemoveAll(dst); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		return removeErr
	}

	return os.Rename(tmpDst, dst)
}

// Serve serves the built site from dir/public (or dir itself if it is the
// public directory) over HTTP at addr.
func Serve(dir, addr string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("gowiki: in %s at %s: %w", dir, addr, err)
		}
	}()

	if dir == "" {
		dir = "."
	}
	dst := dir
	if filepath.Base(dst) != "public" {
		dst = filepath.Join(dir, "public")
	}
	if !fileOrDirExists(dst) {
		return errors.New("public directory does not exist")
	}

	return http.ListenAndServe(addr, http.FileServer(http.Dir(dst)))
}

// Create writes a new draft page with the given slug into dir/content (or
// dir itself if it is the content directory). The slug must match
// ^[a-z0-9-]+$. Create fails with ErrAlreadyExists if the file exists.
func Create(slug, dir string) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("gowiki: %s in %s: %w", slug, dir, err)
		}
	}()

	if dir == "" {
		dir = "."
	}
	if !validateSlug(slug) {
		return ErrInvalidSlug
	}

	meta := Meta{
		Title: slug,
		Date:  time.Now().Truncate(time.Second),
		Draft: true,
	}

	fileName := slug + ".md"
	dst := dir
	if filepath.Base(dir) != "content" {
		dst = filepath.Join(dir, "content")
	}
	if mkdirErr := os.MkdirAll(dst, 0755); mkdirErr != nil {
		return mkdirErr
	}
	fp := filepath.Join(dst, fileName)
	if fileOrDirExists(fp) {
		return ErrAlreadyExists
	}
	tmp := fp + ".tmp"
	defer func() {
		if removeErr := os.RemoveAll(tmp); removeErr != nil && err == nil {
			err = removeErr
		}
	}()
	if saveErr := saveContent(meta, tmp); saveErr != nil {
		return saveErr
	}

	return os.Rename(tmp, fp)
}

func fileOrDirExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func saveContent(meta Meta, fp string) (err error) {
	f, err := os.Create(fp)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	if _, writeErr := f.WriteString("---\n"); writeErr != nil {
		return writeErr
	}
	encoder := yaml.NewEncoder(f)
	if encodeErr := encoder.Encode(meta); encodeErr != nil {
		return encodeErr
	}
	if _, writeErr := f.WriteString("---\n"); writeErr != nil {
		return writeErr
	}
	return nil
}

func validateSlug(slug string) bool {
	return slugRe.MatchString(slug)
}

func saveHTML(tmpl *template.Template, path string, data any) (err error) {
	if mkdirErr := os.MkdirAll(filepath.Dir(path), 0755); mkdirErr != nil {
		return mkdirErr
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()
	return tmpl.Execute(f, data)
}

func copyStatic(src, dst string) error {
	staticSrc := filepath.Join(src, "static")
	staticDst := filepath.Join(dst, "static")
	if !fileOrDirExists(staticSrc) {
		return nil
	}
	fs := os.DirFS(staticSrc)
	return os.CopyFS(staticDst, fs)
}
