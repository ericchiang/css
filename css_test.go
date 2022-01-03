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
	pClassSelector := testMethod{
		name: "classSelector()",
		fn: func(p *parser) (interface{}, error) {
			return p.classSelector()
		},
	}
	pPseudoClass := testMethod{
		name: "pseudoClassSelector()",
		fn: func(p *parser) (interface{}, error) {
			return p.pseudoClassSelector()
		},
	}
	pWQName := testMethod{
		name: "wqName()",
		fn: func(p *parser) (interface{}, error) {
			return p.wqName()
		},
	}

	tests := []struct {
		method     testMethod
		s          string
		want       interface{}
		wantErrPos int
	}{
		{pClassSelector, ".foo", &classSelector{"foo"}, -1},
		{pClassSelector, ".bar()", nil, 1},
		{pClassSelector, "foo", nil, 0},
		{pPseudoClass, ":foo", &pseudoClassSelector{"foo", "", nil}, -1},
		{pPseudoClass, ": foo", nil, 1}, // https://www.w3.org/TR/selectors-4/#white-space
		{pPseudoClass, ":foo()", &pseudoClassSelector{"", "foo(", nil}, -1},
		{pPseudoClass, ":foo(a)", &pseudoClassSelector{"", "foo(", []token{
			token{tokenIdent, "a", 5},
		}}, -1},
		{pPseudoClass, ":foo(a, b)", &pseudoClassSelector{"", "foo(", []token{
			token{tokenIdent, "a", 5},
			token{tokenComma, ",", 6},
			token{tokenWhitespace, " ", 7},
			token{tokenIdent, "b", 8},
		}}, -1},
		{pWQName, "foo", &wqName{false, "", "foo"}, -1},
		{pWQName, "foo|bar", &wqName{true, "foo", "bar"}, -1},
		{pWQName, "|bar", &wqName{true, "", "bar"}, -1},
		{pWQName, "*|bar", &wqName{true, "*", "bar"}, -1},
		{pWQName, "foo|*", &wqName{false, "", "foo"}, -1},
		{pWQName, "*foo", nil, 1},
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
