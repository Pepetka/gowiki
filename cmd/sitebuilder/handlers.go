package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/pepetka/gowiki/sitebuilder"
)

func buildHandler(args []string) error {
	dir := ""
	if len(args) > 0 {
		dir = args[0]
	}
	return sitebuilder.Build(dir)
}

func serveHandler(args []string) error {
	dir := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		dir = args[0]
		args = args[1:]
	}

	f := flag.NewFlagSet("serve", flag.ExitOnError)
	var port int
	var build string
	f.IntVar(&port, "port", 8080, "server port")
	f.StringVar(&build, "build", "", "build directory")
	err := f.Parse(args)
	if err != nil {
		return err
	}
	addr := ":" + strconv.Itoa(port)

	args = f.Args()
	if dir == "" && len(args) > 0 {
		dir = args[0]
	}

	if build != "" {
		err = sitebuilder.Build(build)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Serving on http://localhost:%d\n", port)
	return http.ListenAndServe(addr, http.FileServer(http.Dir(dir)))
}

func createHandler(args []string) error {
	if len(args) < 1 {
		return errors.New("no slug")
	}
	slug := args[0]
	args = args[1:]
	dir := ""
	if len(args) > 0 {
		dir = args[0]
	}

	return sitebuilder.Create(slug, dir)
}
