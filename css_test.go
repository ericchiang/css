package css

import (
	"errors"
	"reflect"
	"testing"
)

type testMethod struct {
	name string
	fn   func(p *parser) (interface{}, error)
}

func TestParser(t *testing.T) {
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
			token{tokenIdent, "a", 5},
		}}, -1},
		{parsePseudoClass, ":foo(a, b)", &pseudoClassSelector{"", "foo(", []token{
			token{tokenIdent, "a", 5},
			token{tokenComma, ",", 6},
			token{tokenWhitespace, " ", 7},
			token{tokenIdent, "b", 8},
		}}, -1},
		{parseWQName, "foo", &wqName{false, "", "foo"}, -1},
		{parseWQName, "foo|bar", &wqName{true, "foo", "bar"}, -1},
		{parseWQName, "|bar", &wqName{true, "", "bar"}, -1},
		{parseWQName, "*|bar", &wqName{true, "*", "bar"}, -1},
		{parseWQName, "foo|*", &wqName{false, "", "foo"}, -1},
		{parseWQName, "*foo", nil, 1},
		{parseWQName, "foo |bar", &wqName{false, "", "foo"}, -1}, // Whitespace ignored
		{parseWQName, "foo| bar", &wqName{false, "", "foo"}, -1}, // Whitespace ignored
		{parseAttrSel, "[foo]", &attributeSelector{
			&wqName{false, "", "foo"}, "", "", false,
		}, -1},
		{parseAttrSel, "[ foo = \"bar\" ]", &attributeSelector{
			&wqName{false, "", "foo"}, "", "\"bar\"", false,
		}, -1},
		{parseAttrSel, "[foo=\"bar\"]", &attributeSelector{
			&wqName{false, "", "foo"}, "", "\"bar\"", false,
		}, -1},
		{parseAttrSel, "[*|foo=\"bar\"]", &attributeSelector{
			&wqName{true, "*", "foo"}, "", "\"bar\"", false,
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
