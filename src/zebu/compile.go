package zebu

import (
	"flag"
	"fmt"
	"sort"
)

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

// TODO : Might want to move this from a global function to operate
// directly on the Compiler type
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
	unresolved   map[*Sym]*Node
	symscope     *SymList
	opt          [256]bool

	errors       []*CCError
	numSavedErrs int
	numTotalErrs int

	pos *Position

	first  map[*Node]map[*Node]bool
	follow map[*Node][]*Node
}

type CCError struct {
	pos *Position
	msg string
}

func (ce *CCError) Error() string {
	return ce.msg
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
	ce = &CCError{
		pos: p,
		msg: fmt.Sprintf(msg, args...),
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
		unresolved:   make(map[*Sym]*Node),
		errors:       make([]*CCError, 0, 10),
		parser:       NewParser(),
		first:				make(map[*Node]map[*Node]bool),
	}

	flag.BoolVar(&cc.opt['h'], "help", false, "print this help message")
	flag.BoolVar(&cc.opt['d'], "dump", false, "dump the AST after parsing")
	flag.BoolVar(&cc.opt['g'], "grammar", false, "print semantic information about grammar construction")

	// Populate symbol table with known symbols
	for i := 0; i < len(syms); i++ {
		s := cc.symbols.lookup(syms[i].name)
		s.lexical = syms[i].kind
	}
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

func declare(n *Node) {
	s := n.sym
	if s.defn != nil && s.defn.op != ONONAME {
		cc.error(cc.pos, "%s previously defined at %s.", s, s.pos)
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
	s.pos = cc.pos
	return
}

func marksyms() {
	if cc.symscope != nil {
		panic("marking should always start with empty symscope")
	}
	cc.symscope = new(SymList)
}

func pushsym(s *Sym) (err error) {
	if s.defv {
		//cc.error("multiply defined varid %s", s)
		return fmt.Errorf("multiply defined varid %s", s)
	}
	cc.symscope = cc.symscope.add(s)
	s.defv = true
	return
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

func Main() {
	flag.Parse()
	args := flag.Args()

	if cc.opt['h'] || len(args) == 0 {
		flag.Usage()
		return
	}

	// Pass #1: Parse grammar (dependencies must be in include path)
	name := args[0]
	grammar := cc.parser.parse(name)
	if cc.numTotalErrs > 0 {
		cc.flushErrors()
		return
	}

	if cc.opt['d'] {
		grammar.dumpTree()
	}

	// Pass #2: Perform semantic analysis over the grammar. Transforms
	// the grammar to a valid LL(1) grammar if possible.
	//semanticPass(grammar)

	// Pass #3: Generate a DFA for the lexer

	// Pass #4: Codegen

	fmt.Printf("Successful parse!\n")

	return
}
