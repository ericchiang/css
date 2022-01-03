package css

import (
	"reflect"
	"testing"
)

func tok(typ tokenType, s string) token {
	return token{typ: typ, s: s}
}

func TestLexer(t *testing.T) {
	tests := []struct {
		s    string
		want []token
	}{
		{
			"   ",
			[]token{
				tok(tokenWhitespace, "   "),
			},
		},
		{
			" \t\n",
			[]token{
				tok(tokenWhitespace, " \t\n"),
			},
		},
		{
			" \"hello\" ",
			[]token{
				tok(tokenWhitespace, " "),
				tok(tokenString, "\"hello\""),
				tok(tokenWhitespace, " "),
			},
		},
		{
			` "\t" `,
			[]token{
				tok(tokenWhitespace, " "),
				tok(tokenString, `"\t"`),
				tok(tokenWhitespace, " "),
			},
		},
		{
			` "\0af" `,
			[]token{
				tok(tokenWhitespace, " "),
				tok(tokenString, `"\0af"`),
				tok(tokenWhitespace, " "),
			},
		},
		{
			` "\0a f" `,
			[]token{
				tok(tokenWhitespace, " "),
				tok(tokenString, `"\0a f"`),
				tok(tokenWhitespace, " "),
			},
		},
		{
			`# "foo"`,
			[]token{
				tok(tokenDelim, "#"),
				tok(tokenWhitespace, " "),
				tok(tokenString, `"foo"`),
			},
		},
		{
			`#foo`,
			[]token{
				tok(tokenHash, "#foo"),
			},
		},
		{
			`#\0100`,
			[]token{
				tok(tokenHash, `#\0100`),
			},
		},
		{
			`#foo()`,
			[]token{
				tok(tokenHash, "#foo"),
				tok(tokenParenOpen, "("),
				tok(tokenParenClose, ")"),
			},
		},
		{
			`+`,
			[]token{
				tok(tokenDelim, "+"),
			},
		},
		{
			`+1`,
			[]token{
				tok(tokenNumber, "+1"),
			},
		},
		{
			`+1.1 +1.11e11 +1.11e+11 +`,
			[]token{
				tok(tokenNumber, "+1.1"),
				tok(tokenWhitespace, " "),
				tok(tokenNumber, "+1.11e11"),
				tok(tokenWhitespace, " "),
				tok(tokenNumber, "+1.11e+11"),
				tok(tokenWhitespace, " "),
				tok(tokenDelim, "+"),
			},
		},
		{
			`+1cm`,
			[]token{
				tok(tokenDimension, "+1cm"),
			},
		},
		{
			`+50%`,
			[]token{
				tok(tokenPercent, "+50%"),
			},
		},
		{
			`,`,
			[]token{
				tok(tokenComma, ","),
			},
		},
		{
			`-1.1 -1.11e11 --> -1.11e-11 -`,
			[]token{
				tok(tokenNumber, "-1.1"),
				tok(tokenWhitespace, " "),
				tok(tokenNumber, "-1.11e11"),
				tok(tokenWhitespace, " "),
				tok(tokenCDC, "-->"),
				tok(tokenWhitespace, " "),
				tok(tokenNumber, "-1.11e-11"),
				tok(tokenWhitespace, " "),
				tok(tokenDelim, "-"),
			},
		},
		{
			`.1 .11e11 .11e-11 .`,
			[]token{
				tok(tokenNumber, ".1"),
				tok(tokenWhitespace, " "),
				tok(tokenNumber, ".11e11"),
				tok(tokenWhitespace, " "),
				tok(tokenNumber, ".11e-11"),
				tok(tokenWhitespace, " "),
				tok(tokenDelim, "."),
			},
		},
		{
			`:;`,
			[]token{
				tok(tokenColon, ":"),
				tok(tokenSemicolon, ";"),
			},
		},
		{
			`< <!--`,
			[]token{
				tok(tokenDelim, "<"),
				tok(tokenWhitespace, " "),
				tok(tokenCDO, "<!--"),
			},
		},
		{
			`@ @foo @-bar`,
			[]token{
				tok(tokenDelim, "@"),
				tok(tokenWhitespace, " "),
				tok(tokenAtKeyword, "@foo"),
				tok(tokenWhitespace, " "),
				tok(tokenAtKeyword, "@-bar"),
			},
		},
		{
			`[]{}`,
			[]token{
				tok(tokenBracketOpen, "["),
				tok(tokenBracketClose, "]"),
				tok(tokenCurlyOpen, "{"),
				tok(tokenCurlyClose, "}"),
			},
		},
		{
			`4.123e-2`,
			[]token{
				tok(tokenNumber, "4.123e-2"),
			},
		},
		{
			`foo bar(`,
			[]token{
				tok(tokenIdent, "foo"),
				tok(tokenWhitespace, " "),
				tok(tokenFunction, "bar("),
			},
		},
		{
			`url(foo) url( foo ) url url("foo")`,
			[]token{
				tok(tokenURL, "url(foo)"),
				tok(tokenWhitespace, " "),
				tok(tokenURL, "url( foo )"),
				tok(tokenWhitespace, " "),
				tok(tokenIdent, "url"),
				tok(tokenWhitespace, " "),
				tok(tokenFunction, "url("),
				tok(tokenString, "\"foo\""),
				tok(tokenParenClose, ")"),
			},
		},
		{
			`*`,
			[]token{
				tok(tokenDelim, "*"),
			},
		},
		{
			`.foo`,
			[]token{
				tok(tokenDelim, "."),
				tok(tokenIdent, "foo"),
			},
		},
	}

L:
	for _, test := range tests {
		test.want = append(test.want, tok(tokenEOF, ""))

		pos := 0
		for i, t := range test.want {
			t.pos = pos
			pos = t.pos + len(t.s)
			test.want[i] = t
		}

		var got []token
		l := newLexer(test.s)

		for {
			tok, err := l.next()
			if err != nil {
				t.Errorf("tokenize selector %q: %v", test.s, err)
				continue L
			}
			got = append(got, tok)
			if tok.typ == tokenEOF {
				break
			}
		}

		if !reflect.DeepEqual(test.want, got) {
			t.Errorf("tokenize selector %q\n got=%s\nwant=%s", test.s, got, test.want)
		}
	}
}

func TestLexerErr(t *testing.T) {
	tests := []string{
		"\"\\\n\"",        // Escape sequence is followed by a newline.
		"\"\\000000000\"", // Escape sequence contains too many hex characters.
		"\\",              // Invalid escape.
		"\"",              // Unclosed string.
		"\"\n\"",          // Newline in string.
		"url(foo",         // URL hits EOF.
		"url(foo())",      // URL hits '('.
	}

	for _, test := range tests {
		l := newLexer(test)
		for {
			tok, err := l.next()
			if err != nil {
				break
			}
			if tok.typ == tokenEOF {
				t.Errorf("expected error parsing %q", test)
				break
			}
		}
	}
}

func TestLexerPop(t *testing.T) {
	tests := []struct {
		s    string
		want []rune
	}{
		{
			"hello, world!",
			[]rune{'h', 'e', 'l', 'l', 'o', ',', ' ', 'w', 'o', 'r', 'l', 'd', '!'},
		},
		{
			"hello, 世界!",
			[]rune{'h', 'e', 'l', 'l', 'o', ',', ' ', '世', '界', '!'},
		},
	}

	for _, test := range tests {
		var got []rune
		l := newLexer(test.s)
		for l.peek() != eof {
			got = append(got, l.pop())
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("lexer parsing code points for %q: got=%v, want=%v", test.s, got, test.want)
		}
	}
}
