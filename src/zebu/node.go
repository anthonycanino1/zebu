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
	llist *NodeList
	rlist *NodeList
	nodes []*Node
	ntype *Node

	// Common
	op  NodeOp
	sym *Sym
	lit *Strlit
	byt byte
	pos *Position

	// Walking
	resolve  bool
	ll1check bool

	// OPRODELEM
	svar *Sym

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

func (n *Node) list() *NodeList {
	nl := new(NodeList)
	nl.n = n
	nl.next = nil
	nl.end = nl
	return nl
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
	n.llist = c.llist
	n.rlist = c.rlist
	n.op = c.op
	return n
}

func nodeAsList(n *Node) *NodeList {
	return n.list()
}

func nodeRuleFromFactoring(dcl *Node, remain *[]*NodeList) (rule *Node) {
	prods := new(NodeList)
	for i := 0; i < len(*remain); i++ {
		elmns := (*remain)[i]
		if elmns == nil {
			elmn := &Node{
				op:   OPRODELEM,
				left: nepsilon,
			}
			elmns = elmn.list()
		}
		prods.add(&Node{
			op:    OPROD,
			llist: elmns,
		})
	}

	rname := primeName(dcl.sym.name)
	s := cc.symbols.lookup(rname)
	rule = &Node{
		op:    ORULE,
		sym:   s,
		rlist: prods,
	}
	declare(rule)
	return
}

func nodeRuleFromLeftRecursion(dcl *Node, prod *Node) (rule *Node) {
	rname := primeName(dcl.sym.name)
	s := cc.symbols.lookup(rname)
	rule = &Node{
		op:  ORULE,
		sym: s,
	}
	declare(rule)
	remain := prod.llist.tail()

	prods := new(NodeList)
	prods.add(&Node{
		op:    OPROD,
		llist: nepsilon.prodElem().list(),
	})
	prods.add(&Node{
		op:    OPROD,
		llist: remain.add(rule.prodElem()),
	})
	rule.rlist = prods
	return
}

type NodeList struct {
	n    *Node
	next *NodeList
	end  *NodeList
}

func (l *NodeList) tail() *NodeList {
	if l.next == nil {
		panic("Cannot take the tail of a single element list")
	}
	l.next.end = l.end
	return l.next
}

func (l *NodeList) concat(r *NodeList) *NodeList {
	l.end.next = r
	l.end = r.end
	r.end = nil
	return l
}

func (l *NodeList) add(n *Node) *NodeList {
	if n == nil {
		return l
	}
	if l.n != nil {
		return l.concat(n.list())
	} else {
		l.n = n
		l.end = l
	}
	return l
}

// TODO : We can cache this to optimize later
func (l *NodeList) len() int {
	ln := 0
	for l1 := l; l1 != nil; l1 = l1.next {
		ln++
	}
	return ln
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

// Just for dumping the tree right now
type Writer struct {
	ind   int
	onlin bool
}

func NewWriter() *Writer {
	w := new(Writer)
	w.ind = 0
	w.onlin = false
	return w
}

func (w *Writer) enter() {
	w.ind++
}

func (w *Writer) exit() {
	w.ind--
}

func (w *Writer) doInd() {
	if !w.onlin {
		for i := 0; i < w.ind; i++ {
			fmt.Printf("  ")
		}
		w.onlin = true
	}
}

func (w *Writer) write(f string, args ...interface{}) {
	w.doInd()
	fmt.Printf(f, args...)
}

func (w *Writer) writeln(f string, args ...interface{}) {
	w.doInd()
	fmt.Printf(f, args...)
	fmt.Print("\n")
	w.onlin = false
}

func (w *Writer) newline() {
	fmt.Printf("\n")
	w.onlin = false
}

func (n *Node) dumpOneLevel() {
	if n == nil || n.op != ORULE {
		panic("dumpOneLevel is for debug purposes only, called on non ORULE node")
	}
	for l := n.rlist; l != nil; l = l.next {
		prod := l.n
		fmt.Printf("\t-Prod\n")
		for l2 := prod.llist; l2 != nil; l2 = l2.next {
			elem := l2.n.left
			switch elem.op {
			case ORULE:
				fmt.Printf("\t\t-Nonterminal: %s\n", elem.sym)
			case OREGDEF:
				fmt.Printf("\t\t-Terminal: %s\n", elem.sym)
			case OSTRLIT:
				fmt.Printf("\t\t-Terminal: '%s'\n", escapeStrlit(elem.lit.lit))
			case OEPSILON:
				fmt.Printf("\t\t-Epsilon\n")
			default:
				panic(fmt.Sprintf("unexpected op %s in production element\n", elem.op))
			}
		}
	}
}

func (n *Node) dumpTree() {
	w := NewWriter()
	walkdump(n, w)
}

func walkdump(n *Node, w *Writer) {
	if n == nil {
		return
	}
	switch n.op {
	case OGRAM:
		w.writeln("(GRAMMAR: %s", n.sym)
		w.enter()

		for l := n.rlist; l != nil; l = l.next {
			walkdump(l.n, w)
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
		for l := n.llist; l != nil; l = l.next {
			walkdump(l.n, w)
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
	case ORULE:
		w.write("(RULE: %s", n.sym)
		if n.ntype != nil {
			nn := n.ntype
			w.write(" -- TYPE: %s", string(nn.code))
		}
		w.newline()
		w.enter()

		for l := n.rlist; l != nil; l = l.next {
			walkdump(l.n, w)
		}

		w.writeln(")")
		w.exit()
	case OPROD:
		w.writeln("(OPROD ")
		w.enter()

		for l := n.llist; l != nil; l = l.next {
			// Dip directly into PRODELEM
			n1 := l.n
			switch n1.left.op {
			case ORULE:
				w.writeln("(TERMINAL: %s)", n1.left.sym)
			case OREGDEF:
				w.writeln("(NONTERMINAL: %s)", n1.left.sym)
			case OSTRLIT:
				w.writeln("(STRLIT: '%s')", escapeStrlit(n1.left.lit.lit))
			case OEPSILON:
				w.writeln("(OEPSILON)")
			case ONONAME:
				w.writeln("(ONONAME: %s)", n1.left.sym)
			}
			w.enter()
			if n1.svar != nil {
				w.writeln("(VARID: %s)", n1.svar)
			}
			if n1.right != nil {
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
