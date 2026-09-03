// Package sitebuilder is a site builder from markdown files
package sitebuilder

import (
	"errors"
	"html/template"
	"os"
	"path"
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
	mdParser = parser.New(
		parser.WithAutoHeadingID(),
		parser.WithExtensions(extension.GFMParser),
	)
	mdRenderer = html.New(
		html.WithExtensions(extension.GFMHTMLRenderer),
	)
)

func Build(dir string) (err error) {
	tmpDst := "tmp"

	err = os.MkdirAll(tmpDst, 0755)
	if err != nil {
		return err
	}
	defer (func() {
		if removeErr := os.RemoveAll(tmpDst); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) && err == nil {
			err = removeErr
		}
	})()

	err = copyStatic(dir, tmpDst)
	if err != nil {
		return err
	}

	site, err := parseSite(path.Join(dir, "site.yaml"))
	contents, err := parseAllContent(path.Join(dir, "content"))
	iTmpl, pTmpl, err := parseTemplates(path.Join(dir, "templates"))

	data := IndexDocument{
		SiteTitle:       site.Title,
		SiteDescription: site.Description,
		Pages:           contents,
	}
	if saveErr := saveHTML(iTmpl, path.Join(tmpDst, "index.html"), data); saveErr != nil {
		return saveErr
	}

	for _, c := range contents {
		data := PageDocument{
			SiteTitle:       site.Title,
			SiteDescription: site.Description,
			Page:            c,
		}
		if saveErr := saveHTML(pTmpl, path.Join(tmpDst, c.Slug+".html"), data); saveErr != nil {
			return saveErr
		}
	}

	dst := path.Join(dir, "public")
	err = os.RemoveAll(dst)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	err = os.Rename(tmpDst, dst)
	if err != nil {
		return err
	}

	return nil
}

func Create(slug string, dir string) (err error) {
	if validateSlug(slug) {
		return errors.New("invalid slug")
	}

	meta := Meta{
		Title: slug,
		Date:  time.Now(),
		Draft: true,
	}

	fileName := slug + ".md"
	if filepath.Base(dir) == "content" {
		dir = path.Join(dir, "content")
	}
	fp := path.Join(dir, fileName)
	if _, statErr := os.Stat(fp); statErr == nil {
		return errors.New("file already exists")
	}

	f, err := os.Create(fp)
	if err != nil {
		return err
	}
	defer (func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	})()

	_, err = f.WriteString("---\n")
	if err != nil {
		return err
	}
	encoder := yaml.NewEncoder(f)
	if encodeErr := encoder.Encode(meta); encodeErr != nil {
		return errors.New(yaml.FormatError(encodeErr, false, true))
	}
	_, err = f.WriteString("---\n")
	if err != nil {
		return err
	}

	return nil
}

func validateSlug(slug string) bool {
	re := regexp.MustCompile(`^[a-z0-9-]+$`)
	return !re.MatchString(slug)
}

func saveHTML(tmpl *template.Template, path string, data any) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer (func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	})()
	if err := tmpl.Execute(f, data); err != nil {
		return err
	}
	return nil
}

func copyStatic(src string, dst string) error {
	staticSrc := path.Join(src, "static")
	staticDst := path.Join(dst, "static")
	fs := os.DirFS(staticSrc)
	return os.CopyFS(staticDst, fs)
}
