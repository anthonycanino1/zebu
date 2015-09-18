// gen.go
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

func NewCodeWriter(writer io.Writer) *CodeWriter {
	return &CodeWriter{
		ind:    0,
		onlin:  false,
		writer: bufio.NewWriter(writer),
	}
}

func stdoutCodeWriter() *CodeWriter {
	return NewCodeWriter(os.Stdout)
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

func pprint(top *Node) {
	if top.op != OGRAM {
		panic("pprint should be called on top node\n")
	}
	w := stdoutCodeWriter()

	w.writeln("grammar %s ;", top.sym)
	w.newline()

	for _, n := range top.nodes {
		pprintWalk(n, w)
		w.newline()
	}

	w.flush()
}

func pprintWalk(n *Node, w *CodeWriter) {
	if n.pprint {
		switch n.op {
		case OREGDEF:
			w.write("%s", n.sym)
			return
		}
	}
	n.pprint = true

	switch n.op {
	case ORULE:
		w.writeln("%s", n.sym)
		first := true
		w.enter()
		for _, p := range n.nodes {
			if first {
				w.write(": ")
				first = false
			} else {
				w.write("| ")
			}

			for _, e := range p.nodes {
				switch e.left.op {
				case OSTRLIT:
					w.write("'%s' ", escapeStrlit(e.left.lit.lit))
				case OREGDEF, ORULE:
					w.write("%s ", e.left.sym)
				case OEPSILON:
					w.write("/* epsilon */ ")
				default:
					panic(fmt.Sprintf("unexpected op %s in pprintWalk\n", e.left.op))
				}
			}

			w.newline()
		}
		w.writeln(";")
		w.exit()
	case OREGDEF:
		w.write("%s : ", n.sym)
		pprintWalk(n.left, w)
		w.write("; ")
		w.newline()
	case OALT:
		pprintWalk(n.left, w)
		if n.right != nil {
			w.write("|")
			pprintWalk(n.right, w)
		}
	case OCAT:
		pprintWalk(n.left, w)
		pprintWalk(n.right, w)
	case OKLEENE:
		pprintWalk(n.left, w)
		w.write("*")
	case OPLUS:
		pprintWalk(n.left, w)
		w.write("+")
	case OREPEAT:
		pprintWalk(n.left, w)
		w.write("{")
		if n.lb != -1 {
			w.write("%d", n.lb)
		}
		w.write(",")
		if n.lb != -1 {
			w.write("%d", n.ub)
		}
		w.write("}")
	case OCLASS:
		w.write("[")
		if n.neg {
			w.write("^")
		}
		for _, p := range n.nodes {
			switch p.op {
			case OCHAR:
				w.write("%s", string(p.byt))
			case ORANGE:
				w.write("%s", string(p.left.byt))
				w.write("-")
				w.write("%s", string(p.right.byt))
			default:
				panic(fmt.Sprintf("unexpected op %s in pprintWalk", p.op))
			}
		}
		w.write("]")
	case OSTRLIT:
		w.write("'%s'", n.lit.lit)
	}
}
