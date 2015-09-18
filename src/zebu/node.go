package zebu

import (
	"bytes"
	"fmt"
)

const (
	OXXX = iota
	ONONAME

	OGRAM
	ORULE
	OPROD
	OPRODELEM
	OSTRLIT
	OTYPE
	OACTION
	OEPSILON
	OVARID

	OREGDEF
	OCAT
	OALT
	OKLEENE
	OPLUS
	OREPEAT
	OCLASS
	ORANGE
	OCHAR
)

type NodeOp int

var nodeOpLabels = map[NodeOp]string{
	OXXX:      "oxxx",
	ONONAME:   "ononame",
	OGRAM:     "ogram",
	ORULE:     "orule",
	OPROD:     "oprod",
	OREGDEF:   "oregdef",
	OSTRLIT:   "ostrlit",
	OPRODELEM: "oprodelem",
	OTYPE:     "otype",
	OACTION:   "oaction",
	OEPSILON:  "oepsilon",
}

func (n NodeOp) String() string {
	return nodeOpLabels[n]
}

type Node struct {
	// Node shape
	left  *Node
	right *Node
	nodes []*Node
	ntype *Node
	orig  *Node

	// Common
	op      NodeOp
	sym     *Sym
	lit     *Strlit
	byt     byte
	pos     *Position
	isError bool

	// Walking
	resolve  bool
	ll1check bool
	pprint   bool

	// OPRODELEM
	action *Node

	// OACTION/OTYPE
	code []byte

	// OREPEAT
	lb int
	ub int

	// OCLASS
	neg bool
}

var nepsilon = &Node{
	op: OEPSILON,
}

func (n *Node) String() string {
	switch n.op {
	case OGRAM, ORULE, OREGDEF:
		return fmt.Sprintf("%s : %s", n.op, n.sym)
	case OSTRLIT:
		return fmt.Sprintf("%s : %s", n.op, n.lit)
	case OPRODELEM:
		return fmt.Sprintf("%s : (%s)", n.op, n.left)
	default:
		return fmt.Sprintf("%s", n.op)
	}
}

func (n *Node) prodElem() *Node {
	if n.op == OPRODELEM {
		panic("should never build an oprodelem from an oprodleme")
	}
	return &Node{
		op:   OPRODELEM,
		left: n,
	}
}

func (n *Node) dcopy(c *Node) *Node {
	n.left = c.left
	n.right = c.right
	n.nodes = c.nodes
	n.op = c.op
	return n
}

func nodeRuleFromFactoring(dcl *Node, remain [][]*Node) (rule *Node) {
	prods := make([]*Node, 0)
	for _, elmns := range remain {
		if len(elmns) == 0 {
			elmns = append(elmns, &Node{
				op:   OPRODELEM,
				left: nepsilon,
			})
		}
		prods = append(prods, &Node{
			op:    OPROD,
			nodes: elmns,
		})
	}

	rname := primeName(dcl.sym.name)
	s := cc.symbols.lookup(rname)
	rule = &Node{
		op:    ORULE,
		sym:   s,
		nodes: prods,
		orig:  dcl,
	}
	declare(rule)
	return
}

func nodeRuleFromLeftRecursion(dcl *Node, prod *Node) (rule *Node) {
	rname := primeName(dcl.sym.name)
	s := cc.symbols.lookup(rname)
	rule = &Node{
		op:   ORULE,
		sym:  s,
		orig: dcl,
	}
	declare(rule)
	remain := prod.nodes[1:]

	prods := make([]*Node, 0)
	prods = append(prods, &Node{
		op:    OPROD,
		nodes: []*Node{nepsilon.prodElem()},
	})
	prods = append(prods, &Node{
		op:    OPROD,
		nodes: append(remain, rule.prodElem()),
	})
	rule.nodes = prods
	return
}

func escapeStrlit(s string) string {
	var b bytes.Buffer
	for _, c := range s {
		switch c {
		case '\n':
			b.WriteString("\\n")
		default:
			b.WriteString(string(c))
		}
	}
	return b.String()
} 

func (n *Node) dumpTree() {
	w := stdoutCodeWriter()
	walkdump(n, w)
	w.flush()
}

func walkdump(n *Node, w *CodeWriter) {
	if n == nil {
		return
	}
	switch n.op {
	case OGRAM:
		w.writeln("(GRAMMAR: %s", n.sym)
		w.enter()

		for _, n2 := range n.nodes {
			walkdump(n2, w)
		}

		w.writeln(")")
		w.exit()
	case OREGDEF:
		w.writeln("(REGDEF: %s", n.sym)
		w.enter()
		walkdump(n.left, w)
		w.writeln(")")
		w.exit()
	case OCAT:
		w.writeln("(OCAT")
		w.enter()
		walkdump(n.left, w)
		walkdump(n.right, w)
		w.writeln(")")
		w.exit()
	case OALT:
		w.writeln("(OALT")
		w.enter()
		walkdump(n.left, w)
		walkdump(n.right, w)
		w.writeln(")")
		w.exit()
	case OKLEENE, OPLUS:
		w.writeln("(OKLEENE")
		w.enter()
		walkdump(n.left, w)
		w.writeln(")")
		w.exit()
	case OREPEAT:
		w.writeln("(OREPEAT")
		w.enter()
		walkdump(n.left, w)
		w.writeln("(LOW BOUND: %d)", n.lb)
		w.writeln("(UP BOUND: %d)", n.ub)
		w.writeln(")")
		w.exit()
	case OCLASS:
		w.writeln("(OCLASS")
		w.enter()
		for _, n2 := range n.nodes {
			walkdump(n2, w)
		}
		w.writeln(")")
		w.exit()
	case ORANGE:
		w.writeln("(ORANGE")
		w.enter()
		walkdump(n.left, w)
		walkdump(n.right, w)
		w.exit()
	case OCHAR:
		w.writeln("(OCHAR '%c')", n.byt)
	case OSTRLIT:
		w.writeln("(OSTRLIT '%s')", escapeStrlit(n.lit.lit))
	case ORULE:
		w.write("(RULE: %s", n.sym)
		if n.ntype != nil {
			nn := n.ntype
			w.write(" -- TYPE: %s", string(nn.code))
		}
		w.newline()
		w.enter()

		for _, n2 := range n.nodes {
			walkdump(n2, w)
		}

		w.writeln(")")
		w.exit()
	case OPROD:
		w.writeln("(OPROD ")
		w.enter()

		for _, n2 := range n.nodes {
			// Dip directly into PRODELEM
			switch n2.left.op {
			case ORULE:
				w.writeln("(TERMINAL: %s)", n2.left.sym)
			case OREGDEF:
				w.writeln("(NONTERMINAL: %s)", n2.left.sym)
			case OSTRLIT:
				w.writeln("(STRLIT: '%s')", escapeStrlit(n2.left.lit.lit))
			case OEPSILON:
				w.writeln("(OEPSILON)")
			case ONONAME:
				w.writeln("(ONONAME: %s)", n2.left.sym)
			}
			w.enter()
			if n2.right != nil {
				w.writeln("(VARID: %s)", n2.right.sym)
			}
			if n2.action != nil {
				w.writeln("(ACTION)")
			}
			w.exit()
		}

		w.writeln(")")
		w.exit()
	default:
		break
	}
}
