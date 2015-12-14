package css

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

type tokenType int

const (
	typeAstr          tokenType = iota // *
	typeBar                            // |
	typeColon                          // :
	typeComma                          // ,
	typeDimension                      // 4n
	typeDot                            // .
	typeFunc                           // nth-child(
	typeHash                           // #foo
	typeIdent                          // h2
	typeLeftBrace                      // [
	typeMatch                          // =
	typeMatchDash                      // |=
	typeMatchIncludes                  // ~=
	typeMatchPrefix                    // ^=
	typeMatchSubstr                    // *=
	typeMatchSuffix                    // $=
	typeNot                            // :not(
	typeNum                            // 37
	typePlus                           // +
	typeRightBrace                     // ]
	typeRightParen                     // )
	typeSpace                          // \t \n\r\f
	typeString                         // 'hello world'
	typeSub                            // -
	typeGreater                        // >
	typeTilde                          // ~

	typeErr
	typeEOF
)

var tokenStr = map[tokenType]string{
	typeAstr:          "*",
	typeBar:           "|",
	typeColon:         ":",
	typeComma:         ",",
	typeDimension:     "DIMENSION",
	typeDot:           ".",
	typeFunc:          "FUNCTION",
	typeHash:          "#",
	typeIdent:         "IDENT",
	typeLeftBrace:     "[",
	typeMatch:         "=",
	typeMatchDash:     "|=",
	typeMatchIncludes: "~=",
	typeMatchPrefix:   "^=",
	typeMatchSubstr:   "*=",
	typeMatchSuffix:   "$=",
	typeNot:           ":not(",
	typeNum:           "NUMBER",
	typePlus:          "+",
	typeRightBrace:    "]",
	typeRightParen:    ")",
	typeSpace:         "SPACE",
	typeString:        "STRING",
	typeSub:           "-",
	typeGreater:       ">",
	typeTilde:         "~",
	typeErr:           "ERROR",
	typeEOF:           "EOF",
}

func (t tokenType) String() string {
	if str, ok := tokenStr[t]; ok {
		return str
	}
	return fmt.Sprintf("tokenType(%d)", t)
}

var matchChar = map[rune]tokenType{
	'|': typeMatchDash,
	'~': typeMatchIncludes,
	'^': typeMatchPrefix,
	'*': typeMatchSubstr,
	'$': typeMatchSuffix,
}

// not included
//   ':' may be ':not(', must check that first
//   '.' may be the beginning of a number
var charToType = map[rune]tokenType{
	'*': typeAstr,
	'|': typeBar,
	',': typeComma,
	'[': typeLeftBrace,
	'=': typeMatch,
	'+': typePlus,
	']': typeRightBrace,
	')': typeRightParen,
	'-': typeSub,
	'>': typeGreater,
	'~': typeTilde,
}

var combinatorChar = map[rune]tokenType{
	'+': typePlus,
	'>': typeGreater,
	',': typeComma,
	// tilde's cannot be matched based on a single rune due to '~='
}

const eof = -1

type token struct {
	typ   tokenType
	val   string
	start int
}

func (t token) String() string {
	return fmt.Sprintf("type=%s val=%s start=%d", t.typ, strconv.Quote(t.val), t.start)
}

type state func() state

type lexer struct {
	s    string // the string to lex
	last int
	pos  int

	c chan token
}

func newLexer(s string) (*lexer, error) {
	if !utf8.ValidString(s) {
		// TODO(eric): maybe detect this during the parsing?
		return nil, errors.New("css: expression is not valid utf8")
	}
	return &lexer{s: s, c: make(chan token, 1)}, nil
}

func (l *lexer) run() {
	for state := l.parseNext; state != nil; state = state() {
	}
}

func (l *lexer) token() token {
	return <-l.c
}

func (l *lexer) next() rune {
	if l.pos >= len(l.s) {
		return eof
	}
	r, size := utf8.DecodeRuneInString(l.s[l.pos:])
	l.pos += size
	return r
}

