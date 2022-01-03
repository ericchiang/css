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
	// peekQueue holds tokens that have been peeked but not consumed. These are
	// consumed before the lexer is consulted.
	peekQueue *queue
	// err is set whenever a lex error occurs. When set, all subsequent calls to
	// next(), peek(), and peekN() will fail.
	err error
}

func newParser(s string) *parser {
	return &parser{l: newLexer(s), peekQueue: newQueue(2)}
}

func (p *parser) peek() (token, error) {
	return p.peekN(0)
}

func (p *parser) peekN(n int) (token, error) {
	if p.err != nil {
		return token{}, p.err
	}
	for n >= p.peekQueue.len() {
		t, err := p.l.next()
		if err != nil {
			p.err = err
			return token{}, err
		}
		p.peekQueue.push(t)
	}
	return p.peekQueue.get(n), nil
}

func (p *parser) next() (token, error) {
	if p.err != nil {
		return token{}, p.err
	}
	if p.peekQueue.len() > 0 {
		return p.peekQueue.pop(), nil
	}
	t, err := p.l.next()
	if err != nil {
		p.err = err
		return t, err
	}
	return t, nil
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
	if !t.isDelim(".") {
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

func (p *parser) skipWhitespace() {
	for {
		t, err := p.peek()
		if err != nil || t.typ != tokenWhitespace {
			return
		}
		p.next()
	}
}

type wqName struct {
	hasPrefix bool
	prefix    string
	value     string
}

// <wq-name> = <ns-prefix>? <ident-token>
// <ns-prefix> = [ <ident-token> | '*' ]? '|'
//
// https://www.w3.org/TR/selectors-4/#typedef-wq-name
func (p *parser) wqName() (*wqName, error) {
	t, err := p.next()
	if err != nil {
		return nil, err
	}
	if t.isDelim("|") {
		t, err := p.next()
		if err != nil {
			return nil, err
		}
		if t.typ != tokenIdent {
			return nil, p.errorf(t, "expected identifier")
		}
		return &wqName{true, "", t.s}, nil
	}
	if t.isDelim("*") {
		delim, err := p.next()
		if err != nil {
			return nil, err
		}
		if !delim.isDelim("|") {
			return nil, p.errorf(delim, "expected '|'")
		}

		ident, err := p.next()
		if err != nil {
			return nil, err
		}
		if ident.typ != tokenIdent {
			return nil, p.errorf(ident, "expected identifier")
		}
		return &wqName{true, t.s, ident.s}, nil
	}
	if t.typ != tokenIdent {
		return nil, p.errorf(t, "expected identifier")
	}

	// See if the stream contains '|' <ident-token>.
	delim, err := p.peek()
	if err != nil {
		return nil, err
	}
	if !delim.isDelim("|") {
		return &wqName{false, "", t.s}, nil
	}
	ident, err := p.peekN(1)
	if err != nil {
		return nil, err
	}
	if ident.typ != tokenIdent {
		return &wqName{false, "", t.s}, nil
	}
	// Consume peeked tokens.
	p.next()
	p.next()
	return &wqName{true, t.s, ident.s}, nil
}
