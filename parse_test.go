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
							element: pseudoClassSelector{ident: "bar"},
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
							element: pseudoClassSelector{ident: "bar"},
							classes: []pseudoClassSelector{{ident: "spam"}, {ident: "biz"}},
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
								function: "myfunc(",
								args: []token{
									{tokenIdent, "a", "a", 12},
									{tokenComma, ",", ",", 13},
									{tokenWhitespace, " ", " ", 14},
									{tokenIdent, "b", "b", 15},
									{tokenComma, ",", ",", 16},
									{tokenWhitespace, " ", " ", 17},
									{tokenParenOpen, "(", "(", 18},
									{tokenIdent, "c", "c", 19},
									{tokenParenClose, ")", ")", 20},
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
		{parsePseudoClass, ":foo", &pseudoClassSelector{"foo", "", nil}, -1},
		{parsePseudoClass, ": foo", nil, 1}, // https://www.w3.org/TR/selectors-4/#white-space
		{parsePseudoClass, ":foo()", &pseudoClassSelector{"", "foo(", nil}, -1},
		{parsePseudoClass, ":foo(a)", &pseudoClassSelector{"", "foo(", []token{
			token{tokenIdent, "a", "a", 5},
		}}, -1},
		{parsePseudoClass, ":foo(a, b)", &pseudoClassSelector{"", "foo(", []token{
			token{tokenIdent, "a", "a", 5},
			token{tokenComma, ",", ",", 6},
			token{tokenWhitespace, " ", " ", 7},
			token{tokenIdent, "b", "b", 8},
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
		{parseTypeSel, "*foo", nil, 1},
		{parseTypeSel, "foo |bar", &typeSelector{0, false, "", "foo"}, -1}, // Whitespace ignored
		{parseTypeSel, "foo| bar", &typeSelector{0, false, "", "foo"}, -1}, // Whitespace ignored
		{parseAttrSel, "[foo]", &attributeSelector{
			&wqName{false, "", "foo"}, "", "", false,
		}, -1},
		{parseAttrSel, "[ foo = \"bar\" ]", &attributeSelector{
			&wqName{false, "", "foo"}, "", "bar", false,
		}, -1},
		{parseAttrSel, "[foo=\"bar\"]", &attributeSelector{
			&wqName{false, "", "foo"}, "", "bar", false,
		}, -1},
		{parseAttrSel, "[*|foo=\"bar\"]", &attributeSelector{
			&wqName{true, "*", "foo"}, "", "bar", false,
		}, -1},
		{parseAttrSel, "[*|foo=bar]", &attributeSelector{
			&wqName{true, "*", "foo"}, "", "bar", false,
		}, -1},
		{parseAttrSel, "[*|foo=bar i]", &attributeSelector{
			&wqName{true, "*", "foo"}, "", "bar", true,
		}, -1},
		{parseAttrSel, "[foo^=bar]", &attributeSelector{
			&wqName{false, "", "foo"}, "^", "bar", false,
		}, -1},
		{parseSubclassSel, "", false, -1},
		{parseSubclassSel, "#foo", &subclassSelector{idSelector: "foo"}, -1},
		{parseSubclassSel, ".foo", &subclassSelector{classSelector: "foo"}, -1},
		{parseSubclassSel, ".foo()", nil, 1},
		{parseSubclassSel, "[foo=bar]", &subclassSelector{
			attributeSelector: &attributeSelector{&wqName{false, "", "foo"}, "", "bar", false},
		}, -1},
		{parseSubclassSel, ":foo", &subclassSelector{
			pseudoClassSelector: &pseudoClassSelector{"foo", "", nil},
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