func (l *lexer) peek() rune {
	if l.pos >= len(l.s) {
		return eof
	}
	r, _ := utf8.DecodeRuneInString(l.s[l.pos:])
	return r
}

func (l *lexer) backup() {
	_, size := utf8.DecodeLastRuneInString(l.s[:l.pos])
	if size == 0 || (l.pos-size) < l.last {
		panic("backed up past last emitted token")
	}
	l.pos -= size
}

func (l *lexer) emit(t tokenType) {
	if l.last == l.pos {
		panic(fmt.Sprintf("css: nothing to emit at at pos %d", l.pos))
	}
	l.c <- token{typ: t, val: l.s[l.last:l.pos], start: l.last}
	l.last = l.pos
}

func (l *lexer) errorf(format string, a ...interface{}) state {
	l.c <- token{
		val:   fmt.Sprintf(format, a...),
		typ:   typeErr,
		start: l.last,
	}
	return nil
}

func (l *lexer) eof() state {
	if l.pos != len(l.s) {
		panic("emitted eof without being at eof")
	}
	if l.last != l.pos {
		panic("emitted eof with unevaluated tokens")
	}
	l.c <- token{typ: typeEOF, start: l.last}
	return nil
}

func (l *lexer) parseNext() state {
	for {
		switch r := l.peek(); {
		case r == eof:
			return l.eof
		case isNum(r), r == '.':
			return l.parseNumOrDot
		case isSpace(r):
			return l.parseSpace
		case r == '\'', r == '"':
			return l.parseString
		case r == '#':
			return l.parseHash
		case r == ':':
			return l.parseColon
		default:
			if typ, ok := matchChar[r]; ok {
				l.next()
				if l.peek() == '=' {
					l.next()
					l.emit(typ)
					break
				} else {
					l.backup()
				}
			}
			if typ, ok := charToType[r]; ok {
				l.next()
				l.emit(typ)
				break
			}
			return l.parseIdent
		}
	}
}

func (l *lexer) parseSpace() state {
	l.skipSpace()
	if l.peek() == '~' {
		l.next()
		if l.peek() == '=' {
			l.backup()
			l.emit(typeSpace)
			l.next()
			l.next()
			l.emit(typeMatchIncludes)
		} else {
			l.emit(typeTilde)
		}
		return l.parseNext
	}
	if typ, ok := combinatorChar[l.peek()]; ok {
		l.next()
		l.emit(typ)
	} else {
		l.emit(typeSpace)
	}
	return l.parseNext
}

func (l *lexer) parseColon() state {
	if l.next() != ':' {
		panic("expected ':' before calling parseColon")
	}

	chars := []string{"nN", "oO", "tT", "("}
	backup := 0
	for _, c := range chars {
		if !strings.ContainsRune(c, l.peek()) {
			for i := 0; i < backup; i++ {
				l.backup()
			}
			l.emit(typeColon)
			return l.parseNext
		}
		l.next()
		backup++
	}
	l.emit(typeNot)
	return l.parseNext
}

func (l *lexer) parseNumOrDot() state {
	r := l.next()
	if r != '.' && !isNum(r) {
		// programmer error, expects next char to be . or 0-9
		panic("expected '.' or 0-9 before calling parseNumOrDot")
	}

	seenDot := r == '.'

	if seenDot {
		if !isNum(l.peek()) {
			l.emit(typeDot)
		}
		return l.parseNext
	}
	l.skipNums()

	if !seenDot && l.peek() == '.' {
		l.next()
		if !isNum(l.peek()) {
			l.backup()
			l.emit(typeNum)
			l.next()
			l.emit(typeDot)
			return l.parseNext
		}
		l.skipNums()
	}
	ok, err := l.skipIdent()
	if err != nil {
		return l.errorf(err.Error())
	}
	if ok {
		l.emit(typeDimension)
	} else {
		l.emit(typeNum)
	}
	return l.parseNext
}

