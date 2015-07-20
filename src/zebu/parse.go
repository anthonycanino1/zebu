package zebu

import (
	"fmt"
)

// Debug, erase when done
var _ = fmt.Printf

type Parser struct {
	lexer *Lexer
	lh    *Token
}

func NewParser() (p *Parser) {
	p = &Parser{
		lexer: nil,
		lh:    nil,
	}
	return
}

func (p *Parser) next() (t *Token) {
	t = p.lh
	p.lh = p.lexer.Next()
	return
}

func (p *Parser) match(k TokenKind) (t *Token) {
	if t = p.next(); t.kind != k {
		zbError(p.lh.pos, "expected %s", k)
		t = nil
	}
	return
}

func (p *Parser) parseTermName() (s *Sym, err error) {
	t := p.match(TERMINAL)
	if t == nil {
		return
	}
	s = t.sym
	return
}

func (p *Parser) parseNontermName() (s *Sym, err error) {
	t := p.match(NONTERMINAL)
	if t == nil {
		return
	}
	s = t.sym
	return
}

// LAST : Working on the parsing
func (p *Parser) parseChar() (n *Node, err error) {
	switch p.lh.kind {
	case NONTERMINAL:
		s, _ := p.parseNontermName()
		if s.defn == nil {
			zbError(p.lh.pos, "undeclared regular definition!")
			return
		}
		n = s.defn
	case STRLIT:
		n, _ = p.parseStrlit()
	default:
		zbError(p.lh.pos, "expected regular definition or string literal, found %s", p.lh)
		panic("can't handle")
		return
	}
	return
}

func (p *Parser) parseGroup() (n *Node, err error) {
	if p.lh.kind == '(' {
		p.match('(')
		n, _ = p.parseCat()
		p.match(')')
		return
	}
	n, _ = p.parseChar()
	return
}

func (p *Parser) parseClassBody() (n *Node, err error) {
	p.match('[')
	p.match(']')
	return
}

func (p *Parser) parseClass() (n *Node, err error) {
	if p.lh.kind == '[' {
		n, _ = p.parseClassBody()
	} else {
		n, _ = p.parseGroup()
	}
	return
}

func (p *Parser) parseRepeatBody() (lb int, ub int, err error) {
	p.match('{')
	lb = -1
	ub = -1
	if p.lh.kind != ',' {
		t := p.match(NUMLIT)
		if t == nil {
			return
		}
		lb = t.nval
	}
	p.match(',')
	if p.lh.kind != '}' {
		t := p.match(NUMLIT)
		if t == nil {
			return
		}
		ub = t.nval
	}
	p.match('}')
	return
}

func (p *Parser) parseRepeat() (n *Node, err error) {
	n, _ = p.parseClass()
	if n == nil {
		return
	}
	if p.lh.kind == '{' {
		lb, ub, _ := p.parseRepeatBody()
		n = &Node{
			op:   OREPEAT,
			left: n,
			lb:   lb,
			ub:   ub,
		}
	}
	return
}

func (p *Parser) parseKleene() (n *Node, err error) {
	n, _ = p.parseRepeat()
	if n == nil {
		return
	}
	switch p.lh.kind {
	case '*':
		p.match('*')
		n = &Node{
			op:   OKLEENE,
			left: n,
		}
	case '+':
		p.match('+')
		n = &Node{
			op:   OPLUS,
			left: n,
		}
	}
	return
}

func (p *Parser) parseAlt() (n *Node, err error) {
	n, _ = p.parseKleene()
	if n == nil {
		return
	}
	for {
		if p.lh.kind != '|' {
			break
		}
		p.match('|')
		n2, _ := p.parseKleene()
		n = &Node{
			op:    OALT,
			left:  n,
			right: n2,
		}
	}
	return
}

func (p *Parser) parseCat() (n *Node, err error) {
	n, _ = p.parseAlt()
	if n == nil {
		return
	}
loop:
	for {
		switch p.lh.kind {
		case '(', '[', NONTERMINAL, STRLIT:
			n2, _ := p.parseAlt()
			n = &Node{
				op:    OCAT,
				left:  n,
				right: n2,
			}
		default:
			break loop
		}
	}
	return
}

func (p *Parser) parseRegdefDcl() (n *Node, err error) {
	n2, _ := p.parseCat()
	if n2 == nil {
		return
	}
	n = &Node{
		op:   OREGDEF,
		left: n2,
	}
	p.match(';')
	return
}

func (p *Parser) parseRegdef() (n *Node, err error) {
	s, _ := p.parseNontermName()
	if s == nil {
		return
	}
	p.match(':')
	n, _ = p.parseRegdefDcl()
	if n == nil {
		return
	}
	n.sym = s
	declare(n)
	return
}

