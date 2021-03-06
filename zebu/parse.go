package zebu

import (
	"fmt"
	"go/ast"
	"go/parser"
)

// Debug, erase when done
var _ = fmt.Printf

type Parser struct {
	lexer *Lexer
	lh    *Token
}

func consParser() (p *Parser) {
	p = &Parser{
		lexer: nil,
		lh:    nil,
	}
	return
}

func (p *Parser) next() (t *Token) {
	t = p.lh
	p.lh = p.lexer.next()
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
		err = compileError(p.lh.pos, "expected %s", k)
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
			err = compileError(p.lh.pos, "undeclared regular definition!")
			return
		}
		n = s.defn
	case STRLIT:
		n, err = p.parseStrlit()
	default:
		err = compileError(p.lh.pos, "expected regular definition or string literal, found %s", p.lh)
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

// TODO : Refactor the RuleHead/RegDef Head, can move declare
func (p *Parser) parseRegdef() (n *Node, err error) {
	n, err = p.parseRegdefHead()
	if err != nil {
		return
	}
	n.op = OREGDEF
	declare(n)

	if p.lh.kind == '=' {
		// TODO : Somewhat hacky to lex/parse right now, clean up
		// later
		lpos := p.lh.pos
		p.check('=')
		p.lexer.raw()
		if n.ntype, err = p.parseType(); err != nil {
			err = reerror(lpos, err.(*CCError))
			return
		}
	}

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
	codebuf := make([]byte, 0, 512)
	dpn := make([]*Node, 0)
	for {
		c := p.lexer.raw()
		if c == 0 {
			exit(1)
		}
		if c == '}' && lvl == 0 {
			break
		}
		switch c {
		case '{':
			lvl++
		case '}':
			lvl--
		case '$':
			buf := []byte{'$'}
			c1 := p.lexer.raw()
			for isVarIdChar(c1) {
				buf = append(buf, c1)
				c1 = p.lexer.raw()
			}
			p.lexer.putc(c1)
			s := symbols.lookup(string(buf))
			if s.defn == nil {
				if s.name != "$$" {
					compileError(zbpos, "undefined variable id %s", s)
				}
			} else {
				s.defn.used = true
				dpn = append(dpn, s.defn)
			}
			codebuf = append(codebuf, buf...)
			continue
		}
		codebuf = append(codebuf, c)
	}
	// Reset the lh token
	p.next()

	// TODO: Lots of tricky logic here. Eventually move.
	// (1) Create an (OACTION) to house the actual action
	// (2) Create a new rule (ORULE) for this action,
	// (3) Create a new epsilon production for this rule that takes the actual action.
	// (4) Return a reference to the ORULE so it gets placed in an OPRODDCL
	// (5) Insert the new ORULE into the grammar
	s := symbols.lookup(fmt.Sprintf("'A%d_%s", nextaction, currule.sym))
	nextaction++

	action := &Node{
		op:   OACTION,
		code: codebuf,
		sym: s,
	} 

	n = newname(s)
	n.op = ORULE
	n.nodes = []*Node{&Node{
		op: OPROD,
		nodes: []*Node{&Node{
			op:    OPRODDCL,
			left:  nepsilon,
			right: action,
		}},
	}}
	declare(n)
	curgram.nodes = append(curgram.nodes, n)

	return
}