func (l *lexer) parseString() state {
	strChar := l.next()
	if strChar != '\'' && strChar != '"' {
		panic("expected '\\'' or '\"' before calling parseString")
	}
	for {
		switch r := l.next(); {
		case r == eof:
			return l.errorf("unmatched string quote")
		case r == '\n', r == '\r', r == '\f':
			return l.errorf("invalid unescaped string character")
		case r == '\\':
			switch l.peek() {
			case '\n', '\f':
				l.next()
			case '\r':
				l.next()
				if l.peek() == '\n' {
					l.next()
				}
			default:
				if err := l.skipEscape(); err != nil {
					return l.errorf(err.Error())
				}
			}
		case r == strChar:
			l.emit(typeString)
			return l.parseNext
		}
	}
}

func (l *lexer) parseIdent() state {
	if ok, err := l.skipIdent(); err != nil {
		return l.errorf(err.Error())
	} else if ok {
		if l.peek() == '(' {
			l.next()
			l.emit(typeFunc)
		} else {
			l.emit(typeIdent)
		}
		return l.parseNext
	} else {
		return l.errorf("unexpected char")
	}
}

func (l *lexer) parseHash() state {
	if l.next() != '#' {
		panic("expected '#' before calling parseHash")
	}
	firstChar := true
	for {
		switch r := l.peek(); {
		case r == '_', r == '-', isAlphaNum(r), isNonAscii(r):
			l.next()
		case r == '\\':
			l.next()
			if err := l.skipEscape(); err != nil {
				return l.errorf(err.Error())
			}
		default:
			if firstChar {
				return l.errorf("expected identifier after '#'")
			}
			l.emit(typeHash)
			return l.parseNext
		}
		firstChar = false
	}
}

func isNonAscii(r rune) bool {
	return r > 0177 && r != eof
}

func isHex(r rune) bool {
	return isNum(r) || ('a' <= r && r <= 'f') || ('A' <= r && r <= 'F')
}

func isAlpha(r rune) bool {
	return ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z')
}

func isNum(r rune) bool {
	return ('0' <= r && r <= '9')
}

func isAlphaNum(r rune) bool {
	return isNum(r) || isAlpha(r)
}

func isSpace(r rune) bool {
	return strings.ContainsRune(" \t\r\n\f", r)
}

func (l *lexer) skipNums() {
	for isNum(l.peek()) {
		l.next()
	}
}

func (l *lexer) skipSpace() {
	for isSpace(l.peek()) {
		l.next()
	}
	return
}

// skipEscape skips the characters following the escape character '\'.
// It assumes that the lexer has already consumed this character.
func (l *lexer) skipEscape() error {
	r := l.next()
	if isHex(r) {
		// parse unicode
		for i := 0; i < 5; i++ {
			if !isHex(l.peek()) {
				break
			}
			l.next()
		}
		switch l.peek() {
		case ' ', '\t', '\n', '\f':
			l.next()
		case '\r':
			l.next()
			if l.peek() == '\n' {
				l.next()
			}
		}
		return nil
	}
	switch r {
	case '\r', '\n', '\f':
		return errors.New("invalid character after escape")
	case eof:
		return errors.New("invalid EOF after escape")
	}
	l.next()
	return nil
}

// skipIdent attempts to move the lexer to the end of the next identifier.
// If found is false and err is nil, the lexer was not advanced.
func (l *lexer) skipIdent() (found bool, err error) {
	if found = l.peek() == '-'; found {
		l.next()
	}

	switch r := l.peek(); {
	case r == '_', isAlpha(r), isNonAscii(r):
		found = true
		l.next()
	case r == '\\':
		found = true
		l.next()
		if err = l.skipEscape(); err != nil {
			return false, err
		}
	default:
		if found == true {
			err = errors.New("expected identifier after '-'")
		}
		return
	}

	for {
		switch r := l.peek(); {
		case r == '_', r == '-', isAlphaNum(r), isNonAscii(r):
			found = true
			l.next()
		case r == '\\':
			found = true
			l.next()
			if err = l.skipEscape(); err != nil {
				return false, err
			}
		default:
			return found, nil
		}
	}
}
