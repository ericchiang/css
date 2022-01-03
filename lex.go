package css

import (
	"fmt"
	"unicode/utf8"
)

// lexer implements tokenization for CSS selectors. The algorithm follows the
// spec recommentations.
//
// https://www.w3.org/TR/css-syntax-3/#tokenization
type lexer struct {
	s    string
	last int
	pos  int
}

func newLexer(s string) *lexer {
	return &lexer{s, 0, 0}
}

const eof = 0

func (l *lexer) peek() rune {
	if len(l.s) <= l.pos {
		return eof
	}
	r, _ := utf8.DecodeRuneInString(l.s[l.pos:])
	return r
}

func (l *lexer) peekN(n int) rune {
	var r rune
	pos := l.pos
	for i := 0; i <= n; i++ {
		if len(l.s) <= pos {
			return eof
		}
		var n int
		r, n = utf8.DecodeRuneInString(l.s[pos:])
		pos += n
	}
	return r
}

// push is the equivalent of "reconsume the current input code point".
func (l *lexer) push(r rune) {
	l.pos -= utf8.RuneLen(r)
}

func (l *lexer) pop() rune {
	if len(l.s) <= l.pos {
		return eof
	}
	r, n := utf8.DecodeRuneInString(l.s[l.pos:])
	l.pos += n
	return r
}

func (l *lexer) popN(n int) {
	for i := 0; i < n; i++ {
		l.pop()
	}
}

type tokenType int

// Create a shorter type aliases so links to csswg.org don't wrap.
type tt = tokenType

const (
	_                 tt = iota
	tokenAtKeyword       // https://drafts.csswg.org/css-syntax-3/#typedef-at-keyword-token
	tokenBracketClose    // https://drafts.csswg.org/css-syntax-3/#tokendef-close-square
	tokenBracketOpen     // https://drafts.csswg.org/css-syntax-3/#tokendef-open-square
	tokenCDC             // https://drafts.csswg.org/css-syntax-3/#typedef-cdc-token
	tokenCDO             // https://drafts.csswg.org/css-syntax-3/#typedef-cdo-token
	tokenColon           // https://drafts.csswg.org/css-syntax-3/#typedef-colon-token
	tokenComma           // https://drafts.csswg.org/css-syntax-3/#typedef-comma-token
	tokenCurlyClose      // https://drafts.csswg.org/css-syntax-3/#tokendef-close-curly
	tokenCurlyOpen       // https://drafts.csswg.org/css-syntax-3/#tokendef-open-curly
	tokenDelim           // https://drafts.csswg.org/css-syntax-3/#typedef-delim-token
	tokenDimension       // https://drafts.csswg.org/css-syntax-3/#typedef-dimension-token
	tokenEOF             // https://drafts.csswg.org/css-syntax-3/#typedef-eof-token
	tokenFunction        // https://drafts.csswg.org/css-syntax-3/#typedef-function-token
	tokenHash            // https://drafts.csswg.org/css-syntax-3/#typedef-hash-token
	tokenIdent           // https://www.w3.org/TR/css-syntax-3/#typedef-ident-token
	tokenNumber          // https://drafts.csswg.org/css-syntax-3/#typedef-number-token
	tokenParenClose      // https://drafts.csswg.org/css-syntax-3/#tokendef-close-paren
	tokenParenOpen       // https://drafts.csswg.org/css-syntax-3/#tokendef-open-paren
	tokenPercent         // https://drafts.csswg.org/css-syntax-3/#typedef-percentage-token
	tokenSemicolon       // https://drafts.csswg.org/css-syntax-3/#typedef-semicolon-token
	tokenString          // https://drafts.csswg.org/css-syntax-3/#typedef-string-token
	tokenURL             // https://drafts.csswg.org/css-syntax-3/#typedef-url-token
	tokenWhitespace      // https://drafts.csswg.org/css-syntax-3/#typedef-whitespace-token
)

