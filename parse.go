package css

import (
	"fmt"
	"strconv"
	"strings"
)

type parseErr struct {
	msg string
	t   token
}

func (p *parseErr) Error() string {
	return fmt.Sprintf("consuming %s: %s", p.t, p.msg)
}

type parser struct {
	l interface {
		next() (token, error)
	}
	// peekQueue holds tokens that have been peeked but not consumed. These are
	// consumed before the lexer is consulted.
	peekQueue *queue
	// err is set whenever a lex error occurs. When set, all subsequent calls to
	// next(), peek(), and peekN() will fail.
	err error
}

type tokens struct {
	i int
	t []token
}

func (t *tokens) next() (token, error) {
	if t.i < len(t.t) {
		tok := t.t[t.i]
		t.i++
		return tok, nil
	}
	lastPos := 0
	if len(t.t) > 0 {
		lastTok := t.t[len(t.t)-1]
		lastPos = lastTok.pos + len(lastTok.raw)
	}
	return token{tokenEOF, "", "", lastPos, 0, ""}, nil
}

// newParserFromTokens allows creating a parser from a token stream. This is
// used for subparsers, such as pseudo-elements.
func newParserFromTokens(t []token) *parser {
	return &parser{l: &tokens{t: t}, peekQueue: newQueue(2)}
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

func (p *parser) parse() ([]complexSelector, error) {
	var sels []complexSelector
	p.skipWhitespace()
	for {
		cs, err := p.complexSelector()
		if err != nil {
			return nil, err
		}
		sels = append(sels, *cs)
		p.skipWhitespace()
		t, err := p.next()
		if err != nil {
			return nil, err
		}
		if t.typ == tokenEOF {
			return sels, nil
		}
		if t.typ != tokenComma {
			return nil, p.errorf(t, "expected ',' or EOF")
		}
		p.skipWhitespace()
	}
}

type complexSelector struct {
	pos        int
	sel        compoundSelector
	combinator string
	next       *complexSelector
}

func (p *parser) complexSelector() (*complexSelector, error) {
	t, err := p.peek() // peek the first token for creating errors.
	if err != nil {
		return nil, err
	}

	sel := &complexSelector{pos: t.pos}
	cs, ok, err := p.compoundSelector()
	if err != nil {
		return nil, err
	}
	if !ok {
		//  <compound-selector> can start with:
		//  |-- <type-selector>
		//  | \-- <ns-prefix>? [ '*' | <ident-token> ]
		//  |   \-- [ <ident-token> | '*' ]? '|'
		//  |-- <subclass-selector>
		//  | |-- <id-selector> = <hash-token>
		//  | |-- <class-selector> = '.' <ident-token>
		//  | |-- <attribute-selector> = '[' ...
		//  | \-- <pseudo-class-selector> = ':' ...
		//  \-- <pseudo-element-selector> = ':' ...
		return nil, p.errorf(t, "expected identifier, '#', '*', '.', '|', '[', ':'")
	}
	sel.sel = *cs

	last := sel
	for {
		p.skipWhitespace()
		t, err = p.peek()
		if err != nil {
			return nil, err
		}
		if t.typ == tokenDelim {
			switch t.s {
			case ">", "+", "~":
				p.next()
				p.skipWhitespace()
				last.combinator = t.s
				if t, err = p.peek(); err != nil {
					return nil, err
				}
			case "|":
				t, err = p.peekN(1)
				if err != nil {
					return nil, err
				}
				if t.isDelim("|") {
					p.next()
					p.next()
					p.skipWhitespace()
					last.combinator = "||"
					if t, err = p.peek(); err != nil {
						return nil, err
					}
				}
			}
		}
		s, ok, err := p.compoundSelector()
		if err != nil {
			return nil, err
		}
		if !ok {
			if last.combinator != "" {
				return nil, p.errorf(t, "expected identifier, '#', '*', '.', '|', '[', ':'")
			}
			return sel, nil
		}
		next := &complexSelector{pos: s.pos, sel: *s}
		last.next = next
		last = next
	}
}

type compoundSelector struct {
	pos             int
	typeSelector    *typeSelector // may be nil
	subClasses      []subclassSelector
	pseudoSelectors []pseudoSelector
}

// <compound-selector> = [ <type-selector>? <subclass-selector>*
//                         [ <pseudo-element-selector> <pseudo-class-selector>* ]* ]!
//
// Whitespace is disallowed between top level elements.
func (p *parser) compoundSelector() (*compoundSelector, bool, error) {
	t, err := p.peek()
	if err != nil {
		return nil, false, err
	}
	found := false
	cs := &compoundSelector{pos: t.pos}
	ts, ok, err := p.typeSelector()
	if err != nil {
		return nil, false, err
	}
	if ok {
		found = true
		cs.typeSelector = ts
	}
	for {
		sc, ok, err := p.subclassSelector()
		if err != nil {
			return nil, false, err
		}
		if !ok {
			break
		}
		found = true
		cs.subClasses = append(cs.subClasses, *sc)
	}
	for {
		ps, ok, err := p.pseudoSelector()
		if err != nil {
			return nil, false, err
		}
		if !ok {
			break
		}
		found = true
		cs.pseudoSelectors = append(cs.pseudoSelectors, *ps)
	}
	if !found {
		return nil, false, nil
	}
	return cs, true, nil
}

type pseudoSelector struct {
	element pseudoClassSelector
	classes []pseudoClassSelector
}

// Implements a subset of the <compound-selector> logic.
//
// <pseudo-element-selector> <pseudo-class-selector>*
func (p *parser) pseudoSelector() (*pseudoSelector, bool, error) {
	t, err := p.peek()
	if err != nil {
		return nil, false, err
	}
	if t.typ != tokenColon {
		return nil, false, nil
	}
	t, err = p.peekN(1)
	if err != nil {
		return nil, false, err
	}
	if t.typ != tokenColon {
		return nil, false, nil
	}
	p.next()

	ele, err := p.pseudoClassSelector()
	if err != nil {
		return nil, false, err
	}
	ps := &pseudoSelector{element: *ele}
	for {
		p.skipWhitespace()
		t, err := p.peek()
		if err != nil {
			return nil, false, err
		}
		if t.typ != tokenColon {
			return ps, true, nil
		}
		cs, err := p.pseudoClassSelector()
		if err != nil {
			return nil, false, err
		}
		ps.classes = append(ps.classes, *cs)
	}
}

type typeSelector struct {
	pos       int
	hasPrefix bool
	prefix    string
	value     string
}

// <type-selector> = <wq-name> | <ns-prefix>? '*'
// <wq-name> = <ns-prefix>? <ident-token>
// <ns-prefix> = [ <ident-token> | '*' ]? '|'
//
// Whitespace is disallowed.
func (p *parser) typeSelector() (*typeSelector, bool, error) {
	t, err := p.peek()
	if err != nil {
		return nil, false, err
	}
	if !(t.typ == tokenIdent || t.isDelim("*") || t.isDelim("|")) {
		return nil, false, nil
	}

	name, err := p.parseName(true)
	if err != nil {
		return nil, false, err
	}
	return &typeSelector{
		pos:       t.pos,
		hasPrefix: name.hasPrefix,
		prefix:    name.prefix,
		value:     name.value,
	}, true, nil
}

type subclassSelector struct {
	pos                 int
	idSelector          string
	classSelector       string
	attributeSelector   *attributeSelector
	pseudoClassSelector *pseudoClassSelector
}

// <subclass-selector> = <id-selector> | <class-selector> |
//                       <attribute-selector> | <pseudo-class-selector>
// https://www.w3.org/TR/selectors-4/#typedef-subclass-selector
func (p *parser) subclassSelector() (*subclassSelector, bool, error) {
	t, err := p.peek()
	if err != nil {
		return nil, false, err
	}
	ss := &subclassSelector{pos: t.pos}
	// <id-selector> = <hash-token>
	if t.typ == tokenHash {
		p.next()
		ss.idSelector = strings.TrimPrefix(t.s, "#")
		return ss, true, nil
	}

	// <class-selector> = '.' <ident-token>
	if t.isDelim(".") {
		p.next()
		t, err := p.next()
		if err != nil {
			return nil, false, err
		}
		if t.typ != tokenIdent {
			return nil, false, p.errorf(t, "expected identifier")
		}
		ss.classSelector = strings.TrimPrefix(t.s, ".")
		return ss, true, nil
	}

	// <attribute-selector> = '[' <wq-name> ']' | ...
	if t.typ == tokenBracketOpen {
		a, err := p.attributeSelector()
		if err != nil {
			return nil, false, err
		}
		ss.attributeSelector = a
		return ss, true, nil
	}

	if t.typ != tokenColon {
		return nil, false, nil
	}

	// Maybe a <pseudo-class-selector>? When parsing <subclass-selector> we could
	// potentially match a <pseudo-element-selector> instead. So if the next
	// token is ':', assume we've hit a <pseudo-element-selector> and stop.
	//
	// <compound-selector> = [ <type-selector>? <subclass-selector>*
	//                       [ <pseudo-element-selector> <pseudo-class-selector>* ]* ]!

	pt, err := p.peekN(1)
	if err != nil {
		return nil, false, err
	}
	if pt.typ == tokenColon {
		// Found a <pseudo-element-selector>.
		return nil, false, nil
	}
	pcs, err := p.pseudoClassSelector()
	if err != nil {
		return nil, false, err
	}
	ss.pseudoClassSelector = pcs
	return ss, true, nil
}

type pseudoClassSelector struct {
	pos      int
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
	pos := t.pos
	if t.typ != tokenColon {
		return nil, p.errorf(t, "expected ':'")
	}

	t, err = p.next()
	if err != nil {
		return nil, err
	}
	if t.typ == tokenIdent {
		return &pseudoClassSelector{pos: pos, ident: t.s}, nil
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
	return &pseudoClassSelector{pos: pos, function: t.s, args: args}, nil
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
		case tokenEOF:
			return nil, p.errorf(t, "unexpected eof attempting to match '%s'", until)
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

func (p *parser) skipWhitespace() bool {
	seen := false
	for {
		t, err := p.peek()
		if err != nil || t.typ != tokenWhitespace {
			return seen
		}
		seen = true
		p.next()
	}
}

func (p *parser) expectWhitespaceOrEOF() error {
	p.skipWhitespace()
	t, err := p.next()
	if err != nil {
		return err
	}
	if t.typ != tokenEOF {
		return p.errorf(t, "expected no more tokens")
	}
	return nil
}

// <attribute-selector> = '[' <wq-name> ']' |
//                        '[' <wq-name> <attr-matcher> [ <string-token> | <ident-token> ] <attr-modifier>? ']'
// <attr-matcher> = [ '~' | '|' | '^' | '$' | '*' ]? '='
// <attr-modifier> = i
// <wq-name> = <ns-prefix>? <ident-token>
// <ns-prefix> = [ <ident-token> | '*' ]? '|'
//
// https://www.w3.org/TR/selectors-4/#typedef-attribute-selector
type attributeSelector struct {
	pos      int
	wqName   *wqName
	matcher  string
	val      string
	modifier bool
}

func (p *parser) attributeSelector() (*attributeSelector, error) {
	// '['
	t, err := p.next()
	if err != nil {
		return nil, err
	}
	if t.typ != tokenBracketOpen {
		return nil, p.errorf(t, "expected '['")
	}
	at := &attributeSelector{pos: t.pos}
	p.skipWhitespace()

	// <wq-name>
	name, err := p.wqName()
	if err != nil {
		return nil, err
	}
	at.wqName = name
	p.skipWhitespace()

	t, err = p.next()
	if err != nil {
		return nil, err
	}
	if t.typ == tokenBracketClose {
		// Found ']', we're done.
		return at, nil
	}

	// <attr-matcher> = [ '~' | '|' | '^' | '$' | '*' ]? '='
	if t.typ != tokenDelim {
		return nil, p.errorf(t, "expected '~', '|', '^', '$', '*' or '='")
	}
	switch t.s {
	case "~", "|", "^", "$", "*", "=":
	default:
		return nil, p.errorf(t, "expected '~', '|', '^', '$', '*' or '='")
	}
	at.matcher = "="
	if t.s != "=" {
		// https://www.w3.org/TR/selectors-4/#white-space
		//
		// Whitespace is forbidden between elements of the <attr-matcher>.

		at.matcher = t.s + "="
		t, err = p.next()
		if err != nil {
			return nil, err
		}
		if !t.isDelim("=") {
			return nil, p.errorf(t, "expected '='")
		}
	}
	p.skipWhitespace()

	// [ <string-token> | <ident-token> ]
	strOrIdent, err := p.next()
	if err != nil {
		return nil, err
	}
	if !(strOrIdent.typ == tokenString || strOrIdent.typ == tokenIdent) {
		return nil, p.errorf(strOrIdent, "expected identifier or string")
	}
	at.val = strOrIdent.s

	p.skipWhitespace()

	// <attr-modifier>?
	t, err = p.next()
	if err != nil {
		return nil, err
	}
	if t.s == "i" {
		at.modifier = true
		p.skipWhitespace()

		t, err = p.next()
		if err != nil {
			return nil, err
		}
	}
	if t.typ != tokenBracketClose {
		return nil, p.errorf(t, "expected ']'")
	}
	return at, nil
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
	return p.parseName(false)
}

// parseName handles either <wq-name> or <type-selector>, which are almost
// identical. However <type-selector> allows '*' as the final element.
//
// <wq-name>       = <ns-prefix>? <ident-token>
// <type-selector> = <ns-prefix>? [ <ident-token> | '*' ]
//
// https://www.w3.org/TR/selectors-4/#typedef-wq-name
// https://www.w3.org/TR/selectors-4/#typedef-type-selector
func (p *parser) parseName(allowStar bool) (*wqName, error) {
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
		delim, err := p.peek()
		if err != nil {
			return nil, err
		}
		if !delim.isDelim("|") {
			if allowStar {
				return &wqName{false, "", "*"}, nil
			}
			return nil, p.errorf(delim, "expected '|'")
		}

		// Consume the "|" delim.
		p.next()

		ident, err := p.next()
		if err != nil {
			return nil, err
		}
		if !(ident.typ == tokenIdent || (allowStar && ident.isDelim("*"))) {
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
	if !(ident.typ == tokenIdent || (allowStar && ident.isDelim("*"))) {
		return &wqName{false, "", t.s}, nil
	}
	// Consume peeked tokens.
	p.next()
	p.next()
	return &wqName{true, t.s, ident.s}, nil
}

// https://drafts.csswg.org/css-syntax-3/#typedef-n-dimension
func isNDimension(t token) bool {
	return t.typ == tokenDimension && t.flag == tokenFlagInteger && t.dim == "n"
}

// https://drafts.csswg.org/css-syntax-3/#typedef-ndash-dimension
func isNDashDimension(t token) bool {
	return t.typ == tokenDimension && t.dim == "n-"
}

func isPrefixWithDigits(s, prefix string) bool {
	if !strings.HasPrefix(s, prefix) {
		return false
	}
	if len(s) == len(prefix) {
		return false
	}
	for _, c := range s[len(prefix):] {
		if !isDigit(c) {
			return false
		}
	}
	return true
}

// isNDashDigitDimension looks for patterns like "n-13213".
//
// https://drafts.csswg.org/css-syntax-3/#typedef-ndashdigit-dimension
func isNDashDigitDimension(t token) bool {
	return t.typ == tokenDimension && isPrefixWithDigits(t.dim, "n-")
}

// https://drafts.csswg.org/css-syntax-3/#typedef-ndashdigit-ident
func isNDashDigitIdent(t token) bool {
	return t.typ == tokenIdent && isPrefixWithDigits(t.s, "n-")
}

// https://drafts.csswg.org/css-syntax-3/#typedef-dashndashdigit-ident
func isDashNDashDigitIdent(t token) bool {
	return t.typ == tokenIdent && isPrefixWithDigits(t.s, "-n-")
}

// https://drafts.csswg.org/css-syntax-3/#typedef-integer
func isInteger(t token) bool {
	return t.typ == tokenNumber && t.flag == tokenFlagInteger
}

// https://drafts.csswg.org/css-syntax-3/#typedef-signed-integer
func isSignedInteger(t token) bool {
	return isInteger(t) && (strings.HasPrefix(t.s, "+") || strings.HasPrefix(t.s, "-"))
}

// https://drafts.csswg.org/css-syntax-3/#typedef-signless-integer
func isSignlessInteger(t token) bool {
	return isInteger(t) && strings.IndexFunc(t.s, isDigit) == 0
}

func parseInt(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// b parses the common pattern of <signed-integer> | ['+' | '-'] <signless-integer>
func (p *parser) b() (int64, error) {
	p.skipWhitespace()
	t, err := p.next()
	if err != nil {
		return 0, err
	}
	if t.typ == tokenEOF {
		return 0, nil
	}

	if isSignedInteger(t) {
		n, err := parseInt(t.s)
		if err != nil {
			return 0, p.errorf(t, "parsing value as integer: %v", err)
		}
		return n, nil
	}
	if !(t.isDelim("+") || t.isDelim("-")) {
		return 0, p.errorf(t, "expected one of the following: <signed-intger>, '+', '-'")
	}
	isNeg := t.isDelim("-")

	p.skipWhitespace()
	t, err = p.next()
	if err != nil {
		return 0, err
	}

	if !isSignlessInteger(t) {
		return 0, p.errorf(t, "expected <signless-integer>")
	}
	n, err := parseInt(t.s)
	if err != nil {
		return 0, p.errorf(t, "parsing value as integer: %v", err)
	}
	if isNeg {
		return 0 - n, nil
	}
	return n, nil
}

// https://drafts.csswg.org/css-syntax-3/#the-anb-type
func (p *parser) aNPlusB() (*nth, error) {
	p.skipWhitespace()
	t, err := p.next()
	if err != nil {
		return nil, err
	}
	if t.isIdent("even") {
		return &nth{a: 2}, nil
	}
	if t.isIdent("odd") {
		return &nth{a: 2, b: 1}, nil
	}
	if isInteger(t) {
		b, err := parseInt(t.s)
		if err != nil {
			return nil, p.errorf(t, "parsing value as integer: %v", err)
		}
		return &nth{b: b}, nil
	}

	if isNDimension(t) {
		a, err := parseInt(t.s)
		if err != nil {
			return nil, p.errorf(t, "parsing value as integer: %v", err)
		}
		b, err := p.b()
		if err != nil {
			return nil, err
		}
		return &nth{a: a, b: b}, nil
	}

	if isNDashDigitDimension(t) {
		// Token is of form "4n-3" where "4" is the string and "n-3" is the
		// dimension.
		a, err := parseInt(t.s)
		if err != nil {
			return nil, p.errorf(t, "parsing value as integer: %v", err)
		}
		b, err := parseInt(strings.TrimPrefix(t.dim, "n"))
		if err != nil {
			return nil, p.errorf(t, "parsing dimension as integer: %v", err)
		}
		return &nth{a: a, b: b}, nil
	}

	if isDashNDashDigitIdent(t) {
		// Token is of form "-n-3".
		b, err := parseInt(strings.TrimPrefix(t.s, "-n"))
		if err != nil {
			return nil, p.errorf(t, "parsing b as integer: %v", err)
		}
		return &nth{a: -1, b: b}, nil
	}

	if isNDashDimension(t) {
		// String is of form "4n- 3".
		a, err := parseInt(t.s)
		if err != nil {
			return nil, p.errorf(t, "parsing value as integer: %v", err)
		}
		p.skipWhitespace()
		t, err := p.next()
		if err != nil {
			return nil, err
		}
		if !isSignlessInteger(t) {
			return nil, p.errorf(t, "expected unsigned integer")
		}
		n, err := parseInt(t.s)
		if err != nil {
			return nil, p.errorf(t, "parsing value as integer: %v", err)
		}
		return &nth{a: a, b: 0 - n}, nil
	}

	if t.isIdent("-n-") {
		// String is of form "-n- 3".
		p.skipWhitespace()
		t, err := p.next()
		if err != nil {
			return nil, err
		}
		if !isSignlessInteger(t) {
			return nil, p.errorf(t, "expected unsigned integer")
		}
		n, err := parseInt(t.s)
		if err != nil {
			return nil, p.errorf(t, "parsing value as integer: %v", err)
		}
		return &nth{a: -1, b: 0 - n}, nil
	}

	if t.isIdent("-n") {
		b, err := p.b()
		if err != nil {
			return nil, err
		}
		return &nth{a: -1, b: b}, nil
	}

	if t.isDelim("+") {
		p.skipWhitespace()
		tok, err := p.next()
		if err != nil {
			return nil, err
		}
		t = tok
	}

	if t.isIdent("n") {
		b, err := p.b()
		if err != nil {
			return nil, err
		}
		return &nth{a: 1, b: b}, nil
	}

	if t.isIdent("n-") {
		p.skipWhitespace()
		tok, err := p.next()
		if err != nil {
			return nil, err
		}
		t = tok
		if !isSignlessInteger(t) {
			return nil, p.errorf(t, "expected unsigned integer")
		}
		n, err := parseInt(t.s)
		if err != nil {
			return nil, p.errorf(t, "parsing value as integer: %v", err)
		}
		return &nth{a: 1, b: 0 - n}, nil
	}
	return nil, p.errorf(t, "expected 'even', 'odd', or integer type")
}
