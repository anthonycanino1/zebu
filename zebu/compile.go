// compile.go
// Copyright 2015 The Zebu Authors. All rights reserved.
//

package zebu

import (
	"flag"
	"fmt"
	"sort"
	"go/ast"
)

type Strlit struct {
	lit  string
	link *Strlit
	defn *Node
	gram *Grammar
}

func (s *Strlit) String() string {
	return s.lit
}

type StrlitTab struct {
	table map[string]*Strlit
}

func newStrlitTab() (t *StrlitTab) {
	t = &StrlitTab{
		table: make(map[string]*Strlit),
	}

	return
}

// TODO : Might want to move this from a global function to operate
// directly on the Compiler type
func (t *StrlitTab) lookup(s string) *Strlit {
	return t.lookupGrammar(s, cc.localGrammar)
}

func (t *StrlitTab) lookupGrammar(s string, g *Grammar) (lit *Strlit) {
	for h := t.table[s]; h != nil; h = h.link {
		if h.lit == s && h.gram == g {
			return h
		}
	}

	h := t.table[s]
	lit = &Strlit{
		lit:  s,
		link: h,
		gram: g,
	}
	t.table[s] = lit

	return
}

func (t *StrlitTab) dump() {
	fmt.Printf("----Table----")
	for _, v := range t.table {
		for s := v; s != nil; s = s.link {
			fmt.Printf("%s\n", s)
		}
	}
}

type Sym struct {
	name string
	link *Sym
	pos  *Position // always points to last position for this sym
	gram *Grammar
	defn *Node
	defv bool

	lexical TokenKind // lexical kind associated with this sym
}

func (s *Sym) String() string {
	return s.name
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
	{"lexer", LEXER},
	{"parser", PARSER},
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
	fmt.Printf("----Table----")
	for _, v := range t.table {
		for s := v; s != nil; s = s.link {
			fmt.Printf("%s:%d", s, s.lexical)
		}
	}
}

// TODO : Not sure I want this as a named type, but will leave it for now
type TypeTab struct {
	table map[string]ast.Expr
}

func (t *TypeTab) lookup(s string) (e ast.Expr, ok bool) {
	e, ok = t.table[s]
	return
}

func (t *TypeTab) insert(s string, typ ast.Expr) {
	t.table[s] = typ
}

func newTypeTab() (t *TypeTab) {
	t = &TypeTab{
		table: make(map[string]ast.Expr),
	}

	return
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

type Compiler struct {
	parser *Parser

	localGrammar *Grammar
	symbols      *SymTab
	types        *TypeTab
	strlits      *StrlitTab
	varids       []*Sym
	opt          [256]bool

	unionval map[string]string

	errors       []*CCError
	numSavedErrs int
	numTotalErrs int

	pos *Position

	first  map[*Node]map[*Node]bool
	follow map[*Node]map[*Node]bool
}

type CCError struct {
	pos *Position
	msg string
}

func (ce *CCError) Error() string {
	return ce.msg
}

func newCCError(p *Position, msg string, args ...interface{}) (*CCError) {
	return &CCError{
		pos: p,
		msg: fmt.Sprintf(msg, args...),
	}
}

type CCErrorByPos []*CCError

func (a CCErrorByPos) Len() int {
	return len(a)
}

func (a CCErrorByPos) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a CCErrorByPos) Less(i, j int) bool {
	// TODO : Need a stronger possitioning (sort by order encountered, then by pos inside a file)
	ie, je := a[i].pos, a[j].pos
	if ie.line == je.line {
		return ie.col < je.col
	}
	return ie.line < je.line
}

func (c *Compiler) error(p *Position, msg string, args ...interface{}) (ce *CCError) {
	ce = newCCError(p, msg, args...)
	c.errors = append(c.errors, ce)
	c.numSavedErrs++
	c.numTotalErrs++
	return
}

func (c *Compiler) reerror(p *Position, re *CCError) (ce *CCError) {
	ce = &CCError {
		pos: p,
		msg: re.msg,
	}
	c.errors = append(c.errors, ce)
	c.numSavedErrs++
	c.numTotalErrs++
	return
}

func (c *Compiler) flushErrors() {
	sort.Sort(CCErrorByPos(c.errors))
	for i := 0; i < len(c.errors); i++ {
		fmt.Printf("%s: %s\n", c.errors[i].pos, c.errors[i].msg)
	}
}

var cc *Compiler = nil

