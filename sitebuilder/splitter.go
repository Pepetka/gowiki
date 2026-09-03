package sitebuilder

import (
	"bufio"
	"bytes"
	"errors"
	"io"
)

func splitMetaMd(r io.Reader) (meta []byte, content []byte, err error) {
	buf := bufio.NewReader(r)

	first, err := buf.ReadBytes('\n')
	if errors.Is(err, io.EOF) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}

	if !isDivider(first) {
		c, readErr := io.ReadAll(buf)
		if readErr != nil {
			return nil, nil, readErr
		}
		return nil, append(first, c...), nil
	}

	var metaBuf bytes.Buffer
	for {
		line, readErr := buf.ReadBytes('\n')
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return nil, nil, readErr
		}

		if isDivider(line) {
			c, readErr := io.ReadAll(buf)
			if readErr != nil {
				return nil, nil, readErr
			}
			return metaBuf.Bytes(), c, nil
		}

		_, writeErr := metaBuf.Write(line)
		if writeErr != nil {
			return nil, nil, writeErr
		}

		if errors.Is(err, io.EOF) {
			break
		}
	}

	c, readErr := io.ReadAll(r)
	if readErr != nil {
		return nil, nil, readErr
	}
	return nil, append(first, c...), nil
}

func isDivider(line []byte) bool {
	return bytes.Equal(bytes.TrimSpace(line), []byte("---"))
}
