package css

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func FuzzParse(f *testing.F) {
	corpus := []string{
		"*",
		"a",
		"ns|a",
		".red",
		"#demo",
		"[attr]",
		"[attr=value]",
		"[herf~=foo]",
		"[herf|=foo]",
		"[herf^=foo]",
		"[herf$=foo]",
		"[herf*=foo]",
		"[herf=foo i]",
		"h1 a",
		"h1, a",
		"h1 > a",
		"h1 ~ a",
		"h1 + a",
		"h1:empty",
		"h1:first-child",
		"h1:first-of-type",
		"h1:last-child",
		"h1:last-of-type",
		"h1:only-child",
		"h1:only-of-type",
		"h1:root",
		"h1:nth-child(1n + 3)",
		"h1:nth-child(odd)",
		"h1:nth-child(even)",
		"h1:nth-child(1n)",
		"h1:nth-child(3)",
		"h1:nth-child(+3)",
		"h1:last-child(1n + 3)",
		"h1:last-of-type(1n + 3)",
		"h1:nth-of-type(1n + 3)",
	}
	for _, s := range corpus {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		Parse(s)
	})
}

func FuzzSelector(f *testing.F) {
	for _, test := range selectorTests {
		f.Add(test.sel, test.in)
	}
	f.Fuzz(func(t *testing.T, sel, in string) {
		s, err := Parse(sel)
		if err != nil {
			t.Skip()
		}
		root, err := html.Parse(strings.NewReader(in))
		if err != nil {
			t.Skip()
		}
		s.Select(root)
	})
}
