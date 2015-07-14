package zebu

import (
	"fmt"
)

const (
	OXXX = iota
	ONONAME

	OGRAM
	ORULE
	OTYPE
	ORHS
	OREGDEF
	OSTRLIT
	ORDCL
)

type NodeOp int

var nodeOpLabels = map[NodeOp]string{
	OXXX:    "oxxx",
	ONONAME: "ononame",
	OGRAM:   "ogram",
	ORULE:   "orule",
	ORHS:    "orhs",
	OREGDEF: "oregdef",
	OSTRLIT: "ostrlit",
	ORDCL:   "ordcl",
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
	typ   *Node

	// Common
	op   NodeOp
	dump bool

	sym *Sym
	lit string

	// ORDCL
	svar *Sym
}

func NewNode(op NodeOp, l *Node, r *Node) (n *Node) {
	n = new(Node)
	n.op = op
	n.left = l
	n.right = r
	return n
}

func (n *Node) list() *NodeList {
	nl := new(NodeList)
	nl.n = n
	nl.next = nil
	nl.tail = nl
	return nl
}

func (n *Node) dcopy(c *Node) *Node {
	n.left = c.left
	n.right = c.right
	n.llist = c.llist
	n.rlist = c.rlist
	n.op = c.op
	n.dump = c.dump
	return n
}

type NodeList struct {
	n    *Node
	next *NodeList
	tail *NodeList
}

func (l *NodeList) concat(r *NodeList) *NodeList {
	l.tail.next = r
	l.tail = r.tail
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
		l.tail = l
	}
	return l
}

// Stuff related to dcls, here for now
func newname(s *Sym) *Node {
	return &Node{
		op:  ONONAME,
		sym: s,
	}
}

func oldname(s *Sym) (n *Node) {
	if s.defn == nil {
		// Since we have no notion of scope, we will use a trick
		// Declare a dummy node that all future reference will point to
		// for the symbol.
		// Resolve at the end.
		n = newname(s)
		declare(n)
		cc.unresolved[s] = n
		return
	}
	n = s.defn
	return
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

func (n *Node) dumpTree() {
	w := NewWriter()
	walkdump(n, w)
}

func walkdump(n *Node, w *Writer) {
	if n == nil || n.dump {
		return
	}
	n.dump = true
	switch n.op {
	case OGRAM:
		w.writeln("(GRAMMAR: %s", n.sym)
		w.enter()

		for l := n.rlist; l != nil; l = l.next {
			walkdump(l.n, w)
		}

		w.writeln(")")
		w.exit()
	case ORULE:
		w.writeln("(RULE: %s", n.sym)
		w.enter()

		for l := n.rlist; l != nil; l = l.next {
			walkdump(l.n, w)
		}

		w.writeln(")")
		w.exit()
	case ORHS:
		w.writeln("(RHS ")
		w.enter()

		for l := n.llist; l != nil; l = l.next {
			nn := l.n
			switch nn.left.op {
			case ORULE:
				w.write("(TERMINAL: %s)", nn.left.sym)
			case OSTRLIT:
				w.write("(STRLIT: %s)", nn.left.lit)
			}
			if nn.svar != nil {
				w.write("=%s", nn.svar)
			}
			w.newline()
		}

		w.writeln(")")
		w.exit()
	case OSTRLIT:
		w.writeln("(STRLIT: %s)", n.lit)
	default:
		break
	}
}
