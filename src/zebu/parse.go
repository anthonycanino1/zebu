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

func (p *Parser) check(k TokenKind) bool {
	if p.lh.kind != k {
		return false
	}
	return true
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
	n = &Node{
		op:  OCHAR,
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
	l := make([]*Node, 0)
	for {
		if p.lh.kind == ']' {
			break
		}
		var n1 *Node
		if n1, err = p.parseClassBodyRange(); err != nil {
			return
		}
		l = append(l, n1)
	}
	_, err = p.match(']')
	n = &Node{
		op:    OCLASS,
		nodes: l,
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

func (p *Parser) parseCat() (n *Node, err error) {
	n, err = p.parseKleene()
	if err != nil {
		return
	}
loop:
	for {
		switch p.lh.kind {
		case '(', '[', NONTERMINAL, STRLIT:
			var n2 *Node
			if n2, err = p.parseKleene(); err != nil {
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

func (p *Parser) parseAlt() (n *Node, err error) {
	if n, err = p.parseCat(); err != nil {
		return
	}
	for {
		if p.lh.kind != '|' {
			break
		}
		p.match('|')
		var n2 *Node
		if n2, err = p.parseCat(); err != nil {
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

func (p *Parser) parseRegdefHead() (n *Node, err error) {
	s, err := p.parseNontermName()
	if err != nil {
		return
	}
	n = newname(s)
	return
}

func (p *Parser) parseRegdef() (n *Node, err error) {
	n, err = p.parseRegdefHead()
	if err != nil {
		return
	}
	n.op = OREGDEF
	declare(n)

	_, err = p.match(':')
	if err != nil {
		return
	}
	n.left, err = p.parseAlt()
	if err != nil {
		return
	}
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

func (p *Parser) parseVarId() (n *Node, err error) {
	t, err := p.match(VARID)
	if err != nil {
		return
	}
	n = newname(t.sym)
	n.op = OVARID
	declare(n)
	return
}

func (p *Parser) parseEpsilon() (n *Node, err error) {
	n = nepsilon
	return
}

func (p *Parser) parseStrlit() (n *Node, err error) {
	t, err := p.match(STRLIT)
	if err != nil {
		return
	}
	n = strlitNode(t.lit)
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

func (p *Parser) parseProdElem() (n *Node, err error) {
	var nn *Node
	switch p.lh.kind {
	case TERMINAL:
		nn, err = p.parseTerm()
	case NONTERMINAL:
		nn, err = p.parseNonterm()
	case STRLIT:
		nn, err = p.parseStrlit()
	case '|', '{', ';':
		nn, err = p.parseEpsilon()
	default:
		err = cc.error(p.lh.pos, "unexpected %s.", p.lh)
	}
	if err != nil {
		return
	}
	n = &Node{
		op:   OPRODELEM,
		left: nn,
	}
	if p.lh.kind == '=' {
		p.match('=')
		n.right, err = p.parseVarId()
		if err != nil {
			return
		}
		n.right.ntype = n
		pushvarid(n.right.sym)
	}
	if p.lh.kind == '{' {
		n.action, err = p.parseAction()
	}
	return
}

func (p *Parser) parseProd() (n *Node, err error) {
	l := make([]*Node, 0)
	defer popvarids()
	for {
		var nn *Node
		if nn, err = p.parseProdElem(); err != nil {
			return
		}
		l = append(l, nn)
		if p.lh.kind == '|' || p.lh.kind == ';' {
			break
		}
	}
	n = &Node{
		op:    OPROD,
		nodes: l,
	}
	return
}

func (p *Parser) parseRuleBody() (l []*Node, err error) {
	l = make([]*Node, 0)
	for {
		var n *Node
		if n, err = p.parseProd(); err != nil {
			return
		}
		l = append(l, n)
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

func (p *Parser) parseRuleHead() (n *Node, err error) {
	s, err := p.parseTermName()
	if err != nil {
		return
	}
	n = newname(s)
	return
}

func (p *Parser) parseRule() (n *Node, err error) {
	if n, err = p.parseRuleHead(); err != nil {
		return
	}
	n.op = ORULE
	declare(n)

	if p.lh.kind == '=' {
		// TODO : Somewhat hacky to lex/parse right now, clean up
		// later
		p.check('=') 
		p.lexer.Raw()
		n.ntype, err = p.parseType()
		if err != nil {
			return
		}
	}
	if _, err = p.match(':'); err != nil {
		return
	}

	n.nodes, err = p.parseRuleBody()
	if err != nil {
		return
	}
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

	n = newname(s)
	n.op = OGRAM
	n.nodes = make([]*Node, 0)
	declare(n)

	p.match(';')

	for p.lh.kind != EOF {
		var n2 *Node
		if n2, err = p.parseDecl(); err != nil {
			continue
		}
		n.nodes = append(n.nodes, n2)

		// Save start
		if n2.op == ORULE && n2.sym.name == "start" {
			n.left = n2
		}
	}

	return
}

func (p *Parser) parse(f string) (n *Node) {
	// 1. Create a lexer to scan the file.
	var err error
	p.lexer, err = NewLexer(f)
	if err != nil {
		panic("could not open file")
	}
	cc.numSavedErrs = 0
	p.lh = p.lexer.Next()

	// 2. Parse the file, exiting if any errors were encountered
	if n, err = p.parseGrammar(); err != nil {
		return
	}

	// 3. Check to make sure we have a start rule
	if n.left == nil {
		cc.error(n.pos, "grammar must define a start rule.")
	}

	return
}
