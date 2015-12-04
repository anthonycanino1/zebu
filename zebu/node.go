package zebu

import (
	"bytes"
	"fmt"
	"go/ast"
)

const (
	OXXX = iota
	ONONAME

	OGRAM
	ORULE
	OPROD

	// OPRODDCL is the merge of an OPRODELEM and OVARID
	OPRODDCL
	OSELF

	OSTRLIT
	OTYPE
	OACTION
	OEPSILON

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
	OXXX:     "oxxx",
	ONONAME:  "ononame",
	OGRAM:    "ogram",
	ORULE:    "orule",
	OPROD:    "oprod",
	OREGDEF:  "oregdef",
	OSTRLIT:  "ostrlit",
	OTYPE:    "otype",
	OACTION:  "oaction",
	OEPSILON: "oepsilon",
	OPRODDCL: "oproddcl",
}

func (n NodeOp) String() string {
	return nodeOpLabels[n]
}

// TODO: Fix the way dpn is handled across OACTION and everything else
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
	dpn     []*Node // OPRODDCL that this node depends upon

	// Walking
	resolve  bool
	ll1check bool
	pprint   bool

	// OPRODDCL
	used bool

	// OACTION/OTYPE
	code  []byte
	typ   string
	etype ast.Expr

	// OREPEAT
	lb int
	ub int

	// OCLASS
	neg bool
}

var nepsilon = &Node{
	op: OEPSILON,
}

// HINT: This is where I may need to rework things
func (n *Node) prodDcl() *Node {
	if n.op == OPRODDCL {
		panic("should never build an oprodelem from an oprodleme")
	}
	return &Node{
		op:   OPRODDCL,
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
				op:   OPRODDCL,
				left: nepsilon,
			})
		}

		prods = append(prods, &Node{
			op:    OPROD,
			nodes: elmns,
		})
	}

	rname := primeName(dcl.sym.name)
	s := symbols.lookup(rname)
	rule = &Node{
		op:    ORULE,
		sym:   s,
		nodes: prods,
		orig:  dcl,
	}
	declare(rule)
	return
}

// NOTE: Example for the uglyness of OOP
func nodeRuleFromLeftRecursion(dcl *Node, prod *Node) (rule *Node) {
	rname := primeName(dcl.sym.name)
	s := symbols.lookup(rname)
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
		nodes: []*Node{nepsilon.prodDcl()},
	})

	prods = append(prods, &Node{
		op:    OPROD,
		nodes: append(remain, rule.prodDcl()),
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
	w := consStdoutCodeWriter()
	walkdump(n, w)
	w.flush()
}

func walkstring(n *Node) string {
	switch n.left.op {
	case ORULE:
		return fmt.Sprintf("(TERMINAL: %s -- VAR:%s)", n.left.sym, n.sym)
	case OREGDEF:
		return fmt.Sprintf("(NONTERMINAL: %s -- VAR:%s)", n.left.sym, n.sym)
	case OSTRLIT:
		return fmt.Sprintf("(STRLIT: '%s' -- VAR:%s)", escapeStrlit(n.left.lit.lit), n.sym)
	case OEPSILON:
		return fmt.Sprintf("(OEPSILON -- VAR:%s)", n.sym)
	case ONONAME:
		return fmt.Sprintf("(ONONAME: %s -- VAR:%s)", n.left.sym, n.sym)
	case OSELF:
		return fmt.Sprintf("(OSELF -- VAR:%s)", n.sym)
	default:
		panic(fmt.Sprintf("unexpected op %s in walkstring\n", n.left.op))
	}
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
		w.write("(REGDEF: %s", n.sym)
		if n.ntype != nil {
			w.write(" -- TYPE: %s", n.ntype.typ)
		}
		w.newline()
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
			w.write(" -- TYPE: %s", n.ntype.typ)
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
			switch n2.left.op {
			case ORULE:
				w.write("(TERMINAL: %s -- VAR:%s", n2.left.sym, n2.sym)
			case OREGDEF:
				w.write("(NONTERMINAL: %s -- VAR:%s", n2.left.sym, n2.sym)
			case OSTRLIT:
				w.write("(STRLIT: '%s' -- VAR:%s", escapeStrlit(n2.left.lit.lit), n2.sym)
			case OEPSILON:
				w.write("(OEPSILON -- VAR:%s", n2.sym)
			case ONONAME:
				w.write("(ONONAME: %s -- VAR:%s", n2.left.sym, n2.sym)
			default:
				panic(fmt.Sprintf("unexpected op %s in walkdump\n", n2.left.op))
			}
			if n2.right == nil {
				w.writeln(")")
				continue
			}
			w.newline()
			w.enter()
			w.writeln("(OACTION: %s)", n2.right.sym)
			w.writeln(")")
			w.exit()
		}

		w.writeln(")")
		w.exit()
	default:
		break
	}
}
