package zebu

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
)

type TokenKind int

// The list of tokens
const (
	EOF     = iota
	UNKNOWN = 0 - iota

	// Literal tokens
	NAME
	TERMINAL
	NONTERMINAL
	VARID
	CHARLIT
	STRLIT
	REGLIT
	NUMLIT

	// Keyword tokens
	GRAMMAR
	IMPORT
	KEYWORD
	EXTEND
	INHERIT
	OVERRIDE
	DELETE
	MODIFY
)

var tokenLabels = map[TokenKind]string{
	EOF: "eof",

	NAME:        "name",
	TERMINAL:    "terminal",
	NONTERMINAL: "nonterminal",
	VARID:       "varid",
	CHARLIT:     "charlit",
	STRLIT:      "strlit",
	REGLIT:      "reglit",
	NUMLIT:      "numlit",

	// Keyword tokens
	GRAMMAR:  "grammar",
	IMPORT:   "import",
	KEYWORD:  "keyword",
	EXTEND:   "extend",
	INHERIT:  "inherit",
	OVERRIDE: "override",
	DELETE:   "delete",
	MODIFY:   "modify",
}

func (k TokenKind) String() string {
	if int(k) < len(tokenLabels) {
		return tokenLabels[k]
	} else {
		return string(k)
	}
}

type Position struct {
	file string
	line int
	col  int
}

func (p *Position) String() string {
	return fmt.Sprintf("%s:%d:%d", p.file, p.line, p.col)
}

type Token struct {
	pos  *Position
	kind TokenKind

	// Simulating a union
	lit  *Strlit
	nval int
	byt  byte
	sym  *Sym
	code []byte
}

func (t *Token) String() string {
	switch t.kind {
	case NAME:
		return fmt.Sprintf("%s", t.sym)
	case REGLIT:
		return fmt.Sprintf("%c", t.byt)
	case STRLIT:
		return fmt.Sprintf("%s", t.lit)
	case VARID:
		return fmt.Sprintf("%s", t.sym)
	case NUMLIT:
		return fmt.Sprintf("%d", t.nval)
	default:
		return t.kind.String()
	}
}

func (t *Token) isSym() bool {
	switch t.kind {
	case NAME, TERMINAL, NONTERMINAL:
		return true
	}
	return false
}

type LexerMode int

const (
	Normal = iota
	Regex
)

type Lexer struct {
	// State based on current loaded file
	fileName string
	file     *os.File
	buf      *bufio.Reader

	mode LexerMode
	line int
	col  int

	// Current char peeked by the lexer
	ch  byte
	ch1 byte
}

func NewLexer(fileName string) (l *Lexer, err error) {
	var file *os.File = nil
	var buf *bufio.Reader = nil

	file, err = os.Open(fileName)
	if err != nil {
		return
	}

	buf = bufio.NewReader(file)

	err = nil
	l = &Lexer{
		fileName: fileName,
		file:     file,
		buf:      buf,
		mode:     Normal,
		line:     1,
		col:      0,
		ch:       0,
		ch1:      0,
	}

	return
}

func (l *Lexer) getc() {
	var err error
	switch l.ch1 {
	case 0:
		l.ch, err = l.buf.ReadByte()
		if err != nil {
			l.ch = 0
			return
		}
		if l.ch == '\n' {
			l.line++
			l.col = 0
		} else {
			l.col++
		}
		break

	default:
		l.ch = l.ch1
		l.ch1 = 0
	}
}

func (l *Lexer) putc(b byte) {
	l.ch1 = b
}

func isWhitespace(c byte) bool {
	switch c {
	case ' ', '\n', '\t', '\r':
		return true
	default:
		return false
	}
}

func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isAlphanum(c byte) bool {
	return isAlpha(c) || isNum(c) || c == '_'
}

func isNum(c byte) bool {
	return c >= '0' && c <= '9'
}

func isLower(c byte) bool {
	return c >= 'a' && c <= 'z'
}

func isUpper(c byte) bool {
	return c >= 'A' && c <= 'Z'
}

func isVarId(c byte) bool {
	return c == '$'
}

