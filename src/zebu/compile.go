package zebu

import (
	"fmt"
)

type Compiler struct {
	localGrammar *Grammar
	symbols      *SymTab
	unresolved   map[*Sym]*Node
	symscope     *SymList
}

var cc *Compiler = nil

func init() {
	localGrammar := NewGrammar("_")
	cc = &Compiler{
		localGrammar: localGrammar,
		symbols:      newSymTab(),
		unresolved:   make(map[*Sym]*Node),
	}

	// Populate symbol table with known symbols
	for i := 0; i < len(syms); i++ {
		s := cc.symbols.lookup(syms[i].name)
		s.lexical = syms[i].kind
	}
}

func Compile(f string) {
	p := NewParser()
	tr := p.parse(f)
	tr.dumpTree()
}

type Sym struct {
	name string
	pos  *Position
	link *Sym
	gram *Grammar
	defn *Node
	defv bool

	lexical TokenKind // lexical kind associated with this sym
}

func (s *Sym) String() string {
	return s.name
}

func (s *Sym) list() *SymList {
	sl := new(SymList)
	sl.s = s
	sl.next = nil
	sl.tail = sl
	return sl
}

var syms = []struct {
	name string
	kind TokenKind
}{
	{"grammar", GRAMMAR},
	{"import", IMPORT},
	{"keyword", KEYWORD},
	{"extend", EXTEND},
	{"inherit", INHERIT},
	{"override", OVERRIDE},
	{"delete", DELETE},
	{"modify", MODIFY},
}

type SymList struct {
	s    *Sym
	next *SymList
	tail *SymList
}

func (l *SymList) concat(r *SymList) *SymList {
	l.tail.next = r
	l.tail = r.tail
	return l
}

func (l *SymList) add(s *Sym) *SymList {
	if s == nil {
		return l
	}
	if l.s != nil {
		return l.concat(s.list())
	} else {
		l.s = s
		l.tail = l
	}
	return l
} 

type SymTab struct {
	table map[string]*Sym
}

func newSymTab() (t *SymTab) {
	t = &SymTab{
		table: make(map[string]*Sym),
	}

	return
}

func (t *SymTab) lookup(s string) *Sym {
	return t.lookupGrammar(s, cc.localGrammar)
}

func (t *SymTab) lookupGrammar(s string, g *Grammar) (sym *Sym) {
	for h := t.table[s]; h != nil; h = h.link {
		if h.name == s && h.gram == g {
			return h
		}
	}

	h := t.table[s]
	sym = &Sym{
		name:    s,
		pos:     nil,
		link:    h,
		gram:    g,
		lexical: NAME, // this get's refined
	}
	t.table[s] = sym

	return
}

func (t *SymTab) dump() {
	dbg("----Table----")
	for _, v := range t.table {
		for s := v; s != nil; s = s.link {
			dbg("%s:%d", s, s.lexical)
		}
	}
}

type Grammar struct {
	name string
}

func NewGrammar(s string) (g *Grammar) {
	g = &Grammar{
		name: s,
	}
	return
}

func (g *Grammar) String() string {
	return g.name
}

func zbError(p *Position, m string, args ...interface{}) {
	fmt.Printf("%s: %s\n", p, fmt.Sprintf(m, args...))
}

// dcl

func declare(n *Node) {
	s := n.sym
	if s.defn != nil && s.defn.op != ONONAME {
		// Previously declared, flag error
		fmt.Printf("Previously declared error")
		return
	}
	if s.defn != nil && s.defn.op == ONONAME {
		// Remove from unresolved
		if _, ok := cc.unresolved[s]; !ok {
			panic("ONONAME decl not found in unresolved map!")
		}
		delete(cc.unresolved, s)
		s.defn.dcopy(n)
		return
	}
	s.defn = n
	return
}

func marksyms() {
	if cc.symscope != nil {
		panic("marking should always start with empty symscope")
	}
	cc.symscope = new(SymList)
}

func pushsym(s *Sym) bool {
	if s.defv {
		fmt.Printf("multiply defined varid %s\n", s)
		return false
	}
	cc.symscope = cc.symscope.add(s)
	s.defv = true
	return true
}

func popsyms() {
	for l := cc.symscope; l != nil; l = l.next {
		if l.s == nil {
			continue
		}
		l.s.defv = false
	}
	cc.symscope = nil
}