var tokenTypeString = map[tokenType]string{
	tokenAtKeyword:    "<at-keyword-token>",
	tokenBracketClose: "<]-token>",
	tokenBracketOpen:  "<[-token>",
	tokenCDC:          "<CDC-token>",
	tokenCDO:          "<CDO-token>",
	tokenColon:        "<colon-token>",
	tokenComma:        "<comma-token>",
	tokenCurlyClose:   "<}-token>",
	tokenCurlyOpen:    "<{-token>",
	tokenDelim:        "<delim-token>",
	tokenDimension:    "<dimension-token>",
	tokenEOF:          "<eof-token>",
	tokenFunction:     "<function-token>",
	tokenHash:         "<hash-token>",
	tokenIdent:        "<ident-token>",
	tokenNumber:       "<number-token>",
	tokenParenClose:   "<)-token>",
	tokenParenOpen:    "<(-token>",
	tokenPercent:      "<percentage-token>",
	tokenSemicolon:    "<semicolon-token>",
	tokenString:       "<string-token>",
	tokenURL:          "<url-token>",
	tokenWhitespace:   "<whitespace-token>",
}

func (t tokenType) String() string {
	if s, ok := tokenTypeString[t]; ok {
		return s
	}
	return fmt.Sprintf("<0x%x-token>", int(t))
}

type token struct {
	typ tokenType
	s   string
	pos int
}

func (t token) String() string {
	return fmt.Sprintf("%s %q pos=%d", t.typ, t.s, t.pos)
}

type lexErr struct {
	msg  string
	last int
	pos  int
}

func (l *lexErr) Error() string {
	return l.msg
}

func (l *lexer) errorf(format string, v ...interface{}) error {
	return &lexErr{fmt.Sprintf(format, v...), l.last, l.pos}
}

func (l *lexer) token(typ tokenType) token {
	t := token{typ, l.s[l.last:l.pos], l.last}
	l.last = l.pos
	return t
}

// https://www.w3.org/TR/css-syntax-3/#consume-token
func (l *lexer) next() (token, error) {
	r := l.pop()

	if isWhitespace(r) {
		for isWhitespace(l.peek()) {
			l.pop()
		}
		return l.token(tokenWhitespace), nil
	}

	if isDigit(r) {
		l.push(r)
		return l.consumeNumericToken()
	}

	if isNameStart(r) {
		l.push(r)
		return l.consumeIdentLikeToken()
	}

	switch r {
	case '"', '\'':
		return l.string(r)
	case eof:
		return l.token(tokenEOF), nil
	case '#':
		if isName(l.peek()) || isValidEscape(l.peek(), l.peekN(1)) {
			if err := l.consumeName(); err != nil {
				return token{}, err
			}
			return l.token(tokenHash), nil
		}
		return l.token(tokenDelim), nil
	case '(':
		return l.token(tokenParenOpen), nil
	case ')':
		return l.token(tokenParenClose), nil
	case '+':
		if isNumStart(r, l.peek(), l.peekN(1)) {
			l.push(r)
			return l.consumeNumericToken()
		}
		return l.token(tokenDelim), nil
	case ',':
		return l.token(tokenComma), nil
	case '-':
		if isNumStart(r, l.peek(), l.peekN(1)) {
			l.push(r)
			return l.consumeNumericToken()
		}
		if l.peek() == '-' && l.peekN(1) == '>' {
			l.popN(2)
			return l.token(tokenCDC), nil
		}
		return l.token(tokenDelim), nil
	case '.':
		if isNumStart(r, l.peek(), l.peekN(1)) {
			l.push(r)
			return l.consumeNumericToken()
		}
		return l.token(tokenDelim), nil
	case ':':
		return l.token(tokenColon), nil
	case ';':
		return l.token(tokenSemicolon), nil
	case '<':
		if l.peek() == '!' && l.peekN(1) == '-' && l.peekN(2) == '-' {
			l.popN(3)
			return l.token(tokenCDO), nil
		}
		return l.token(tokenDelim), nil
	case '@':
		if isIdentStart(l.peek(), l.peekN(1), l.peekN(2)) {
			if err := l.consumeName(); err != nil {
				return token{}, err
			}
			return l.token(tokenAtKeyword), nil
		}
		return l.token(tokenDelim), nil
	case '[':
		return l.token(tokenBracketOpen), nil
	case '\\':
		if !isValidEscape(r, l.peek()) {
			return token{}, l.errorf("invalid escape character")
		}
		l.push(r)
		return l.consumeIdentLikeToken()
	case ']':
		return l.token(tokenBracketClose), nil
	case '{':
		return l.token(tokenCurlyOpen), nil
	case '}':
		return l.token(tokenCurlyClose), nil
	}
	return l.token(tokenDelim), nil
}

