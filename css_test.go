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
			[]string{
				`<div><div></div></div>`,
				`<div></div>`,
			},
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
		{
			"a",
			`<div><svg xmlns="http://www.w3.org/2000/svg"><a class="foo"></a></svg></div>`,
			[]string{`<a class="foo"></a>`},
		},
		{
			"*|a",
			`<div><svg xmlns="http://www.w3.org/2000/svg"><a class="foo"></a></svg></div>`,
			[]string{`<a class="foo"></a>`},
		},
		{
			"svg|a",
			`<div><svg xmlns="http://www.w3.org/2000/svg"><a class="foo"></a></svg></div>`,
			[]string{`<a class="foo"></a>`},
		},
		{
			"|a",
			`<div><svg xmlns="http://www.w3.org/2000/svg"><a class="foo"></a></svg></div>`,
			[]string{},
		},
		{
			"other|a",
			`<div><svg xmlns="http://www.w3.org/2000/svg"><a class="foo"></a></svg></div>`,
			[]string{},
		},
		{
			"svg|*",
			`<div><svg xmlns="http://www.w3.org/2000/svg"><a class="foo"></a></svg></div>`,
			[]string{
				`<svg xmlns="http://www.w3.org/2000/svg"><a class="foo"></a></svg>`,
				`<a class="foo"></a>`,
			},
		},
		{
			"div[class=foo]",
			`<h1><h2 class="foo"></h2><div class="foo"></div><div id="foo"></div></h1>`,
			[]string{
				`<div class="foo"></div>`,
			},
		},
		{
			"div[class*=o]",
			`<h1><h2 class="foo"></h2><div class="foo"></div><div id="foo"></div></h1>`,
			[]string{
				`<div class="foo"></div>`,
			},
		},
		{
			"div[class~=foo]",
			`<h1><h2 class="foo"></h2><div class="bar foo"></div><div id="foo"></div></h1>`,
			[]string{
				`<div class="bar foo"></div>`,
			},
		},
		{
			"div[class|=foo]",
			`<h1><div class="foo bar"></div><div class="foo"></div><div class="foo-bar"></div></h1>`,
			[]string{
				`<div class="foo"></div>`,
				`<div class="foo-bar"></div>`,
			},
		},
		{
			"div[class^=foo]",
			`<h1><div class="bar foo"></div><div class="foo"></div><div class="foo-bar"></div></h1>`,
			[]string{
				`<div class="foo"></div>`,
				`<div class="foo-bar"></div>`,
			},
		},
		{
			"div[class$=foo]",
			`<h1><div class="bar foo"></div><div class="foo"></div><div class="foo-bar"></div></h1>`,
			[]string{
				`<div class="bar foo"></div>`,
				`<div class="foo"></div>`,
			},
		},
		{
			"div[class]",
			`<h1><div class="bar foo"></div><div class="foo"></div><div class="foo-bar"></div></h1>`,
			[]string{
				`<div class="bar foo"></div>`,
				`<div class="foo"></div>`,
				`<div class="foo-bar"></div>`,
			},
		},
		{
			"div[class^=foO i]",
			`<h1><div class="bar foo"></div><div class="fOo"></div><div class="Foo-bar"></div></h1>`,
			[]string{
				`<div class="fOo"></div>`,
				`<div class="Foo-bar"></div>`,
			},
		},
		{
			"div a",
			`
			<h1>
				<div>
					<a href="http://bar"></a>
				</div>
				<div>
					<div>
						<a href="http://foo"></a>
					</div>
				</div>
				<a href="http://spam"></a>
			</h1>
			`,
			[]string{
				`<a href="http://bar"></a>`,
				`<a href="http://foo"></a>`,
				`<a href="http://foo"></a>`,
			},
		},
		{
			"div > a",
			`
			<h1>
				<div>
					<a href="http://bar"></a>
				</div>
				<div>
					<div>
						<a href="http://foo"></a>
					</div>
				</div>
				<a href="http://spam"></a>
			</h1>
			`,
			[]string{
				`<a href="http://bar"></a>`,
				`<a href="http://foo"></a>`,
			},
		},
		{
			"div + a",
			`
			<h1>
				<div>
					<a href="http://bar"></a>
				</div>
				<a href="http://spam"></a>
				<p></p>
				<a href="http://foo"></a>
			</h1>
			`,
			[]string{
				`<a href="http://spam"></a>`,
			},
		},
		{
			"div ~ a",
			`
			<h1>
				<div>
					<a href="http://bar"></a>
				</div>
				<a href="http://spam"></a>
				<p></p>
				<a href="http://foo"></a>
			</h1>
			`,
			[]string{
				`<a href="http://spam"></a>`,
				`<a href="http://foo"></a>`,
			},
		},
		{
			"body p em", // https://github.com/ericchiang/css/issues/7
			`
				<html>
					<body>
						<p>
							<em></em>
						</p>
					</body>
				</html>
			`,
			[]string{"<em></em>"},
		},
		{
			"div:empty",
			`
				<div class="foo"><p></p></div>
				<div class="bar">  </div>
			`,
			[]string{`<div class="bar">  </div>`},
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

		// Re-render test case in case the parser is interpeting it differently than
		// we expect.
		b := &bytes.Buffer{}
		if err := html.Render(b, root); err != nil {
			t.Errorf("Re-rendering input %s: %v", test.in, err)
			continue
		}
		in := b.String()

		got := []string{}
		for _, n := range s.Select(root) {
			b := &bytes.Buffer{}
			if err := html.Render(b, n); err != nil {
				t.Errorf("Failed to render result of selecting %q from %s: %v", test.sel, in, err)
				continue
			}
			got = append(got, b.String())
		}
		if diff := cmp.Diff(test.want, got); diff != "" {
			t.Errorf("Selecting %q from %s returned diff (-want, +got): %s", test.sel, in, diff)
		}
	}
}
