package css

import "errors"

var eofErr = errors.New("eof")

type parser struct {
	l *lexer

	peeked bool
	token  token
	err    error
}

func (p *parser) peekIs(typ tokenType) bool {
	t, err := p.peek()
	return err == nil && t.typ == typ
}

func (p *parser) peek() (token, error) {
	if !p.peeked {
		p.token, p.err = p.l.next()
	}
	return p.token, p.err
}

func (p *parser) pop() (token, error) {
	t, err := p.peek()
	p.peeked = false
	return t, err
}

type selector struct {
	complexSelectors []*complexSelector
}

func (p *parser) skipWhitespace() {
	for {
		t, err := p.peek()
		if err != nil {
			return
		}
		if t.typ != tokenWhitespace {
			return
		}
		p.pop()
	}
}

func (p *parser) selector() (*selector, error) {
	s := &selector{}
	for {
		p.skipWhitespace()
		if p.peekIs(tokenEOF) {
			return s, nil
		}
		cs, err := p.complexSelector()
		if err != nil {
			return nil, err
		}
		s.complexSelectors = append(s.complexSelectors, cs)
	}
}

type complexSelector struct {
	compoundSelector *compoundSelector

	combinator *combinator
	next       *complexSelector
}

func (p *parser) complexSelector() (*complexSelector, error) {
	return nil, nil
}

type compoundSelector struct {
}

type combinator struct {
}