func regEscape(c byte) byte {
	return c
}

func (l *Lexer) strEscape() byte {
	if l.ch != '\\' {
		return l.ch
	}
	l.getc()
	switch l.ch {
	case 'n':
		return '\n'
	case '\\':
		return '\\'
	case '\'':
		return '\''
	default:
		panic(fmt.Sprintf("unknown escape character %c", l.ch))
	}
}

func (l *Lexer) Raw() byte {
	c := l.ch
	l.getc()
	return c
}

func (l *Lexer) Next() (t *Token) {
	t = new(Token)
	var cp int
	var ep int
	var lxbuf [512]byte
	var b byte

lex_whitespace:
	l.getc()
	if isWhitespace(l.ch) {
		goto lex_whitespace
	}

	t.pos = &Position{
		file: l.fileName,
		line: l.line,
		col:  l.col,
	}
	// Update compiler position for better diagnostics
	cc.pos = t.pos

	// Let's keep mode handling as the prefix to all lexing (minus whitespace)
	// Easier to jump state once mode is handled
	switch l.ch {
	case '[':
		l.mode = Regex
		goto lex_charlit
	case ']':
		l.mode = Normal
		goto lex_charlit
	case '^':
		goto lex_charlit
	case '-':
		goto lex_charlit
	}

lex_begin:
	if l.ch == 0 {
		t.kind = EOF
		goto lex_out
	}

	if l.mode == Regex {
		goto lex_regex
	}

	if isAlpha(l.ch) || l.ch == '$' {
		lxbuf[0] = l.ch
		cp = 1
		ep = 512
		goto lex_alpha
	}

	if isNum(l.ch) {
		lxbuf[0] = l.ch
		cp = 1
		ep = 512
		goto lex_num
	}

	switch l.ch {
	case '\'':
		goto lex_strlit

	case '/':
		l.getc()
		switch l.ch {
		case '/':
			l.getc()
			for l.ch != '\n' {
				l.getc()
			}
			goto lex_whitespace
		case '*':
			for {
				l.getc()
				if l.ch == 0 {
					goto lex_begin
				}
				if l.ch == '*' {
					l.getc()
					if l.ch == '/' {
						break
					}
				}
			}
			goto lex_whitespace

		default:
			l.putc(l.ch)
			l.ch = '/'
		}
	}

	// This is meant to be the default
lex_charlit:
	t.kind = TokenKind(l.ch)
	t.byt = l.ch
	goto lex_out

lex_alpha:
	for {
		if cp+10 >= ep {
			panic("identifier too long")
		}
		l.getc()
		if !isAlphanum(l.ch) {
			l.putc(l.ch)
			break
		}
		lxbuf[cp] = l.ch
		cp++
	}
	t.sym = cc.symbols.lookup(string(lxbuf[:cp]))
	t.kind = t.sym.lexical
	if t.kind == NAME {
		// Refine the kind to be more specific
		switch {
		case isLower(t.sym.name[0]):
			t.kind = TERMINAL
		case isUpper(t.sym.name[0]):
			t.kind = NONTERMINAL
		case isVarId(t.sym.name[0]):
			t.kind = VARID
		default:
			panic("non matching alpha sym")
		}
	}
	goto lex_out

lex_num:
	for {
		if cp+10 >= ep {
			panic("number too long")
		}
		l.getc()
		if !isNum(l.ch) {
			l.putc(l.ch)
			break
		}
		lxbuf[cp] = l.ch
		cp++
	}
	t.kind = NUMLIT
	t.nval, _ = strconv.Atoi(string(lxbuf[:cp]))
	goto lex_out

lex_regex:
	b = regEscape(l.ch)
	switch {
	}
	t.kind = REGLIT
	t.byt = b
	goto lex_out

lex_strlit:
	for {
		l.getc()
		if l.ch == '\'' {
			break
		}
		b := l.strEscape()
		lxbuf[cp] = b
		cp++
	}
	t.kind = STRLIT
	t.lit = cc.strlits.lookup(string(lxbuf[:cp]))

	goto lex_out

lex_out:
	return t
}