func (p *Parser) parseAction() (n *Node, err error) {
	p.match('{')
	lvl := 0
	codebuf := make([]byte, 512)
	for {
		c := p.lexer.Raw()
		if c == '}' && lvl == 0 {
			break
		}
		if c == '{' {
			lvl++
		} else if c == '}' {
			lvl--
		}
		codebuf = append(codebuf, c)
	}
	// Reset the lh token
	p.next()

	n = &Node{
		op:   OACTION,
		code: codebuf,
	}

	return
}

func (p *Parser) parseVarId() (s *Sym, err error) {
	t := p.match(VARID)
	if t == nil {
		return
	}
	s = t.sym
	if !pushsym(s) {
		return
	}
	return
}

func (p *Parser) parseEpsilon() (n *Node, err error) {
	n = &Node{
		op: OEPSILON,
	}
	return
}

func (p *Parser) parseStrlit() (n *Node, err error) {
	t := p.match(STRLIT)
	if t == nil {
		return
	}
	n = &Node{
		op:  OSTRLIT,
		lit: t.lit,
	}
	return
}

func (p *Parser) parseNonterm() (n *Node, err error) {
	s, _ := p.parseNontermName()
	if s == nil {
		return
	}
	n = oldname(s)
	return
}

func (p *Parser) parseTerm() (n *Node, err error) {
	s, _ := p.parseTermName()
	if s == nil {
		return
	}
	n = oldname(s)
	return
}

func (p *Parser) parseRuleDcl() (n *Node, err error) {
	var nn *Node
	switch p.lh.kind {
	case TERMINAL:
		nn, _ = p.parseTerm()

	case NONTERMINAL:
		nn, _ = p.parseNonterm()

	case STRLIT:
		nn, _ = p.parseStrlit()

	case '|', '{':
		nn, _ = p.parseEpsilon()

	}
	if nn == nil {
		return
	}
	n = &Node{
		op:   ORDCL,
		left: nn,
	}
	if p.lh.kind == '=' {
		p.match('=')
		n.svar, _ = p.parseVarId()
	}
	if p.lh.kind == '{' {
		n.right, _ = p.parseAction()
	}
	return
}

func (p *Parser) parseRHS() (n *Node, err error) {
	l := new(NodeList)
	marksyms()
	for {
		nn, _ := p.parseRuleDcl()
		if nn == nil {
			break
		}
		l = l.add(nn)
		if p.lh.kind == '|' {
			break
		}
	}
	n = &Node{
		op:    ORHS,
		llist: l,
	}
	popsyms()
	return
}

func (p *Parser) parseRHSList() (l *NodeList, err error) {
	l = new(NodeList)
	for {
		n, _ := p.parseRHS()
		l = l.add(n)
		if p.lh.kind == '|' {
			p.match('|')
			continue
		}
		break
	}
	p.match(';')
	return
}

// Let's do some semantic analysis here, check that
// a type is actually a valid go type.
func (p *Parser) parseType() (n *Node, err error) {
	// For now, just grab the raw string
	typ := make([]byte, 10, 10)
	for {
		c := p.lexer.Raw()
		if c == ':' {
			break
		}
		if isWhitespace(c) {
			continue
		}
		typ = append(typ, c)
	}

	p.lexer.putc(':')
	p.next()

	n = &Node{
		op:   OTYPE,
		code: typ,
	}

	return
}

func (p *Parser) parseLHS() (n *Node, err error) {
	s, _ := p.parseTermName()
	if s == nil {
		return
	}
	n = newname(s)
	return
}

func (p *Parser) parseRule() (n *Node, err error) {
	if n, _ = p.parseLHS(); n == nil {
		return
	}
	n.op = ORULE
	declare(n)

	if p.lh.kind == '=' {
		p.match('=')
		n.ntype, _ = p.parseType()
	}
	p.match(':')

	rl, _ := p.parseRHSList()
	if rl == nil {
		return
	}

	n.rlist = rl
	return
}

func (p *Parser) parseDecl() (n *Node, err error) {
	switch p.lh.kind {
	case TERMINAL:
		n, err = p.parseRule()
	case NONTERMINAL:
		n, err = p.parseRegdef()
	default:
		zbError(p.lh.pos, "expected terminal or nonterminal declaration, found %s", p.lh)
	}
	return
}

func (p *Parser) parseGrammar() (n *Node, err error) {
	if p.match(GRAMMAR) == nil {
		return
	}

	s, _ := p.parseTermName()
	if s == nil {
		return
	}

	n = &Node{
		op: OGRAM,
	}
	n.sym = s
	//declare(n)

	p.match(';')

	l := new(NodeList)
	for p.lh.kind != EOF {
		n2, _ := p.parseDecl()
		if n2 == nil {
			zbError(p.lh.pos, "Error encountered, quiting parse.")
			break
		}
		l = l.add(n2)
	}

	n.rlist = l
	return
}

func (p *Parser) parse(f string) (l *Node) {
	// Let the parse create a lexer for this file
	var err error
	p.lexer, err = NewLexer(f)
	if err != nil {
		panic("could not open file")
	}

	// Start the lexing for a lookahead
	p.lh = p.lexer.Next()

	l, _ = p.parseGrammar()

	return
}
