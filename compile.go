package css

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func Compile(expr string) (*Selector, error) {
	lexer, err := newLexer(expr)
	if err != nil {
		return nil, err
	}
	go lexer.run()
	selectors, err := newCompiler(lexer).compileSelectorsGroup()
	if err != nil {
		return nil, err
	}
	return &Selector{selectorsGroup: selectors}, nil
}

func MustCompile(expr string) *Selector {
	sel, err := Compile(expr)
	if err != nil {
		panic(`css: Compile(` + strconv.Quote(expr) + `): ` + err.Error())
	}
	return sel
}

type SyntaxError struct {
	msg    string
	Offset int
}

func (s *SyntaxError) Error() string {
	return s.msg
}

type tokenEmitter interface {
	token() token
}

type compiler struct {
	t         tokenEmitter
	firstPeek bool
	peekTok   token
}

func newCompiler(t tokenEmitter) *compiler {
	return &compiler{t: t, firstPeek: true}
}

func lexError(t token) *SyntaxError {
	return &SyntaxError{
		msg:    t.val,
		Offset: t.start,
	}
}

func syntaxError(got token, exp ...tokenType) *SyntaxError {
	return &SyntaxError{
		msg:    fmt.Sprintf("expected %s, got %s %s", exp, got.typ, strconv.Quote(got.val)),
		Offset: got.start,
	}
}

func (c *compiler) peek() token {
	if c.firstPeek {
		c.firstPeek = false
		c.peekTok = c.t.token()
	}
	return c.peekTok
}

func (c *compiler) next() token {
	tok := c.peek()
	if tok.typ == typeErr || tok.typ == typeEOF {
		return tok
	}
	c.peekTok = c.t.token()
	return tok
}

func (c *compiler) skipSpace() token {
	for c.peek().typ == typeSpace {
		c.next()
	}
	return c.peek()
}

func (c *compiler) compileSelectorsGroup() ([]selector, error) {
	sel, err := c.compileSelector()
	if err != nil {
		return nil, err
	}
	selectors := []selector{sel}
	for {
		switch t := c.next(); t.typ {
		case typeEOF:
			return selectors, nil
		case typeComma:
			c.skipSpace()
			sel, err := c.compileSelector()
			if err != nil {
				return nil, err
			}
			selectors = append(selectors, sel)
		default:
			return nil, syntaxError(t, typeEOF, typeComma)
		}
	}
}

func (c *compiler) compileSelector() (selector, error) {
	selSeq, err := c.compileSimpleSelectorSeq()
	if err != nil {
		return selector{}, err
	}
	sel := selector{selSeq: selSeq}
	for {
		switch t := c.peek(); t.typ {
		case typePlus, typeGreater, typeTilde, typeSpace:
			c.next()
			c.skipSpace()
			selSeq, err := c.compileSimpleSelectorSeq()
			if err != nil {
				return selector{}, err
			}
			sel.combs = append(sel.combs, combinatorSelector{t.typ, selSeq})
		default:
			return sel, nil
		}
		c.skipSpace()
	}
}

func (c *compiler) compileSimpleSelectorSeq() (selectorSequence, error) {
	var matchers []matcher
	firstLoop := true
	for {
		switch t := c.peek(); t.typ {
		case typeIdent:
			if !firstLoop {
				return selectorSequence{matchers}, nil
			}
			matchers = []matcher{typeSelector{t.val}}
		case typeAstr:
			if !firstLoop {
				return selectorSequence{matchers}, nil
			}
			matchers = []matcher{universal{}}
		case typeDot:
			c.next()
			if t = c.peek(); t.typ != typeIdent {
				return selectorSequence{}, syntaxError(t, typeIdent)
			}
			matchers = []matcher{attrMatcher{"class", t.val}}
		case typeHash:
			matchers = []matcher{attrMatcher{"id", strings.TrimPrefix(t.val, "#")}}
		case typeLeftBrace:
			attrMatcher, err := c.compileAttr()
			if err != nil {
				return selectorSequence{}, err
			}
			matchers = append(matchers, attrMatcher)
		case typeColon:
			m, err := c.compilePseudo()
			if err != nil {
				return selectorSequence{}, err
			}
			matchers = append(matchers, m)
		default:
			if firstLoop {
				return selectorSequence{}, syntaxError(t, typeIdent, typeDot, typeHash)
			}
			return selectorSequence{matchers}, nil
		}
		c.next()
		firstLoop = false
	}
}

func (c *compiler) compileAttr() (matcher, error) {
	if tok := c.next(); tok.typ != typeLeftBrace {
		return nil, syntaxError(tok, typeLeftBrace)
	}
	c.skipSpace()
	tok := c.next()
	if tok.typ != typeIdent {
		return nil, syntaxError(tok, typeIdent)
	}
	key := tok.val
	c.skipSpace()

	var matcherType tokenType

	switch tok := c.next(); tok.typ {
	case typeMatch, typeMatchDash, typeMatchIncludes, typeMatchPrefix, typeMatchSubstr, typeMatchSuffix:
		matcherType = tok.typ
	case typeRightBrace:
		return attrSelector{tok.val}, nil
	default:
		return nil, syntaxError(tok, typeRightBrace)
	}

	c.skipSpace()
	val := ""
	switch tok := c.next(); tok.typ {
	case typeIdent:
		val = tok.val
	case typeString:
		if len(tok.val) > 2 {
			// string correctness is guaranteed by the lexer
			val = tok.val[1 : len(tok.val)-1]
		}
	default:
		return nil, syntaxError(tok, typeIdent, typeString)
	}
	c.skipSpace()

	if t := c.next(); t.typ != typeRightBrace {
		return nil, syntaxError(t, typeRightBrace)
	}

	switch matcherType {
	case typeMatchDash:
		return attrCompMatcher{key, val, dashMatcher}, nil
	case typeMatchIncludes:
		return attrCompMatcher{key, val, includesMatcher}, nil
	case typeMatchPrefix:
		return attrCompMatcher{key, val, prefixMatcher}, nil
	case typeMatchSubstr:
		return attrCompMatcher{key, val, subStrMatcher}, nil
	case typeMatchSuffix:
		return attrCompMatcher{key, val, suffixMatcher}, nil
	default:
		return attrMatcher{key, val}, nil
	}
}

