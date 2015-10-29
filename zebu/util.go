// util.go
// Copyright 2015 The Zebu Authors. All rights reserved.
//
package zebu

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

type CodeWriter struct {
	ind    int
	onlin  bool
	writer *bufio.Writer
}

func consCodeWriter(writer io.Writer) *CodeWriter {
	return &CodeWriter{
		ind:    0,
		onlin:  false,
		writer: bufio.NewWriter(writer),
	}
}

func consStdoutCodeWriter() *CodeWriter {
	return consCodeWriter(os.Stdout)
}

func (w *CodeWriter) enter() {
	w.ind++
}

func (w *CodeWriter) exit() {
	w.ind--
}

func (w *CodeWriter) doInd() {
	if !w.onlin {
		for i := 0; i < w.ind; i++ {
			fmt.Fprintf(w.writer, "  ")
		}
		w.onlin = true
	}
}

func (w *CodeWriter) write(f string, args ...interface{}) {
	w.doInd()
	fmt.Fprintf(w.writer, f, args...)
}

func (w *CodeWriter) writeln(f string, args ...interface{}) {
	w.doInd()
	fmt.Fprintf(w.writer, f, args...)
	fmt.Fprint(w.writer, "\n")
	w.onlin = false
}

func (w *CodeWriter) newline() {
	fmt.Fprintf(w.writer, "\n")
	w.onlin = false
}

func (w *CodeWriter) flush() {
	w.writer.Flush()
}


