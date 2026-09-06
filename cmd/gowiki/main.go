package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		return errors.New("no command")
	}
	cmd, err := parseCommand(args[0])
	if err != nil {
		return err
	}
	switch cmd {
	case BuildCommand:
		return buildHandler(args[1:])
	case ServeCommand:
		return serveHandler(args[1:])
	case NewCommand:
		return createHandler(args[1:])
	default:
		return errors.New("unknown command")
	}
}

type Command string

const (
	BuildCommand Command = "build"
	ServeCommand Command = "serve"
	NewCommand   Command = "new"
)

func parseCommand(raw string) (Command, error) {
	parsed := Command(raw)
	switch parsed {
	case BuildCommand, ServeCommand, NewCommand:
		return parsed, nil
	default:
		return "", fmt.Errorf("unknown command: %s", raw)
	}
}
