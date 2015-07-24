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

func (p *Parser) match(k TokenKind) (t *Token, err error) {
	if t = p.next(); t.kind != k {
		t = nil
		err = cc.error(p.lh.pos, "expected %s", k)
	}
	return
}

func (p *Parser) parseTermName() (s *Sym, err error) {
	t, err := p.match(TERMINAL)
	if err != nil {
		return
	}
	s = t.sym
	return
}

func (p *Parser) parseNontermName() (s *Sym, err error) {
	t, err := p.match(NONTERMINAL)
	if err != nil {
		return
	}
	s = t.sym
	return
}

func (p *Parser) parseChar() (n *Node, err error) {
	switch p.lh.kind {
	case NONTERMINAL:
		var s *Sym
		if s, err = p.parseNontermName(); err != nil {
			return
		}
		if s.defn == nil {
			err = cc.error(p.lh.pos, "undeclared regular definition!")
			return
		}
		n = s.defn
	case STRLIT:
		n, err = p.parseStrlit()
	default:
		err = cc.error(p.lh.pos, "expected regular definition or string literal, found %s", p.lh)
		return
	}
	return
}

func (p *Parser) parseGroup() (n *Node, err error) {
	if p.lh.kind == '(' {
		_, err = p.match('(')
		if err != nil {
			return
		}
		n, err = p.parseCat()
		if err != nil {
			return
		}
		_, err = p.match(')')
		if err != nil {
			return
		}
		return
	}
	n, err = p.parseChar()
	return
}

func (p *Parser) parseClassBodyChar() (n *Node, err error) {
	var t *Token
	if t, err = p.match(REGLIT); err != nil {
		return
	}
	n = &Node {
		op: OCHAR,
		byt: t.byt,
	}
	return
}

func (p *Parser) parseClassBodyRange() (n *Node, err error) {
	var n1 *Node
	if n1, err = p.parseClassBodyChar(); err != nil {
		return
	}
	if p.lh.kind != '-' {
		n = n1
		return
	}
	p.match('-')
	var n2 *Node
	if n2, err = p.parseClassBodyChar(); err != nil {
		return
	}
	n = &Node{
		op:    ORANGE,
		left:  n1,
		right: n2,
	}
	return
}

func (p *Parser) parseClassBody() (n *Node, err error) {
	if _, err = p.match('['); err != nil {
		return
	}
	neg := false
	if p.lh.kind == '^' {
		neg = true
		p.match('^')
	}
	l := new(NodeList)
	for {
		if p.lh.kind == ']' {
			break
		}
		var n1 *Node
		if n1, err = p.parseClassBodyRange(); err != nil {
			return
		}
		l = l.add(n1)
	}
	_, err = p.match(']')
	n = &Node{
		op:    OCLASS,
		llist: l,
		neg:   neg,
	}
	return
}

func (p *Parser) parseClass() (n *Node, err error) {
	if p.lh.kind == '[' {
		n, err = p.parseClassBody()
	} else {
		n, err = p.parseGroup()
	}
	return
}

func (p *Parser) parseRepeatBody() (lb int, ub int, err error) {
	p.match('{')
	lb = -1
	ub = -1
	if p.lh.kind != ',' {
		var t *Token
		if t, err = p.match(NUMLIT); err != nil {
			return
		}
		lb = t.nval
	}
	if _, err = p.match(','); err != nil {
		return
	}
	if p.lh.kind != '}' {
		var t *Token
		if t, err = p.match(NUMLIT); err != nil {
			return
		}
		ub = t.nval
	}
	_, err = p.match('}')
	return
}