func (p *Parser) parseVarId() (s *Sym, err error) {
	t, err := p.match(VARID)
	if err != nil {
		return
	}
	s = t.sym
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

// LAST : Transform the parse of an Action into a "nonterm" rule
func (p *Parser) parseProdElem() (n *Node, err error) {
	var n2 *Node
	switch p.lh.kind {
	case TERMINAL:
		n2, err = p.parseTerm()
	case NONTERMINAL:
		n2, err = p.parseNonterm()
	case STRLIT:
		n2, err = p.parseStrlit()
	case '|', ';':
		n2, err = p.parseEpsilon()
	case '{':
		n2, err = p.parseAction()
	default:
		err = compileError(p.lh.pos, "unexpected %s.", p.lh)
	}
	if err != nil {
		return
	}
	n = &Node{
		op:   OPRODDCL,
		left: n2,
	}
	if p.lh.kind == '=' {
		p.match('=')
		n.sym, err = p.parseVarId()
		if err != nil {
			return
		}
		declare(n)
		pushvarid(n.sym)
	} else {
		n.sym = pushnextvarid()
		declare(n)
	}

	return
}

func (p *Parser) parseProd() (n *Node, err error) {
	l := make([]*Node, 0)
	nextvarid = 1
	defer popvarids()
	for {
		var n2 *Node
		if n2, err = p.parseProdElem(); err != nil {
			return
		}
		l = append(l, n2)
		if p.lh.kind == '|' || p.lh.kind == ';' {
			break
		}
		nextvarid++
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

// TODO : This one is big, this is very ugly. Right now this is
// a mini lexer inside the parser. This should be swapped out for
// actual type checking, possibly using some of the APIs go provides.
func (p *Parser) parseType() (n *Node, err error) {
	typ := make([]byte, 0, 10)
	var c, c1 byte
	for {
	whitespace:
		c = p.lexer.raw()
		if c == ':' {
			break
		}
		if isWhitespace(c) {
			goto whitespace
		}
		if c == '/' {
			c1 = p.lexer.raw()
			switch c1 {
			case '/':
				for c1 != '\n' {
					c1 = p.lexer.raw()
				}
				goto whitespace
			case '*':
				c1 = p.lexer.raw()
				for {
					if c1 == '*' {
						c1 = p.lexer.raw()
						if c1 == '/' {
							break
						}
					}
				}
				goto whitespace
			default:
				p.lexer.putc(c1)
			}
		}
		typ = append(typ, c)
	}

	if len(typ) == 0 {
		err = consCCError(nil, "expected type")
		return
	}

	p.lexer.putc(':')
	p.next()

	// Check type table to intern the ast.Expr
	typestr := string(typ[:])
	if etype, ok := types.lookup(typestr); ok {
		n = &Node{
			op:    OTYPE,
			typ:   typestr,
			etype: etype,
		}
		return
	}

	var etype ast.Expr
	if etype, err = parser.ParseExpr(typestr); err != nil {
		err = consCCError(nil, "invalid type declaration %s", typestr)
		return
	}

	switch etype.(type) {
	case *ast.ArrayType, *ast.ChanType, *ast.FuncType, *ast.Ident, *ast.InterfaceType, *ast.MapType, *ast.SelectorExpr, *ast.StructType:
		// TODO : Place holder
		break
	default:
		err = consCCError(nil, "invalid type declaration %s", typestr)
		return
	}

	types.insert(typestr, etype)

	n = &Node{
		op:    OTYPE,
		typ:   typestr,
		etype: etype,
	}

	return
}

// TODO : Refactor the RuleHead/RegDef Head, can move declare
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
	currule = n
	nextaction = 0

	if p.lh.kind == '=' {
		// TODO : Somewhat hacky to lex/parse right now, clean up
		// later
		lpos := p.lh.pos
		p.check('=')
		p.lexer.raw()
		if n.ntype, err = p.parseType(); err != nil {
			err = reerror(lpos, err.(*CCError))
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
		err = compileError(p.lh.pos, "expected terminal or nonterminal declaration, found %s", p.lh)
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

func (p *Parser) parseEscapeCode() (code []byte, err error) {
	if !p.check(ESCOPEN) {
		return
	}
	p.lexer.raw()
	for {
		c := p.lexer.raw()
		if c == '@' {
			c2 := p.lexer.raw()
			if c2 == '}' {
				break
			}
			p.lexer.putc(c2)
		}
		code = append(code, c)
	}
	// reset the lh token
	p.next()
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
	curgram = n
	declare(n)

	p.match(';')

	if p.lh.kind == ESCOPEN {
		if n.code, err = p.parseEscapeCode(); err != nil {
			return
		}
	}

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
	p.lexer, err = consLexer(f)
	if err != nil {
		panic("could not open file")
	}
	numSavedErrs = 0
	p.lh = p.lexer.next()

	// 2. Parse the file, exiting if any errors were encountered
	if n, err = p.parseGrammar(); err != nil {
		return
	}

	// 3. Check to make sure we have a start rule
	if n.left == nil {
		compileError(n.pos, "grammar must define a start rule.")
	}

	return
}
