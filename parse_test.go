package css

import (
	"errors"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func cmpDiff(x, y interface{}) string {
	return cmp.Diff(x, y, cmp.AllowUnexported(
		attributeSelector{},
		complexSelector{},
		compoundSelector{},
		pseudoClassSelector{},
		pseudoSelector{},
		subclassSelector{},
		token{},
		typeSelector{},
		wqName{},
	))
}

func TestParse(t *testing.T) {
	tests := []struct {
		s    string
		want []complexSelector
	}{
		{"foo", []complexSelector{
			{
				sel: compoundSelector{
					typeSelector: &typeSelector{pos: 0, value: "foo"},
				},
			},
		}},
		{"foo bar", []complexSelector{
			{
				sel: compoundSelector{
					typeSelector: &typeSelector{pos: 0, value: "foo"},
				},
				combinator: "",
				next: &complexSelector{
					pos: 4,
					sel: compoundSelector{
						pos:          4,
						typeSelector: &typeSelector{pos: 4, value: "bar"},
					},
				},
			},
		}},
		{"foo bar spam", []complexSelector{
			{
				sel: compoundSelector{
					typeSelector: &typeSelector{pos: 0, value: "foo"},
				},
				combinator: "",
				next: &complexSelector{
					pos: 4,
					sel: compoundSelector{
						pos:          4,
						typeSelector: &typeSelector{pos: 4, value: "bar"},
					},
					combinator: "",
					next: &complexSelector{
						pos: 8,
						sel: compoundSelector{
							pos:          8,
							typeSelector: &typeSelector{pos: 8, value: "spam"},
						},
					},
				},
			},
		}},
		{"foo bar > spam", []complexSelector{
			{
				sel: compoundSelector{
					typeSelector: &typeSelector{pos: 0, value: "foo"},
				},
				combinator: "",
				next: &complexSelector{
					pos: 4,
					sel: compoundSelector{
						pos:          4,
						typeSelector: &typeSelector{pos: 4, value: "bar"},
					},
					combinator: ">",
					next: &complexSelector{
						pos: 10,
						sel: compoundSelector{
							pos:          10,
							typeSelector: &typeSelector{pos: 10, value: "spam"},
						},
					},
				},
			},
		}},
		{"foo > bar", []complexSelector{
			{
				sel: compoundSelector{
					typeSelector: &typeSelector{pos: 0, value: "foo"},
				},
				combinator: ">",
				next: &complexSelector{
					pos: 6,
					sel: compoundSelector{
						pos:          6,
						typeSelector: &typeSelector{pos: 6, value: "bar"},
					},
				},
			},
		}},
		{"foo, bar", []complexSelector{
			{
				sel: compoundSelector{
					typeSelector: &typeSelector{pos: 0, value: "foo"},
				},
			},
			{
				pos: 5,
				sel: compoundSelector{
					pos:          5,
					typeSelector: &typeSelector{pos: 5, value: "bar"},
				},
			},
		}},
		{"foo, .bar", []complexSelector{

			{
				sel: compoundSelector{
					typeSelector: &typeSelector{pos: 0, value: "foo"},
				},
			},
			{
				pos: 5,
				sel: compoundSelector{
					pos:        5,
					subClasses: []subclassSelector{{pos: 5, classSelector: "bar"}},
				},
			},
		}},
		{".foo", []complexSelector{
			{
				sel: compoundSelector{
					subClasses: []subclassSelector{
						{classSelector: "foo"},
					},
				},
			},
		}},
		{"#foo", []complexSelector{
			{
				sel: compoundSelector{
					subClasses: []subclassSelector{
						{idSelector: "foo"},
					},
				},
			},
		}},
		{"foo > bar", []complexSelector{
			{
				sel: compoundSelector{
					typeSelector: &typeSelector{pos: 0, value: "foo"},
				},
				combinator: ">",
				next: &complexSelector{
					pos: 6,
					sel: compoundSelector{
						pos:          6,
						typeSelector: &typeSelector{pos: 6, value: "bar"},
					},
				},
			},
		}},
		{"foo > bar||spam", []complexSelector{
			{
				sel: compoundSelector{
					typeSelector: &typeSelector{value: "foo"},
				},
				combinator: ">",
				next: &complexSelector{
					pos: 6,
					sel: compoundSelector{
						pos:          6,
						typeSelector: &typeSelector{pos: 6, value: "bar"},
					},
					combinator: "||",
					next: &complexSelector{
						pos: 11,
						sel: compoundSelector{
							pos:          11,
							typeSelector: &typeSelector{pos: 11, value: "spam"},
						},
					},
				},
			},
		}},
		{"foo::bar", []complexSelector{
			{
				sel: compoundSelector{
					typeSelector: &typeSelector{pos: 0, value: "foo"},
					pseudoSelectors: []pseudoSelector{
						{
							element: pseudoClassSelector{pos: 4, ident: "bar"},
						},
					},
				},
			},
		}},
		{"foo::bar :spam :biz", []complexSelector{
			{
				sel: compoundSelector{
					typeSelector: &typeSelector{pos: 0, value: "foo"},
					pseudoSelectors: []pseudoSelector{
						{
							element: pseudoClassSelector{pos: 4, ident: "bar"},
							classes: []pseudoClassSelector{{pos: 9, ident: "spam"}, {pos: 15, ident: "biz"}},
						},
					},
				},
			},
		}},
		{"foo::myfunc(a, b, (c))", []complexSelector{
			{
				sel: compoundSelector{
					typeSelector: &typeSelector{pos: 0, value: "foo"},
					pseudoSelectors: []pseudoSelector{
						{
							element: pseudoClassSelector{
								pos:      4,
								function: "myfunc(",
								args: []token{
									{tokenIdent, "a", "a", 12, 0, ""},
									{tokenComma, ",", ",", 13, 0, ""},
									{tokenWhitespace, " ", " ", 14, 0, ""},
									{tokenIdent, "b", "b", 15, 0, ""},
									{tokenComma, ",", ",", 16, 0, ""},
									{tokenWhitespace, " ", " ", 17, 0, ""},
									{tokenParenOpen, "(", "(", 18, 0, ""},
									{tokenIdent, "c", "c", 19, 0, ""},
									{tokenParenClose, ")", ")", 20, 0, ""},
								},
							},
						},
					},
				},
			},
		}},
		{":nth-child(4n+3)", []complexSelector{
			{
				sel: compoundSelector{

					subClasses: []subclassSelector{
						{
							pseudoClassSelector: &pseudoClassSelector{
								function: "nth-child(",
								args: []token{
									{tokenDimension, "4n", "4", 11, tokenFlagInteger, "n"},
									{tokenNumber, "+3", "+3", 13, tokenFlagInteger, ""},
								},
							},
						},
					},
				},
			},
		}},
		{":nth-child(4n + 3)", []complexSelector{
			{
				sel: compoundSelector{

					subClasses: []subclassSelector{
						{
							pseudoClassSelector: &pseudoClassSelector{
								function: "nth-child(",
								args: []token{
									{tokenDimension, "4n", "4", 11, tokenFlagInteger, "n"},
									{tokenWhitespace, " ", " ", 13, 0, ""},
									{tokenDelim, "+", "+", 14, 0, ""},
									{tokenWhitespace, " ", " ", 15, 0, ""},
									{tokenNumber, "3", "3", 16, tokenFlagInteger, ""},
								},
							},
						},
					},
				},
			},
		}},
	}
	for _, test := range tests {
		p := newParser(test.s)
		got, err := p.parse()
		if err != nil {
			t.Errorf("parsing %q: %v", test.s, err)
			continue
		}
		if diff := cmpDiff(test.want, got); diff != "" {
			t.Errorf("parsing %q returned diff (-want +got) %s", test.s, diff)
		}
	}
}

type testMethod struct {
	name string
	fn   func(p *parser) (interface{}, error)
}

func TestSubParser(t *testing.T) {
	parsePseudoClass := testMethod{
		name: "pseudoClassSelector()",
		fn: func(p *parser) (interface{}, error) {
			return p.pseudoClassSelector()
		},
	}
	parseWQName := testMethod{
		name: "wqName()",
		fn: func(p *parser) (interface{}, error) {
			return p.wqName()
		},
	}
	parseAttrSel := testMethod{
		name: "attributeSelector()",
		fn: func(p *parser) (interface{}, error) {
			return p.attributeSelector()
		},
	}
	parseSubclassSel := testMethod{
		name: "subclassSelector()",
		fn: func(p *parser) (interface{}, error) {
			ss, ok, err := p.subclassSelector()
			if err != nil {
				return nil, err
			}
			if !ok {
				return false, nil
			}
			return ss, nil
		},
	}
	parseTypeSel := testMethod{
		name: "typeSelector()",
		fn: func(p *parser) (interface{}, error) {
			s, ok, err := p.typeSelector()
			if err != nil {
				return nil, err
			}
			if !ok {
				return false, nil
			}
			return s, nil
		},
	}

	tests := []struct {
		method     testMethod
		s          string
		want       interface{}
		wantErrPos int
	}{
		{parsePseudoClass, ":foo", &pseudoClassSelector{0, "foo", "", nil}, -1},
		{parsePseudoClass, ": foo", nil, 1}, // https://www.w3.org/TR/selectors-4/#white-space
		{parsePseudoClass, ":foo()", &pseudoClassSelector{0, "", "foo(", nil}, -1},
		{parsePseudoClass, ":foo(a)", &pseudoClassSelector{0, "", "foo(", []token{
			token{tokenIdent, "a", "a", 5, 0, ""},
		}}, -1},
		{parsePseudoClass, ":foo(a, b)", &pseudoClassSelector{0, "", "foo(", []token{
			token{tokenIdent, "a", "a", 5, 0, ""},
			token{tokenComma, ",", ",", 6, 0, ""},
			token{tokenWhitespace, " ", " ", 7, 0, ""},
			token{tokenIdent, "b", "b", 8, 0, ""},
		}}, -1},
		{parseWQName, "foo", &wqName{false, "", "foo"}, -1},
		{parseWQName, "foo|bar", &wqName{true, "foo", "bar"}, -1},
		{parseWQName, "|bar", &wqName{true, "", "bar"}, -1},
		{parseWQName, "*|bar", &wqName{true, "*", "bar"}, -1},
		{parseWQName, "foo|*", &wqName{false, "", "foo"}, -1},
		{parseWQName, "*|*", nil, 2},
		{parseWQName, "*foo", nil, 1},
		{parseWQName, "foo |bar", &wqName{false, "", "foo"}, -1}, // Whitespace ignored
		{parseWQName, "foo| bar", &wqName{false, "", "foo"}, -1}, // Whitespace ignored
		{parseTypeSel, "foo", &typeSelector{0, false, "", "foo"}, -1},
		{parseTypeSel, "foo|bar", &typeSelector{0, true, "foo", "bar"}, -1},
		{parseTypeSel, "|bar", &typeSelector{0, true, "", "bar"}, -1},
		{parseTypeSel, "*|bar", &typeSelector{0, true, "*", "bar"}, -1},
		{parseTypeSel, "foo|*", &typeSelector{0, true, "foo", "*"}, -1},
		{parseTypeSel, "*|*", &typeSelector{0, true, "*", "*"}, -1},
		{parseTypeSel, "*foo", &typeSelector{0, false, "", "*"}, -1},
		{parseTypeSel, "foo |bar", &typeSelector{0, false, "", "foo"}, -1}, // Whitespace ignored
		{parseTypeSel, "foo| bar", &typeSelector{0, false, "", "foo"}, -1}, // Whitespace ignored
		{parseAttrSel, "[foo]", &attributeSelector{
			0, &wqName{false, "", "foo"}, "", "", false,
		}, -1},
		{parseAttrSel, "[ foo = \"bar\" ]", &attributeSelector{
			0, &wqName{false, "", "foo"}, "=", "bar", false,
		}, -1},
		{parseAttrSel, "[foo=\"bar\"]", &attributeSelector{
			0, &wqName{false, "", "foo"}, "=", "bar", false,
		}, -1},
		{parseAttrSel, "[*|foo=\"bar\"]", &attributeSelector{
			0, &wqName{true, "*", "foo"}, "=", "bar", false,
		}, -1},
		{parseAttrSel, "[*|foo=bar]", &attributeSelector{
			0, &wqName{true, "*", "foo"}, "=", "bar", false,
		}, -1},
		{parseAttrSel, "[*|foo=bar i]", &attributeSelector{
			0, &wqName{true, "*", "foo"}, "=", "bar", true,
		}, -1},
		{parseAttrSel, "[foo^=bar]", &attributeSelector{
			0, &wqName{false, "", "foo"}, "^=", "bar", false,
		}, -1},
		{parseSubclassSel, "", false, -1},
		{parseSubclassSel, "#foo", &subclassSelector{idSelector: "foo"}, -1},
		{parseSubclassSel, ".foo", &subclassSelector{classSelector: "foo"}, -1},
		{parseSubclassSel, ".foo()", nil, 1},
		{parseSubclassSel, "[foo=bar]", &subclassSelector{
			attributeSelector: &attributeSelector{0, &wqName{false, "", "foo"}, "=", "bar", false},
		}, -1},
		{parseSubclassSel, ":foo", &subclassSelector{
			pseudoClassSelector: &pseudoClassSelector{0, "foo", "", nil},
		}, -1},
		{parseSubclassSel, "::foo", false, -1},
		{parseWQName, "foo", &wqName{false, "", "foo"}, -1},
		{parseWQName, "foo|bar", &wqName{true, "foo", "bar"}, -1},
		{parseWQName, "|bar", &wqName{true, "", "bar"}, -1},
		{parseWQName, "*|bar", &wqName{true, "*", "bar"}, -1},
		{parseWQName, "foo|*", &wqName{false, "", "foo"}, -1},
		{parseWQName, "*|*", nil, 2},
		{parseWQName, "*foo", nil, 1},
		{parseWQName, "foo |bar", &wqName{false, "", "foo"}, -1}, // Whitespace ignored
		{parseWQName, "foo| bar", &wqName{false, "", "foo"}, -1}, // Whitespace ignored
	}
	for _, test := range tests {
		t.Run(test.method.name+test.s, func(t *testing.T) {
			p := newParser(test.s)
			got, err := test.method.fn(p)
			if err != nil {
				if test.wantErrPos < 0 {
					t.Fatalf("parsing failed %v", err)
				}
				var perr *parseErr
				if !errors.As(err, &perr) {
					t.Fatalf("got err %v, want *parseErr", err)
				}
				if perr.t.pos != test.wantErrPos {
					t.Fatalf("got error at pos %d, want %d", perr.t.pos, test.wantErrPos)
				}
				return
			}

			if test.wantErrPos >= 0 {
				t.Fatalf("expected error at position %d", test.wantErrPos)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("got %v, want %v", got, test.want)
			}
		})
	}
}

func TestANPlusB(t *testing.T) {
	tests := []struct {
		s       string
		a       int64
		b       int64
		wantErr bool
	}{
		{"even", 2, 0, false},
		{"odd", 2, 1, false},
		{"even odd", 0, 0, true},
		{"4n", 4, 0, false},
		{"+4n", 4, 0, false},
		{"-4n", -4, 0, false},
		{"+ 4n", 0, 0, true},
		{"4n +3", 4, 3, false},
		{"4n -3", 4, -3, false},
		{"4n + 3", 4, 3, false},
		{"4n - 3", 4, -3, false},
		{"4n+3", 4, 3, false},
		{"4n-3", 4, -3, false},
		{"-n-3", -1, -3, false},
		{"-n -3", -1, -3, false},
		{"-n - 3", -1, -3, false},
		{"-n + 3", -1, 3, false},
		{"-n", -1, 0, false},
		{"4n- 3", 4, -3, false},
		{"-n- 3", -1, -3, false},
		{"+n", 1, 0, false},
		{"+n- 3", 1, -3, false},
		{"+n- -3", 0, 0, true},
	}

	for _, test := range tests {
		p := newParser(test.s)
		got, err := p.aNPlusB()
		if err != nil {
			if test.wantErr {
				continue
			}
			t.Errorf("Failed to parse string %s: %v", test.s, err)
			continue
		}
		tok, err := p.peek()
		if err != nil {
			t.Errorf("Failed to peek next token for string %s: %v", test.s, err)
			continue
		}
		if err := p.expectWhitespaceOrEOF(); err != nil {
			if test.wantErr {
				continue
			}
			t.Errorf("Parsing string, expected eof or whitespace at token %s for string %s", tok, test.s)
			continue
		}
		if test.wantErr {
			t.Errorf("Expected error parsing %s: %v", test.s, err)
			continue
		}
		if test.a != got.a || test.b != got.b {
			t.Errorf("Parsing failed for %s, got a=%d, b=%d, want a=%d, b=%d", test.s, got.a, got.b, test.a, test.b)
		}
	}
}
