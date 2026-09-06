package main

import (
	"errors"
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/pepetka/gowiki"
)

func buildHandler(args []string) error {
	dir := ""
	if len(args) > 0 {
		dir = args[0]
	}
	return gowiki.Build(dir)
}

func serveHandler(args []string) error {
	dir := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		dir = args[0]
		args = args[1:]
	}

	f := flag.NewFlagSet("serve", flag.ExitOnError)
	var port int
	var build bool
	f.IntVar(&port, "port", 8080, "server port")
	f.BoolVar(&build, "build", false, "build before serving")
	err := f.Parse(args)
	if err != nil {
		return err
	}
	addr := ":" + strconv.Itoa(port)

	args = f.Args()
	if dir == "" && len(args) > 0 {
		dir = args[0]
	}

	if build {
		err = gowiki.Build(dir)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Serving on %s\n", addr)
	return gowiki.Serve(dir, addr)
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

	return gowiki.Create(slug, dir)
}
