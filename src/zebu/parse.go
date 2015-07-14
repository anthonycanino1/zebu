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

func (p *Parser) parseLHS() (n *Node, err error) {
	s, _ := p.parseTermName()
	if s == nil {
		return
	}
	n = newname(s)
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

func (p *Parser) parseNonterm() (n *Node, err error) {
	s, _ := p.parseNontermName()
	if s == nil {
		return
	}
	n = oldname(s)
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

// LAST : Was working on parsing symvars
func (p *Parser) parseRuleDcl() (n *Node, err error) {
	var l *Node
	switch p.lh.kind {
	case TERMINAL:
		l, _ = p.parseTerm()

	case NONTERMINAL:
		l, _ = p.parseNonterm()

	case STRLIT:
		l, _ = p.parseStrlit()
	}
	if l == nil {
		return
	}
	n = &Node{
		op:   ORDCL,
		left: l,
	}
	if p.lh.kind == '=' {
		p.match('=')
		n.svar, _ = p.parseVarId()
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
	return
}

func (p *Parser) parseRule() (n *Node, err error) {
	if n, _ = p.parseLHS(); n == nil {
		return
	}
	n.op = ORULE
	declare(n)

	if p.lh.kind == '=' {
		// type decl
		p.match('=')
		n.typ, _ = p.parseType()
	}
	p.match(':')

	rl, _ := p.parseRHSList()
	if rl == nil {
		return
	}

	n.rlist = rl
	return
}

func (p *Parser) parseRegex() (n *Node, err error) {
	return
}

func (p *Parser) parseDecl() (n *Node, err error) {
	switch p.lh.kind {
	case TERMINAL:
		n, err = p.parseRule()
		break
	case NONTERMINAL:
		n, err = p.parseRegex()
		break
	default:
		zbError(p.lh.pos, "expected terminal or nonterminal declaration, found %s", p.lh)
		break
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

	n = NewNode(OGRAM, nil, nil)
	n.sym = s
	//declare(n)

	p.match(';')

	l := new(NodeList)
	for p.lh.kind != EOF {
		n2, _ := p.parseDecl()
		if n2 == nil {
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
