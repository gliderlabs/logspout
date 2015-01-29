package main

import "io"

type IgnorantWriter struct {
	original io.Writer
}

func (w IgnorantWriter) Write(p []byte) (int, error) {
	n, _ := w.original.Write(p)
	return n, nil
}
