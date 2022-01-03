package css

import "fmt"

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

func (p *parser) errorf(t token, msg string, v ...interface{}) error {
	return &parseErr{fmt.Sprintf(msg, v...), t}
}

type pseudoClassSelector struct {
	ident    string
	function string
	args     []token
}

// https://www.w3.org/TR/selectors-4/#typedef-pseudo-class-selector
func (p *parser) pseudoClassSelector() (*pseudoClassSelector, error) {
	t, err := p.next()
	if err != nil {
		return nil, err
	}
	if t.typ != tokenColon {
		return nil, p.errorf(t, "expected ':'")
	}

	t, err = p.next()
	if err != nil {
		return nil, err
	}
	if t.typ == tokenIdent {
		return &pseudoClassSelector{ident: t.s}, nil
	}
	if t.typ != tokenFunction {
		return nil, p.errorf(t, "expected identifier or function")
	}

	args, err := p.any(tokenParenClose)
	if err != nil {
		return nil, err
	}

	c, err := p.next()
	if err != nil {
		return nil, err
	}
	if c.typ != tokenParenClose {
		return nil, p.errorf(t, "expected ')'")
	}
	return &pseudoClassSelector{function: t.s, args: args}, nil
}

// https://drafts.csswg.org/css-syntax-3/#typedef-any-value
func (p *parser) any(until tokenType) ([]token, error) {
	var (
		tokens      []token
		wantClosing []tokenType
	)
	for {
		if len(wantClosing) == 0 {
			t, err := p.peek()
			if err != nil {
				return nil, err
			}
			if t.typ == until {
				return tokens, nil
			}
		}

		t, err := p.next()
		if err != nil {
			return nil, err
		}
		switch t.typ {
		case tokenBracketOpen:
			wantClosing = append(wantClosing, tokenBracketClose)
		case tokenCurlyOpen:
			wantClosing = append(wantClosing, tokenCurlyClose)
		case tokenParenOpen:
			wantClosing = append(wantClosing, tokenParenClose)
		case tokenBracketClose, tokenCurlyClose, tokenParenClose:
			if len(wantClosing) == 0 || wantClosing[len(wantClosing)-1] != t.typ {
				return nil, p.errorf(t, "unmatched '%s'", t.s)
			}
			wantClosing = wantClosing[:len(wantClosing)-1]
		}
		tokens = append(tokens, t)
	}
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
		return nil, p.errorf(t, "expected '.'")
	}

	t, err = p.next()
	if err != nil {
		return nil, err
	}
	if t.typ != tokenIdent {
		return nil, p.errorf(t, "expect idententifier")
	}
	return &classSelector{t.s}, nil
}
