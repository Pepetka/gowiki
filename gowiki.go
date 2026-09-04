// Package gowiki builds a static site from markdown files
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

type Site struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
}

type Meta struct {
	Title string    `yaml:"title"`
	Date  time.Time `yaml:"date"`
	Draft bool      `yaml:"draft"`
}

type IndexDocument struct {
	SiteTitle       string
	SiteDescription string

	Pages []Page
}

type PageDocument struct {
	SiteTitle       string
	SiteDescription string

	Page
}

type Page struct {
	Title string
	Date  time.Time

	Content template.HTML

	Slug string
}

var (
	ErrAlreadyExists = fmt.Errorf("file already exists: %w", os.ErrExist)
	ErrInvalidSlug   = errors.New("invalid slug")
)

var (
	mdParser = parser.New(
		parser.WithAutoHeadingID(),
		parser.WithExtensions(extension.GFMParser),
	)
	mdRenderer = html.New(
		html.WithExtensions(extension.GFMHTMLRenderer),
	)
)

func Build(dir string) (err error) {
	defer (func(dir string) {
		if err != nil {
			if dir == "" {
				dir = "."
			}
			err = fmt.Errorf("gowiki: in %s: %w", dir, err)
		}
	})(dir)
	tmpDst := filepath.Join(dir, "tmp")

	err = os.MkdirAll(tmpDst, 0755)
	if err != nil {
		return err
	}
	defer (func() {
		if !fileOrDirExists(tmpDst) {
			return
		}
		if removeErr := os.RemoveAll(tmpDst); removeErr != nil && err == nil {
			err = removeErr
		}
	})()

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

func Serve(dir string, addr string) (err error) {
	defer (func(dir string, addr string) {
		if err != nil {
			if dir == "" {
				dir = "."
			}
			err = fmt.Errorf("gowiki: in %s at %s: %w", dir, addr, err)
		}
	})(dir, addr)
	dst := dir
	if filepath.Base(dst) != "public" {
		dst = filepath.Join(dir, "public")
	}
	if !fileOrDirExists(dst) {
		return errors.New("public directory does not exist")
	}

	fmt.Printf("Serving on %s\n", addr)
	return http.ListenAndServe(addr, http.FileServer(http.Dir(dst)))
}

func Create(slug string, dir string) (err error) {
	defer (func(slug string, dir string) {
		if err != nil {
			if dir == "" {
				dir = "."
			}
			err = fmt.Errorf("gowiki: %s in %s: %w", slug, dir, err)
		}
	})(slug, dir)
	if validateSlug(slug) {
		return ErrInvalidSlug
	}

	meta := Meta{
		Title: slug,
		Date:  time.Now(),
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
	defer (func() {
		if !fileOrDirExists(tmp) {
			return
		}
		if removeErr := os.RemoveAll(tmp); removeErr != nil && err == nil {
			err = removeErr
		}
	})()
	if saveErr := saveContent(meta, tmp); saveErr != nil {
		return saveErr
	}

	return os.Rename(tmp, fp)
}

func fileOrDirExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if !errors.Is(err, os.ErrNotExist) {
		return false
	}
	return false
}

func saveContent(meta Meta, fp string) (err error) {
	f, err := os.Create(fp)
	if err != nil {
		return err
	}
	defer (func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	})()

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
	re := regexp.MustCompile(`^[a-z0-9-]+$`)
	return !re.MatchString(slug)
}

func saveHTML(tmpl *template.Template, path string, data any) (err error) {
	if mkdirErr := os.MkdirAll(filepath.Dir(path), 0755); mkdirErr != nil {
		return mkdirErr
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer (func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	})()
	return tmpl.Execute(f, data)
}

func copyStatic(src string, dst string) error {
	staticSrc := filepath.Join(src, "static")
	staticDst := filepath.Join(dst, "static")
	fs := os.DirFS(staticSrc)
	return os.CopyFS(staticDst, fs)
}
