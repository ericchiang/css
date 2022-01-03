package css

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/net/html"
)

func TestSelector(t *testing.T) {
	tests := []struct {
		sel  string
		in   string
		want []string
	}{
		{
			"a",
			`<h1><a></a></h1>`,
			[]string{`<a></a>`},
		},
		{
			"div",
			`<h1><div><div></div></div></h1>`,
			[]string{`<div><div></div></div>`},
		},
		{
			"div",
			`<h1><div></div><div></div></h1>`,
			[]string{`<div></div>`, `<div></div>`},
		},
		{
			".foo",
			`<h1><h2 class="foo"></h2><div class="foo"></div><div id="foo"></div></h1>`,
			[]string{
				`<h2 class="foo"></h2>`,
				`<div class="foo"></div>`,
			},
		},
		{
			"div.foo",
			`<h1><h2 class="foo"></h2><div class="foo"></div><div id="foo"></div></h1>`,
			[]string{`<div class="foo"></div>`},
		},
		{
			"#foo",
			`<h1><h2 class="foo"></h2><div class="foo"></div><div id="foo"></div></h1>`,
			[]string{`<div id="foo"></div>`},
		},
		{
			"div#foo",
			`<h1><h2 class="foo"></h2><div class="foo"></div><div id="foo"></div></h1>`,
			[]string{`<div id="foo"></div>`},
		},
	}
	for _, test := range tests {
		s, err := Parse(test.sel)
		if err != nil {
			t.Errorf("Parse(%q) failed %v", test.sel, err)
			continue
		}
		root, err := html.Parse(strings.NewReader(test.in))
		if err != nil {
			t.Errorf("html.Parse(%q) failed %v", test.in, err)
			continue
		}

		got := []string{}
		for _, n := range s.Select(root) {
			b := &bytes.Buffer{}
			if err := html.Render(b, n); err != nil {
				t.Errorf("Failed to render result of selecting %q from %q: %v", test.sel, test.in, err)
				continue
			}
			got = append(got, b.String())
		}
		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("Selecting %q from %q returned diff (-want, +got): %s", test.sel, test.in, diff)
		}
	}
}