func init() {
	localGrammar := NewGrammar("_")
	cc = &Compiler{
		localGrammar: localGrammar,
		symbols:      newSymTab(),
		types:        newTypeTab(),
		strlits:      newStrlitTab(),
		errors:       make([]*CCError, 0, 10),
		parser:       NewParser(),
		first:        make(map[*Node]map[*Node]bool),
		follow:       make(map[*Node]map[*Node]bool),
		varids:       make([]*Sym, 0, 0),
		unionval:     make(map[string]string),
	}

	flag.BoolVar(&cc.opt['h'], "h", false, "print this help message")
	flag.BoolVar(&cc.opt['d'], "d", false, "dump the AST after parsing")
	flag.BoolVar(&cc.opt['t'], "t", false, "dump the AST after transformation")
	flag.BoolVar(&cc.opt['g'], "g", false, "print semantic information about grammar construction")

	// Populate symbol table with known symbols
	for i := 0; i < len(syms); i++ {
		s := cc.symbols.lookup(syms[i].name)
		s.lexical = syms[i].kind
	}
}

// Stuff related to dcls, here for now
func strlitNode(s *Strlit) *Node {
	if s.defn != nil {
		return s.defn
	}
	n := &Node{
		op:  OSTRLIT,
		lit: s,
	}
	s.defn = n
	return n
}

func newname(s *Sym) *Node {
	return &Node{
		op:  ONONAME,
		sym: s,
		pos: s.pos,
	}
}

func oldname(s *Sym) (n *Node) {
	if s.defn == nil {
		n = newname(s)
		declare(n)
		return
	}
	n = s.defn
	return
}

// TODO : Check for position
func declare(n *Node) {
	s := n.sym
	if s.defn != nil && s.defn.op != ONONAME {
		cc.error(s.pos, "%s previously defined at %s.", s, s.defn.pos)
		return
	}
	s.defn = n
	return
}

func resolve(n *Node) *Node {
	if n.op != ONONAME {
		return n
	}
	s := n.sym
	if s.defn == nil || s.defn.op == ONONAME {
		if s.lexical == TERMINAL {
			cc.error(s.pos, "unresolved terminal symbol %s.", s)
		} else {
			cc.error(s.pos, "unresolved nonterminal symbol %s.", s)
		}
		return nil
	}
	return s.defn
}

func pushvarid(s *Sym) (err error) {
	cc.varids = append(cc.varids, s)
	return
}

func popvarids() {
	for _, s := range cc.varids {
		s.defn = nil
	}
	cc.varids = cc.varids[:0]
}

// This is for development purpose only, to take my mind off resolution
// so I can focus on semantic analysis. This resolution has a cost of
// O(n) in the size of the AST, which can be done inside one of the
// other O(n) passes.
func resolveSymbols(n *Node) *Node {
	cc.numSavedErrs = 0
	return walkResolve(n)
}

func walkResolve(n *Node) *Node {
	if n == nil || (n.op != ONONAME && n.resolve) {
		return n
	}
	n.resolve = true
	switch n.op {
	case ONONAME:
		return resolve(n)
	case OGRAM:
		for i := 0; i < len(n.nodes); i++ {
			n.nodes[i] = walkResolve(n.nodes[i])
		}
	case ORULE:
		for i := 0; i < len(n.nodes); i++ {
			for j := 0; j < len(n.nodes[i].nodes); j++ {
				// TODO : Might want to remove the extra lookups
				n.nodes[i].nodes[j].left = walkResolve(n.nodes[i].nodes[j].left)
			}
		}
	case OREGDEF:
		n.left = walkResolve(n.left)
	case OALT, OCAT, OKLEENE, OPLUS, OREPEAT:
		n.left = walkResolve(n.left)
		n.right = walkResolve(n.right)
	}
	return n
}

func Main() {
	flag.Parse()
	args := flag.Args()

	if cc.opt['h'] || len(args) == 0 {
		flag.Usage()
		return
	}

	// Pass #1: Parse grammar (dependencies must be in include path)
	name := args[0]
	top := cc.parser.parse(name)
	if cc.numTotalErrs > 0 {
		cc.flushErrors()
		return
	}

	// Pass #1.5: Resolve symbols (this resolution should be pushed
	// into Pass #2 in the future to amortize the cost).
	top = resolveSymbols(top)
	if cc.numTotalErrs > 0 {
		cc.flushErrors()
		return
	}

	if cc.opt['d'] {
		top.dumpTree()
	}

	// Pass #2: Type check and transform the tree into a valid LL(1) grammar.
	typeCheck(top)

	if cc.numTotalErrs > 0 {
		cc.flushErrors()
		return
	}

	// Pass #3: Generate code in memory for the generated compiler. At this point
	// we should be error free. Use panics to check for cases that should never
	// occur.
	codeGen(top)

	// Pass #4: Dump out the generated code
	codeDump(top)

	return
}