func (c *compiler) compilePseudo() (matcher, error) {
	if tok := c.next(); tok.typ != typeColon {
		return nil, syntaxError(tok, typeColon)
	}
	doubleColon := c.peek().typ == typeColon
	if doubleColon {
		c.next()
	}
	switch t := c.next(); t.typ {
	case typeIdent:
		if doubleColon {
		} else {
			switch t.val {
			case "empty":
				return matcherFunc(empty), nil
			case "first-child":
				return matcherFunc(firstChild), nil
			case "first-of-type":
				return matcherFunc(firstOfType), nil
			case "last-child":
				return matcherFunc(lastChild), nil
			case "last-of-type":
				return matcherFunc(lastOfType), nil
			case "only-child":
				return matcherFunc(onlyChild), nil
			case "only-of-type":
				return matcherFunc(onlyOfType), nil
			case "root":
				return matcherFunc(root), nil
			}
		}
		s := ":"
		if doubleColon {
			s = "::"
		}
		return nil, &SyntaxError{"unknown pseudo: " + strconv.Quote(s+t.val), t.start}

	case typeFunc:
		var m matcher
		var err error
		if doubleColon {
			err = &SyntaxError{"unknown pseudo: ::" + strconv.Quote(t.val), t.start}
		} else {
			var a, b int
			switch t.val {
			case "nth-child(":
				a, b, err = c.parseNthArgs()
				m = nthChild{a, b}
			default:
				err = &SyntaxError{"unknown pseudo: :" + strconv.Quote(t.val), t.start}
			}
		}
		if err == nil && c.peek().typ != typeRightParen {
			err = syntaxError(c.next(), typeRightParen)
		}
		c.next()
		return m, err
	default:
		return nil, syntaxError(t, typeIdent, typeFunc)
	}
}

/*
  : [ 'even' | 'odd' ]
  | [ '-' | PLUS ]? DIMENSION
  | [ '-' | PLUS ]? DIMENSION [ '-' | PLUS ] NUMBER
  [ [ '-' | PLUS ]? NUMBER
*/

func (c *compiler) parseNthArgs() (a, b int, err error) {
	minus := false
	c.skipSpace()
	switch t := c.peek(); t.typ {
	case typeIdent:
		c.next()
		switch t.val {
		case "even":
			return 2, 0, nil
		case "odd":
			return 2, 1, nil
		}
	case typeNum:
		c.next()
		if b, err = strconv.Atoi(t.val); err != nil {
			err = &SyntaxError{err.Error(), t.start}
		}
		return
	case typeSub:
		c.next()
		minus = true
	case typePlus:
		c.next()
	case typeDimension:
	default:
		return 0, 0, syntaxError(t, typeIdent, typeNum, typeSub, typePlus)
	}

	c.skipSpace()
	switch t := c.next(); t.typ {
	case typeDimension:
		var bFound bool
		if a, b, bFound, err = parseNth(t.val); err != nil {
			return 0, 0, &SyntaxError{err.Error(), t.start}
		}
		if minus {
			a = 0 - a
		}
		if bFound {
			return
		}
		b = 0
	case typeNum:
		if b, err = strconv.Atoi(t.val); err != nil {
			return 0, 0, &SyntaxError{err.Error(), t.start}
		}
		if minus {
			b = 0 - b
		}
		c.skipSpace()
		return
	default:
		return 0, 0, syntaxError(t, typeIdent, typeNum, typeSub, typePlus)
	}

	c.skipSpace()
	switch c.peek().typ {
	case typeSub:
		minus = true
	case typePlus:
		minus = false
	default:
		return
	}
	c.next()
	c.skipSpace()
	t := c.next()
	if t.typ != typeNum {
		return 0, 0, syntaxError(t, typeIdent, typeNum, typeSub, typePlus)
	}
	if b, err = strconv.Atoi(t.val); err != nil {
		return 0, 0, &SyntaxError{err.Error(), t.start}
	}
	if minus {
		b = 0 - b
	}
	c.skipSpace()
	return
}

var parseNthErr = errors.New("string is not of form {number}n or {number}n{number}")

var nthRegexp = regexp.MustCompile(`^([-+]?[\d]+)n([-+]?[\d]+)?$`)

func parseNth(s string) (a, b int, bFound bool, err error) {
	submatch := nthRegexp.FindStringSubmatch(s)
	if submatch == nil || len(submatch) != 3 {
		return 0, 0, false, parseNthErr
	}
	if a, err = strconv.Atoi(submatch[1]); err != nil {
		err = parseNthErr
		return
	}
	if bFound = submatch[2] != ""; !bFound {
		return
	}
	if b, err = strconv.Atoi(submatch[2]); err != nil {
		err = parseNthErr
	}
	return
}
