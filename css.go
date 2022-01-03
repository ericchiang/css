package css

type parseErr struct {
	msg string
	t   token
}

func (p *parseErr) Error() string {
	return p.msg
}

type parser struct {
	l *lexer

	peeked bool
	t      token
	err    error
}

func newParser(s string) *parser {
	return &parser{l: newLexer(s)}
}

func (p *parser) peek() (token, error) {
	if p.peeked {
		return p.t, p.err
	}
	p.peeked = true
	p.t, p.err = p.l.next()
	return p.t, p.err
}

func (p *parser) next() (token, error) {
	if p.peeked {
		p.peeked = false
		return p.t, p.err
	}
	return p.l.next()
}

func (p *parser) error(t token, msg string) error {
	return &parseErr{msg, t}
}

type classSelector struct {
	class string
}

// https://www.w3.org/TR/selectors-4/#typedef-class-selector
func (p *parser) classSelector() (*classSelector, error) {
	t, err := p.next()
	if err != nil {
		return nil, err
	}
	if !(t.typ == tokenDelim && t.s == ".") {
		return nil, p.error(t, "expected '.'")
	}

	t, err = p.next()
	if err != nil {
		return nil, err
	}
	if t.typ != tokenIdent {
		return nil, p.error(t, "expect idententifier")
	}
	return &classSelector{t.s}, nil
}
