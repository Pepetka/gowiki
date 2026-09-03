package sitebuilder

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
)

func parseSite(path string) (site Site, err error) {
	file, err := os.Open(path)
	if err != nil {
		return site, err
	}
	defer (func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	})()

	decoder := yaml.NewDecoder(file, yaml.Strict())
	if err = decoder.Decode(&site); err != nil {
		return site, errors.New(yaml.FormatError(err, false, true))
	}
	return site, nil
}

func parseAllContent(dir string) ([]Page, error) {
	c := make([]Page, 0)

	err := dirWalker(dir, func(relPath string) error {
		meta, content, err := parseContent(path.Join(dir, relPath))
		if err != nil {
			return err
		}
		if meta.Draft {
			return nil
		}
		c = append(c, Page{
			Title:   meta.Title,
			Date:    meta.Date,
			Content: content,
			Slug:    strings.TrimSuffix(relPath, ".md"),
		})
		return nil
	}, ".md")
	if err != nil {
		return nil, err
	}
	slices.SortFunc(c, func(a, b Page) int {
		aTime := a.Date
		bTime := b.Date

		if aTime.Equal(bTime) {
			return strings.Compare(a.Title, b.Title)
		}
		if aTime.After(bTime) {
			return 1
		} else {
			return -1
		}
	})

	return c, nil
}

func parseContent(path string) (meta Meta, content template.HTML, err error) {
	file, err := os.Open(path)
	if err != nil {
		return meta, template.HTML(""), err
	}
	defer (func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	})()

	metaBytes, c, err := splitMetaMd(file)
	if err != nil {
		return meta, template.HTML(""), err
	}

	if err = yaml.UnmarshalWithOptions(metaBytes, &meta, yaml.Strict()); err != nil {
		return meta, template.HTML(""), errors.New(yaml.FormatError(err, false, true))
	}

	parsed := mdParser.Parse(c)
	var buf bytes.Buffer
	if renderErr := mdRenderer.Render(&buf, c, parsed); renderErr != nil {
		return meta, template.HTML(""), renderErr
	}

	return meta, template.HTML(buf.String()), nil
}

type PageTemplates struct {
	Index string

	Layout string
	Page   string
	Others []string
}

func parseTemplates(dir string) (*template.Template, *template.Template, error) {
	pt, err := groupTemplates(dir)
	if err != nil {
		return nil, nil, err
	}
	if pt.Index == "" {
		return nil, nil, fmt.Errorf("index template not found")
	}
	if pt.Page == "" {
		return nil, nil, fmt.Errorf("page template not found")
	}

	iTmpl, err := template.ParseFiles(pt.Index)
	if err != nil {
		return nil, nil, err
	}

	tf := make([]string, 0, 3)
	if pt.Layout != "" {
		tf = append(tf, pt.Layout)
	}
	tf = append(tf, pt.Page)
	if len(pt.Others) > 0 {
		tf = append(tf, pt.Others...)
	}
	pTmpl, err := template.ParseFiles(tf...)
	if err != nil {
		return nil, nil, err
	}

	return iTmpl, pTmpl, nil
}

func groupTemplates(root string) (PageTemplates, error) {
	allowedExt := ".html"
	pt := PageTemplates{}

	files, err := os.ReadDir(root)
	if err != nil {
		return pt, err
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		switch f.Name() {
		case "index.html":
			pt.Index = path.Join(root, f.Name())
		case "layout.html":
			pt.Layout = path.Join(root, f.Name())
		case "page.html":
			pt.Page = path.Join(root, f.Name())
		default:
			if strings.HasSuffix(f.Name(), allowedExt) {
				pt.Others = append(pt.Others, path.Join(root, f.Name()))
			}
		}
	}
	return pt, nil
}

func dirWalker(root string, fn func(path string) error, allowedExt string) error {
	fileSistem := os.DirFS(root)

	err := fs.WalkDir(fileSistem, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(p, allowedExt) {
			return fn(p)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
