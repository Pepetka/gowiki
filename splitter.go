package gowiki

import (
	"bytes"
	"io"
)

func splitMD(r io.Reader) (meta []byte, content []byte, err error) {
	md, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}

	if len(md) == 0 {
		return nil, nil, nil
	}

	oi := bytes.Index(md, []byte("\n"))
	first := bytes.TrimSpace(md[:oi])
	if !bytes.Equal(first, []byte("---")) {
		return nil, md, nil
	}

	g := md[oi+1:]
	ci := bytes.Index(g, []byte("---"))
	if ci == -1 {
		return nil, md, nil
	}
	if ci == 0 {
		return nil, g[oi+3:], nil
	}

	return bytes.TrimSpace(g[:ci]), bytes.TrimSpace(g[ci+3:]), nil
}
