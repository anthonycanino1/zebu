// gen.go
// Copyright 2015 The Zebu Authors. All rights reserved.
//
package zebu

import (
	"fmt"
) 

func pprint(top *Node) {
	if top.op != OGRAM {
		panic("pprint should be called on top node\n")
	}
	w := consStdoutCodeWriter()

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
		w.write(" ; ")
		w.newline()
	case OALT:
		pprintWalk(n.left, w)
		if n.right != nil {
			w.write("|")
			pprintWalk(n.right, w)
		}
	case OCAT:
		pprintWalk(n.left, w)
		w.write(" ")
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

func codeGen(top *Node) {
}

func codeDump(top *Node) {
	topDump(top)
	lexDump(top)
}

func topDump(top *Node) {
	if len(top.code) != 0 {
		fmt.Fprintf(codeout, "%s\n", string(top.code[:]))
	}
	fmt.Fprintf(codeout, "\n")
}

func lexDump(top *Node) {
	/* 1. lex types */
	fmt.Fprintf(codeout, "type ZbTokenKind int\n")

	fmt.Fprintf(codeout, "const (\n")
	fmt.Fprintf(codeout, "ZBEOF = iota\n")
	fmt.Fprintf(codeout, "ZBUNKNOWN = 0 - iota\n")
	// TODO : We will table this for use later
	for _, n := range top.nodes {
		if (n.op != OREGDEF) {
			continue
		}
		fmt.Fprintf(codeout, "ZB%s\n", n.sym)
	}
	fmt.Fprintf(codeout, ")\n")
	fmt.Fprintf(codeout, "\n")

	fmt.Fprintf(codeout, "type ZbToken struct {\n")
	fmt.Fprintf(codeout, "pos int\n")
	fmt.Fprintf(codeout, "kind ZbTokenKind\n")
	fmt.Fprintf(codeout, "val interface{}\n")
	fmt.Fprintf(codeout, "}\n")
}

// Code Generated 

// Lex
// func Zblex() (tok *Token, err error) { } 


