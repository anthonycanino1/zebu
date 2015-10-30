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
	lexerDump(top)
	parserDump(top)
}

var imports = []string{"bufio", "fmt", "os"}

func topDump(top *Node) {
	// 1. Dump supplied code
	if len(top.code) != 0 {
		fmt.Fprintf(codeout, "%s\n", string(top.code[:]))
	}
	fmt.Fprintf(codeout, "\n")

	// 2. Add imports required by our generated cod3
	fmt.Fprintf(codeout, "import (\n")
	for _, v := range imports {
		fmt.Fprintf(codeout, "\"%s\"\n", v)
	}
	fmt.Fprintf(codeout, ")\n")
	fmt.Fprintf(codeout, "\n")
}

func lexerDump(top *Node) {
	// 1. Lexer types
	fmt.Fprintf(codeout, "// Lexing\n")
	fmt.Fprintf(codeout, "type ZbTokenKind int\n")

	fmt.Fprintf(codeout, "const (\n")
	fmt.Fprintf(codeout, "ZBEOF = iota\n")
	fmt.Fprintf(codeout, "ZBUNKNOWN = 0 - iota\n")
	// TODO : We will table this for use later
	for _, n := range top.nodes {
		if n.op != OREGDEF {
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

	fmt.Fprintf(codeout, "type ZbLexer struct {\n")
	fmt.Fprintf(codeout, "buf *bufio.Reader\n")
	fmt.Fprintf(codeout, "}\n")
	fmt.Fprintf(codeout, "\n")

	// 2. Lexer code
	fmt.Fprintf(codeout, "func consZbLexer(buf *bufio.Reader) *ZbLexer {\n")
	fmt.Fprintf(codeout, "return &ZbLexer {\n")
	fmt.Fprintf(codeout, "buf: buf,\n")
	fmt.Fprintf(codeout, "}\n")
	fmt.Fprintf(codeout, "}\n")
	fmt.Fprintf(codeout, "\n")

	fmt.Fprintf(codeout, "func (l *ZbLexer) next() (tok *Token, err error) {\n")
	fmt.Fprintf(codeout, "return\n")
	fmt.Fprintf(codeout, "}\n")
}

func genFriendly(sym *Sym) string {
	s := ""
	b := sym.name[0]
	if b >= 'a' && b <= 'z' {
		b = 'A' + (b - 'a')
	}
	s += string(b)
	for i := 1; i < len(sym.name); i++ {
		if sym.name[i] == '\'' {
			s += "Prime"
		} else {
			s += string(sym.name[i])
		}
	}
	return s
}

func callFriendly(n *Node) string {
	return fmt.Sprintf("parse%s()", genFriendly(n.sym))
}

func caseFriendly(n *Node) string {
	switch n.op {
	case OSTRLIT:
		return fmt.Sprintf("'%s'", n.lit.lit)
	case OREGDEF:
		return fmt.Sprintf("ZB%s", n.sym)
	default:
		panic(fmt.Sprintf("unexpected node op %s during caseFriendly\n", n.op))
	}
}

func firstFriendly(n *Node) map[*Node]bool {
	switch n.op {
	case ORULE:
		return first[n]
	case OREGDEF, OSTRLIT:
		return map[*Node]bool{n: true}
	default:
		panic(fmt.Sprintf("unexpected op %s in firstFriendly\n", n.op))
	}
}

func parserDump(top *Node) {
	// 1. Parser types
	fmt.Fprintf(codeout, "// Parser\n")
	fmt.Fprintf(codeout, "type ZbParser struct {\n")
	fmt.Fprintf(codeout, "lexer *ZbLexer\n")
	fmt.Fprintf(codeout, "lookahead *ZbToken\n")
	fmt.Fprintf(codeout, "}\n")
	fmt.Fprintf(codeout, "\n")

	// 2. Parser code
	fmt.Fprintf(codeout, "func consZbParser(buf *bufio.Reader) *ZbParser {\n")
	fmt.Fprintf(codeout, "return &ZbParser {\n")
	fmt.Fprintf(codeout, "lexer: consZbLexer(buf),\n")
	fmt.Fprintf(codeout, "}\n")
	fmt.Fprintf(codeout, "}\n")
	fmt.Fprintf(codeout, "\n")

	fmt.Fprintf(codeout, "func (p *ZbParser) expect(kind TokenKind) error {\n")
	fmt.Fprintf(codeout, "if p.lookahead != kind {\n")
	fmt.Fprintf(codeout, "fmt.Printf(\"error!\")\n")
	fmt.Fprintf(codeout, "os.Exit(1)\n")
	fmt.Fprintf(codeout, "}\n")
	fmt.Fprintf(codeout, "p.lookahead = p.lexer.next()\n")
	fmt.Fprintf(codeout, "return nil\n")
	fmt.Fprintf(codeout, "}\n")
	fmt.Fprintf(codeout, "\n")

	// 3. Rule code
	for _, n := range top.nodes {
		if n.op != ORULE {
			continue
		}
		ruleDump(n)
	}
}

func ruleDump(n *Node) {
	fmt.Fprintf(codeout, "func (p *ZbParser) parse%s() ", genFriendly(n.sym))
	if n.ntype != nil {
		fmt.Fprintf(codeout, "(result %s, err error) {\n", n.ntype.typ)
	} else {
		fmt.Fprintf(codeout, "(err error) {\n")
	}

	// 3.1. Switch for all productions
	hasepsilon := false
	fmt.Fprintf(codeout, "switch p.lookahead {\n")
	for _, prod := range n.nodes {
		n1 := prod.nodes[0]
		if n1.left.op == OEPSILON {
			hasepsilon = true
			continue
		}
		firsts := firstFriendly(n1.left)
		fmt.Fprintf(codeout, "case ")
		prev := false
		for n1, _ := range firsts {
			if prev {
				fmt.Fprintf(codeout, ", %s", caseFriendly(n1))
			} else {
				fmt.Fprintf(codeout, "%s", caseFriendly(n1))
				prev = true
			}
		}
		fmt.Fprintf(codeout, ":\n")

		// 3.2. Production code
		for _, n1 := range prod.nodes {
			switch n1.left.op {
			case OSTRLIT, OREGDEF:
				fmt.Fprintf(codeout, "expect(%s)\n", caseFriendly(n1.left))
			case ORULE:
				fmt.Fprintf(codeout, "%s\n", callFriendly(n1.left))
			}
		}
	}

	// 3.x. Default and End Switch
	fmt.Fprintf(codeout, "default:\n")
	if hasepsilon {
		fmt.Fprintf(codeout, "// epsilon\n")
		fmt.Fprintf(codeout, "return\n")
	} else {
		fmt.Fprintf(codeout, "fmt.Printf(\"error!\")\n")
		fmt.Fprintf(codeout, "os.Exit(1)\n")
	}
	fmt.Fprintf(codeout, "}\n")

	fmt.Fprintf(codeout, "return\n")
	fmt.Fprintf(codeout, "}\n")
	fmt.Fprintf(codeout, "\n")

}

// Code Generated

// Lex
// type ZbLexer struct {
//   buf *bufio.Reader
// }
// func consZbLexer(buf *bufio.Reader) *ZbLexer { }
// func (l *ZbLexer) next() (tok *Token, err error) { }

// Parser
// type ZbParser struct {
//   lexer *ZbLexer
//	 lookahead *ZbToken
// }

// func consZbParser(buf *bufio.Reader) *ZbParser { }
// func (p *ZbParser) expect(kind ZbTokenKind) (err error) {
//   if p.lookahead != kind {
//     fmt.Printf("error!")
//     os.Exit(1)
//	 }
//   p.lookahead, _ = p.lexer.next()
//}
