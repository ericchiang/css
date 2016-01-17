package css

import (
	"reflect"
	"testing"
)

func TestLexerNext(t *testing.T) {

	l, err := newLexer("hello ")
	if err != nil {
		t.Fatal(err)
	}
	backup := func() rune {
		l.backup()
		return ' '
	}
	tests := []struct {
		f   func() rune
		exp rune
	}{
		{l.peek, 'h'},
		{l.next, 'h'},
		{l.next, 'e'},
		{backup, ' '},
		{l.peek, 'e'},
		{l.next, 'e'},
		{l.next, 'l'},
		{l.next, 'l'},
		{l.next, 'o'},
		{backup, ' '},
		{backup, ' '},
		{l.next, 'l'},
		{l.next, 'o'},
		{l.peek, ' '},
		{l.next, ' '},
		{l.next, eof},
		{l.next, eof},
		{l.peek, eof},
	}
	for i, tt := range tests {
		got := tt.f()
		if got != tt.exp {
			t.Errorf("case=%d: exp=%c, got=%c", i, tt.exp, got)
		}
	}
}

func TestEmit(t *testing.T) {
	l, err := newLexer("hello world")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		f   func()
		exp string
	}{
		{
			func() {
				for i := 0; i < len("hello"); i++ {
					l.next()
				}
			},
			"hello",
		},
		{
			func() { l.next() },
			" ",
		},
		{
			func() {
				for {
					if l.next() == eof {
						return
					}
				}
			},
			"world",
		},
	}

	for i, tt := range tests {
		tt.f()
		l.emit(0)
		tok := l.token()
		if tok.val != tt.exp {
			t.Errorf("case=%d: exp=%s, got=%s", i, tt.exp, tok.val)
		}
	}
}

func TestIsNonAscii(t *testing.T) {
	tests := []struct {
		r    rune
		want bool
	}{
		{' ', false},
		{'a', false},
		{eof, false},
		{'世', true},
	}
	for i, tt := range tests {
		got := isNonAscii(tt.r)
		if got != tt.want {
			t.Errorf("case=%d: want=%t, got=%t", i, tt.want, got)
		}
	}
}

func TestLexer(t *testing.T) {
	tests := []struct {
		s    string
		want []token
	}{
		{"7.3", []token{
			token{typeNum, "7.3", 0}, token{typeEOF, "", 3},
		}},
		{"7.", []token{
			token{typeNum, "7", 0}, token{typeDot, ".", 1}, token{typeEOF, "", 2},
		}},
		{"7 \t5n", []token{
			token{typeNum, "7", 0}, token{typeSpace, " \t", 1}, token{typeDimension, "5n", 3},
			token{typeEOF, "", 5},
		}},
		{"  ~", []token{
			token{typeTilde, "  ~", 0}, token{typeEOF, "", 3},
		}},
		{"  ~=", []token{
			token{typeSpace, "  ", 0}, token{typeMatchIncludes, "~=", 2}, token{typeEOF, "", 4},
		}},
		{"lang", []token{
			token{typeIdent, "lang", 0}, token{typeEOF, "", 4},
		}},
		{"lang(", []token{
			token{typeFunc, "lang(", 0}, token{typeEOF, "", 5},
		}},
		{"hi#name 43", []token{
			token{typeIdent, "hi", 0}, token{typeHash, "#name", 2}, token{typeSpace, " ", 7},
			token{typeNum, "43", 8}, token{typeEOF, "", 10},
		}},
		{`'this is  \' a string ' "another string"`, []token{
			token{typeString, `'this is  \' a string '`, 0}, token{typeSpace, " ", 23},
			token{typeString, `"another string"`, 24}, token{typeEOF, "", 40},
		}},
		{"::foo(", []token{
			token{typeColon, ":", 0}, token{typeColon, ":", 1}, token{typeFunc, "foo(", 2},
			token{typeEOF, "", 6},
		}},
		{":not(#h2", []token{
			token{typeNot, ":not(", 0}, token{typeHash, "#h2", 5}, token{typeEOF, "", 8},
		}},
		{":not#h2", []token{
			token{typeColon, ":", 0}, token{typeIdent, "not", 1}, token{typeHash, "#h2", 4},
			token{typeEOF, "", 7},
		}},
		{"a[href^='https://']", []token{
			token{typeIdent, "a", 0}, token{typeLeftBrace, "[", 1}, token{typeIdent, "href", 2},
			token{typeMatchPrefix, "^=", 6}, token{typeString, "'https://'", 8},
			token{typeRightBrace, "]", 18}, token{typeEOF, "", 19},
		}},
		{"h2~a", []token{
			token{typeIdent, "h2", 0}, token{typeTilde, "~", 2}, token{typeIdent, "a", 3},
			token{typeEOF, "", 4},
		}},
		{"p ~ span", []token{
			token{typeIdent, "p", 0}, token{typeTilde, " ~", 1}, token{typeSpace, " ", 3},
			token{typeIdent, "span", 4}, token{typeEOF, "", 8},
		}},
		{"span > p, p", []token{
			token{typeIdent, "span", 0}, token{typeGreater, " >", 4}, token{typeSpace, " ", 6},
			token{typeIdent, "p", 7}, token{typeComma, ",", 8}, token{typeSpace, " ", 9},
			token{typeIdent, "p", 10}, token{typeEOF, "", 11},
		}},
		{"-2n-1", []token{
			token{typeSub, "-", 0}, token{typeDimension, "2n-1", 1}, token{typeEOF, "", 5},
		}},
	}
	for i, tt := range tests {
		l, err := newLexer(tt.s)
		if err != nil {
			t.Errorf("case=%d: could not create lexer: %v", i, err)
		}
		go l.run()
		var got []token
		for {
			tok := l.token()
			got = append(got, tok)
			if tok.typ == typeErr || tok.typ == typeEOF {
				break
			}
		}
		min := len(got)
		if len(got) != len(tt.want) {
			t.Errorf("case=%d: wanted=%d tokens, got=%d", i, len(tt.want), len(got))
		}
		if min > len(tt.want) {
			min = len(tt.want)
		}
		for j := 0; j < min; j++ {
			tokGot, tokWant := got[j], tt.want[j]
			if !reflect.DeepEqual(tokGot, tokWant) {
				t.Errorf("case=%d, tok=%d: wanted token=(%v), got token=(%v)", i, j, tokWant, tokGot)
			}
		}
	}
}