// https://www.w3.org/TR/css-syntax-3/#consume-a-string-token
func (l *lexer) string(quote rune) (token, error) {
	for {
		switch l.pop() {
		case quote:
			return l.token(tokenString), nil
		case eof:
			return token{}, l.errorf("unexpected eof parsing string")
		case '\n':
			return token{}, l.errorf("unexpected newline parsing string")
		case '\\':
			switch l.peek() {
			case eof:
			case '\n':
				return token{}, l.errorf("unexpected newline after '\\' parsing string")
			default:
				if err := l.consumeEscape(); err != nil {
					return token{}, l.errorf("parsing string: %v", err)
				}
			}
		}
	}
}

// https://www.w3.org/TR/css-syntax-3/#consume-an-escaped-code-point
func (l *lexer) consumeEscape() error {
	r := l.pop()
	if r == eof {
		return l.errorf("unexpected newline after escape sequence")
	}
	if !isHex(r) {
		return nil
	}
	n := 0
	for {
		r := l.peek()
		if isHex(r) {
			l.pop()
			n++
			if n > 5 {
				return l.errorf("too many hex digits consuming escape sequence")
			}
			continue
		}

		if isWhitespace(r) {
			l.pop()
			continue
		}
		return nil
	}
}

// https://www.w3.org/TR/css-syntax-3/#consume-a-name
func (l *lexer) consumeName() error {
	for {
		r := l.peek()
		if isName(r) {
			l.pop()
			continue
		}

		if isValidEscape(r, l.peekN(1)) {
			l.pop()
			if err := l.consumeEscape(); err != nil {
				return err
			}
			continue
		}
		return nil
	}
}

// https://www.w3.org/TR/css-syntax-3/#consume-a-numeric-token
func (l *lexer) consumeNumericToken() (token, error) {
	l.consumeNumber()

	if isIdentStart(l.peek(), l.peekN(1), l.peekN(2)) {
		if err := l.consumeName(); err != nil {
			return token{}, err
		}
		return l.token(tokenDimension), nil
	}

	if l.peek() == '%' {
		l.pop()
		return l.token(tokenPercent), nil
	}
	return l.token(tokenNumber), nil
}

// https://www.w3.org/TR/css-syntax-3/#consume-an-ident-like-token
func (l *lexer) consumeIdentLikeToken() (token, error) {
	if l.startsURL() {
		return l.consumeURL()
	}

	if err := l.consumeName(); err != nil {
		return token{}, err
	}

	if l.peek() == '(' {
		l.pop()
		return l.token(tokenFunction), nil
	}

	return l.token(tokenIdent), nil
}

func (l *lexer) startsURL() bool {
	if !(l.peek() == 'u' || l.peek() == 'U') {
		return false
	}
	if !(l.peekN(1) == 'r' || l.peekN(1) == 'R') {
		return false
	}
	if !(l.peekN(2) == 'l' || l.peekN(2) == 'L') {
		return false
	}
	if l.peekN(3) != '(' {
		return false
	}

	// Consume up to two characters of whitespace.
	n := 4
	for i := 0; i < 2; i++ {
		if !isWhitespace(l.peekN(n)) {
			break
		}
		n++
	}

	r1 := l.peekN(n)
	r2 := l.peekN(n + 1)

	r := r1
	if isWhitespace(r1) {
		r = r2
	}
	if r == '\'' || r == '"' {
		return false
	}

	l.popN(4)
	return true
}