func (p *Parser) parseRepeat() (n *Node, err error) {
	n, err = p.parseClass()
	if err != nil {
		return
	}
	if p.lh.kind == '{' {
		var lb, ub int
		if lb, ub, err = p.parseRepeatBody(); err != nil {
			return
		}
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
	if n, err = p.parseRepeat(); err != nil {
		return
	}
	switch p.lh.kind {
	case '*':
		if _, err = p.match('*'); err != nil {
			return
		}
		n = &Node{
			op:   OKLEENE,
			left: n,
		}
	case '+':
		if _, err = p.match('+'); err != nil {
			return
		}
		n = &Node{
			op:   OPLUS,
			left: n,
		}
	}
	return
}

func (p *Parser) parseAlt() (n *Node, err error) {
	if n, err = p.parseKleene(); err != nil {
		return
	}
	for {
		if p.lh.kind != '|' {
			break
		}
		p.match('|')
		var n2 *Node
		if n2, err = p.parseKleene(); err != nil {
			return
		}
		n = &Node{
			op:    OALT,
			left:  n,
			right: n2,
		}
	}
	return
}

func (p *Parser) parseCat() (n *Node, err error) {
	n, err = p.parseAlt()
	if err != nil {
		return
	}
loop:
	for {
		switch p.lh.kind {
		case '(', '[', NONTERMINAL, STRLIT:
			var n2 *Node
			if n2, err = p.parseAlt(); err != nil {
				return
			}
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
	n2, err := p.parseCat()
	if err != nil {
		return
	}
	n = &Node{
		op:   OREGDEF,
		left: n2,
	}
	return
}

func (p *Parser) parseRegdef() (n *Node, err error) {
	s, err := p.parseNontermName()
	if err != nil {
		return
	}
	_, err = p.match(':')
	if err != nil {
		return
	}
	n, err = p.parseRegdefDcl()
	if err != nil {
		return
	}
	n.sym = s
	declare(n)
	return
}

func (p *Parser) parseAction() (n *Node, err error) {
	_, err = p.match('{')
	if err != nil {
		return
	}
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
	t, err := p.match(VARID)
	if err != nil {
		return
	}
	s = t.sym
	err = pushsym(s)
	return
}

func (p *Parser) parseEpsilon() (n *Node, err error) {
	n = &Node{
		op: OEPSILON,
	}
	return
}

func (p *Parser) parseStrlit() (n *Node, err error) {
	t, err := p.match(STRLIT)
	if err != nil {
		return
	}
	n = &Node{
		op:  OSTRLIT,
		lit: t.lit,
	}
	return
}

func (p *Parser) parseNonterm() (n *Node, err error) {
	s, err := p.parseNontermName()
	if err != nil {
		return
	}
	n = oldname(s)
	return
}

func (p *Parser) parseTerm() (n *Node, err error) {
	s, err := p.parseTermName()
	if err != nil {
		return
	}
	n = oldname(s)
	return
}

func (p *Parser) parseRuleDcl() (n *Node, err error) {
	var nn *Node
	switch p.lh.kind {
	case TERMINAL:
		nn, err = p.parseTerm()

	case NONTERMINAL:
		nn, err = p.parseNonterm()

	case STRLIT:
		nn, err = p.parseStrlit()

	case '|', '{':
		nn, err = p.parseEpsilon()

	}
	if err != nil {
		return
	}
	n = &Node{
		op:   ORDCL,
		left: nn,
	}
	if p.lh.kind == '=' {
		p.match('=')
		n.svar, err = p.parseVarId()
		if err != nil {
			return
		}
	}
	if p.lh.kind == '{' {
		n.right, err = p.parseAction()
	}
	return
}

func (p *Parser) parseRHS() (n *Node, err error) {
	l := new(NodeList)
	marksyms()
	defer popsyms()
	for {
		var nn *Node
		if nn, err = p.parseRuleDcl(); err != nil {
			return
		}
		l = l.add(nn)
		if p.lh.kind == '|' || p.lh.kind == ';' {
			break
		}
	}
	n = &Node{
		op:    ORHS,
		llist: l,
	}
	return
}

func (p *Parser) parseRHSList() (l *NodeList, err error) {
	l = new(NodeList)
	for {
		var n *Node
		if n, err = p.parseRHS(); err != nil {
			return
		}
		l = l.add(n)
		if p.lh.kind == '|' {
			p.match('|')
			continue
		}
		break
	}
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
	s, err := p.parseTermName()
	if err != nil {
		return
	}
	n = newname(s)
	return
}

func (p *Parser) parseRule() (n *Node, err error) {
	if n, err = p.parseLHS(); err != nil {
		return
	}
	n.op = ORULE
	declare(n)

	if p.lh.kind == '=' {
		p.match('=')
		n.ntype, err = p.parseType()
		if err != nil {
			return
		}
	}
	if _, err = p.match(':'); err != nil {
		return
	}

	rl, err := p.parseRHSList()
	if err != nil {
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
		err = cc.error(p.lh.pos, "expected terminal or nonterminal declaration, found %s", p.lh)
	}

	// error recovery, simply skip to the next semicolon (the end of a declaration)
	if err != nil {
		for p.lh.kind != ';' {
			p.next()
		}
	}

	p.match(';')

	return
}

func (p *Parser) parseGrammar() (n *Node, err error) {
	if _, err = p.match(GRAMMAR); err != nil {
		return
	}

	var s *Sym
	if s, err = p.parseTermName(); err != nil {
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
		var n2 *Node
		if n2, err = p.parseDecl(); err != nil {
			continue
		}
		l = l.add(n2)
	}

	n.rlist = l
	return
}

func (p *Parser) parse(f string) (l *Node) {
	// 1. Create a lexer to scan the file.
	var err error
	p.lexer, err = NewLexer(f)
	if err != nil {
		panic("could not open file")
	}
	cc.numSavedErrs = 0
	p.lh = p.lexer.Next()

	// 2. Parse the file, exiting if any errors were encountered
	if l, err = p.parseGrammar(); err != nil {
		return
	}

	// 3. Check to make sure all unresolved symbols have been resolved.
	if len(cc.unresolved) > 0 {
		for k, _ := range cc.unresolved {
			cc.error(k.pos, "undefined symbol %s\n", k)
		}
		return
	}

	return
}
