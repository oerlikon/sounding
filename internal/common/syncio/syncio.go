package syncio

import (
	"io"
	"sync"
)

type StringWriter struct {
	sync.Mutex
	w io.StringWriter
}

func NewStringWriter(w io.StringWriter) *StringWriter {
	return &StringWriter{w: w}
}

func (w *StringWriter) WriteString(s string) (n int, err error) {
	w.Lock()
	defer w.Unlock()
	return w.w.WriteString(s)
}
