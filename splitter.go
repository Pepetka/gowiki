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

	before, after, ok := splitBySepLine(md, []byte("---"))
	if !ok {
		return nil, md, nil
	}
	if len(before) != 0 {
		return nil, md, nil
	}

	before, after, ok = splitBySepLine(after, []byte("---"))
	if !ok {
		return nil, md, nil
	}
	if len(before) == 0 {
		return nil, after, nil
	}

	return bytes.TrimSpace(before), bytes.TrimSpace(after), nil
}

func splitBySepLine(s []byte, sep []byte) ([]byte, []byte, bool) {
	j := 0
	for i := range s {
		if s[i] != '\n' {
			continue
		}
		b := bytes.TrimSpace(s[j:i])
		if bytes.Equal(b, sep) {
			return s[:j], s[i+1:], true
		}
		j = i
	}

	return s, nil, false
}
