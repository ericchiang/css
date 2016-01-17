package css

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

type selInterface interface {
	Select(n *html.Node) []*html.Node
}

func runTest(t *testing.T, testNum int, in string, sel selInterface, want []string) {
	node, err := html.Parse(strings.NewReader(in))
	if err != nil {
		t.Errorf("case=%d: failed to parse HTML %v", testNum, err)
		return
	}
	selected := sel.Select(node)
	if len(selected) != len(want) {
		t.Errorf("case=%d: want num selected=%d, got=%d", testNum, len(want), len(selected))
	}
	for i := 0; i < len(selected) && i < len(want); i++ {
		var b bytes.Buffer
		if err := html.Render(&b, selected[i]); err != nil {
			t.Errorf("case=%d ele=%d: failed to render: %v", testNum, i, err)
			continue
		}
		if got := b.String(); got != want[i] {
			t.Errorf("case=%d ele=%d: want=%s, got=%s", testNum, i, strconv.Quote(want[i]), strconv.Quote(got))
		}
	}
}

func TestSelector(t *testing.T) {
	tests := []struct {
		in   string
		want []string
		sel  selector
	}{
		{
			`<span>This is not red.</span>
			<p>Here is a paragraph.</p>
			<code>Here is some code.</code>
			<span>And here is a span.</span>
			<span>And another span.</span>`,
			[]string{
				`<span>And here is a span.</span>`,
				`<span>And another span.</span>`,
			},
			// p ~ span
			selector{
				selSeq: selectorSequence{
					matchers: []matcher{typeSelector{"p"}},
				},
				combs: []combinatorSelector{
					combinatorSelector{
						combinator: typeTilde,
						selSeq: selectorSequence{
							matchers: []matcher{typeSelector{"span"}},
						},
					},
				},
			},
		},
		{
			`<div><p>foo</p><span><p>bar</p></span></div>`,
			[]string{`<p>foo</p>`, `<span><p>bar</p></span>`},
			// div *
			selector{
				selSeq: selectorSequence{
					matchers: []matcher{typeSelector{"div"}},
				},
				combs: []combinatorSelector{
					combinatorSelector{
						combinator: typeSpace,
						selSeq: selectorSequence{
							matchers: []matcher{universal{}},
						},
					},
				},
			},
		},
	}

	for i, tt := range tests {
		runTest(t, i, tt.in, tt.sel, tt.want)
	}
}

func TestMatcher(t *testing.T) {
	tests := []struct {
		in   string
		want []string
		m    []matcher
	}{
		{
			`<p><a id="foo"></a></p>`,
			[]string{`<a id="foo"></a>`},
			[]matcher{attrMatcher{"id", "foo"}},
		},
		{
			`<p><a id="bar"></a></p>`,
			[]string{},
			[]matcher{attrMatcher{"id", "foo"}},
		},
		{
			`<p><a class="bar"></a></p>`,
			[]string{`<a class="bar"></a>`},
			[]matcher{attrMatcher{"class", "bar"}},
		},
		{
			`<p><a id="foo"></a><a></a></p>`,
			[]string{`<a id="foo"></a>`, `<a></a>`},
			[]matcher{typeSelector{"a"}},
		},
		{
			// non-standard HTML
			`<p><foobar></foobar></p>`,
			[]string{`<foobar></foobar>`},
			[]matcher{typeSelector{"foobar"}},
		},
		{
			`<p><a id="foo"></a><a></a></p>`,
			[]string{`<a id="foo"></a>`},
			[]matcher{
				typeSelector{"a"},
				matcherFunc(firstChild),
			},
		},
		{
			`<div class="box"><!-- I will be selected --></div>
			<div class="box">I will be not be selected</div>
			<div class="box">
			    <!-- I will not be selected because of the whitespace around this comment -->
			</div>
			`,
			[]string{`<div class="box"><!-- I will be selected --></div>`},
			[]matcher{
				attrMatcher{"class", "box"},
				matcherFunc(empty),
			},
		},
	}

	for i, tt := range tests {
		runTest(t, i, tt.in, selectorSequence{tt.m}, tt.want)
	}
}

func TestPosMatches(t *testing.T) {
	tests := []struct {
		a, b, pos int
		want      bool
	}{
		{2, 4, 3, true},
		{2, 4, 5, true},
		{2, 4, 13, true},
		{-2, 7, 0, true},
		{-2, 7, 6, true},
		{-2, 7, 7, false},
		{0, 7, 6, true},
	}
	for i, tt := range tests {
		if got := posMatches(tt.a, tt.b, tt.pos); got != tt.want {
			t.Errorf("case=%d (a=%d, b=%d, pos=%d): want=%t, got=%t", i, tt.a, tt.b, tt.pos, tt.want, got)
		}
	}
}