// https://www.w3.org/TR/css-syntax-3/#consume-a-url-token
func (l *lexer) consumeURL() (token, error) {
	for isWhitespace(l.peek()) {
		l.pop()
	}

	for {
		r := l.pop()
		switch {
		case r == ')':
			return l.token(tokenURL), nil
		case r == eof:
			return token{}, l.errorf("unexpected eof parsing URL")
		case isWhitespace(r):
			for isWhitespace(l.peek()) {
				l.pop()
			}
			r := l.pop()
			if r == ')' {
				return l.token(tokenURL), nil
			}
			return token{}, l.errorf("unexpected character parsing URL: %c", r)
		case r == '\'', r == '"', r == '(', isNonPrintable(r):
			return token{}, l.errorf("invalid character parsing URL: %c", r)
		case r == '\\':
			if !isValidEscape(r, l.peek()) {
				return token{}, l.errorf("invalid '\\' parsing URL")
			}
			if err := l.consumeEscape(); err != nil {
				return token{}, l.errorf("invalid escape parsing URL: %v", err)
			}
		}
	}
}

// https://www.w3.org/TR/css-syntax-3/#consume-a-number
func (l *lexer) consumeNumber() {
	if l.peek() == '+' || l.peek() == '-' {
		l.pop()
	}

	for isDigit(l.peek()) {
		l.pop()
	}

	if l.peek() == '.' && isDigit(l.peekN(1)) {
		l.popN(2)

		for isDigit(l.peek()) {
			l.pop()
		}
	}

	r1 := l.peek()
	r2 := l.peekN(1)
	r3 := l.peekN(2)

	if r1 == 'E' || r1 == 'e' {
		if isDigit(r2) {
			l.popN(2)

			for isDigit(l.peek()) {
				l.pop()
			}
		} else if (r2 == '+' || r2 == '-') && isDigit(r3) {
			l.popN(3)
			for isDigit(l.peek()) {
				l.pop()
			}
		}
	}
}

// https://www.w3.org/TR/css-syntax-3/#whitespace
func isWhitespace(r rune) bool {
	switch r {
	case '\n', '\t', ' ':
		return true
	default:
		return false
	}
}

// https://www.w3.org/TR/css-syntax-3/#hex-digit
func isHex(r rune) bool {
	return isDigit(r) || ('A' <= r && r <= 'F') || ('a' <= r && r <= 'f')
}

// https://www.w3.org/TR/css-syntax-3/#digit
func isDigit(r rune) bool {
	return '0' <= r && r <= '9'
}

// https://www.w3.org/TR/css-syntax-3/#letter
func isLetter(r rune) bool {
	return ('A' <= r && r <= 'Z') || ('a' <= r && r <= 'z')
}

// https://www.w3.org/TR/css-syntax-3/#non-ascii-code-point
func isNonASCII(r rune) bool {
	return r > 0x80
}

// https://www.w3.org/TR/css-syntax-3/#name-code-point
func isName(r rune) bool {
	return isNameStart(r) || isDigit(r) || r == '-'
}

// https://www.w3.org/TR/css-syntax-3/#name-start-code-point
func isNameStart(r rune) bool {
	return isLetter(r) || isNonASCII(r) || r == '_'
}

// https://www.w3.org/TR/css-syntax-3/#check-if-three-code-points-would-start-a-number
func isNumStart(r1, r2, r3 rune) bool {
	if r1 == '+' || r1 == '-' {
		if isDigit(r2) {
			return true
		}
		if r2 == '.' && isDigit(r3) {
			return true
		}
		return false
	}

	if r1 == '.' {
		return isDigit(r2)
	}
	return isDigit(r1)
}

// https://www.w3.org/TR/css-syntax-3/#check-if-two-code-points-are-a-valid-escape
func isValidEscape(r1, r2 rune) bool {
	if r1 != '\\' {
		return false
	}
	if r2 == '\n' || r2 == eof {
		return false
	}
	return true
}

// https://www.w3.org/TR/css-syntax-3/#check-if-three-code-points-would-start-an-identifier
func isIdentStart(r1, r2, r3 rune) bool {
	if r1 == '-' {
		if isNameStart(r2) {
			return true
		}
		if isValidEscape(r2, r3) {
			return true
		}
	}
	if isNameStart(r1) {
		return true
	}
	if r1 == '\\' && isValidEscape(r1, r2) {
		return true
	}
	return false
}

func isNonPrintable(r rune) bool {
	if 0x0 <= r && r <= 0x8 {
		return true
	}
	if r == '\t' {
		return true
	}
	if 0xe <= r && r <= 0x1f {
		return true
	}
	if r == 0x7F {
		return true
	}
	return false
}
