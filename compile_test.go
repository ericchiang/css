package css

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func compErr(s string, err error) error {
	if e, ok := err.(*SyntaxError); ok {
		var b bytes.Buffer
		fmt.Fprintf(&b, "%s\n%s\n", err, s)
		fmt.Fprintf(&b, strings.Repeat(" ", e.Offset)+"^\n")
		return errors.New(b.String())
	}
	return err
}

func TestCompile(t *testing.T) {
	tests := []struct {
		in   string
		expr string
		want []string
	}{
		{
			`<div><p>foo</p><span><p>bar</p></span></div>`,
			"span > p, p",
			[]string{`<p>bar</p>`, `<p>foo</p>`, `<p>bar</p>`},
		},
	}
	for i, tt := range tests {
		sel, err := Compile(tt.expr)
		if err != nil {
			t.Errorf("case=%d: failed to compile %v", i, compErr(tt.expr, err))
			continue
		}
		runTest(t, i, tt.in, sel, tt.want)
	}
}

func TestCompileError(t *testing.T) {
	tests := []string{
		"",
		"*foo",
	}
	for i, tt := range tests {
		if _, err := Compile(tt); err == nil {
			t.Errorf("case=%d: expected to fail to compile %s", i, strconv.Quote(tt))
		}
	}
}

func TestCompileSelector(t *testing.T) {
	tests := []struct {
		in   string
		expr string
		want []string
	}{
		{
			`<span>This is not red.</span>
			<p>Here is a paragraph.</p>
			<code>Here is some code.</code>
			<span>And here is a span.</span>
			<span>And another span.</span>`,
			"p ~ span",
			[]string{
				`<span>And here is a span.</span>`,
				`<span>And another span.</span>`,
			},
		},
		{
			`<div><p>foo</p><span><p>bar</p></span></div>`,
			"p",
			[]string{`<p>foo</p>`, `<p>bar</p>`},
		},
		{
			`<div><p>foo</p><span><p>bar</p></span></div>`,
			"div > p",
			[]string{`<p>foo</p>`},
		},
		{
			`<div><p>foo</p><span><p>bar</p></span></div>`,
			"span > p",
			[]string{`<p>bar</p>`},
		},
		{
			`<div><p>foo</p><span><p>bar</p></span></div>`,
			"span p",
			[]string{`<p>bar</p>`},
		},
		{
			`<div><p>foo</p><span><p>bar</p></span></div>`,
			"div p",
			[]string{`<p>foo</p>`, `<p>bar</p>`},
		},
		{
			`<div><p>foo</p><span><p>bar</p></span></div>`,
			"div div",
			[]string{},
		},
		{
			`<div><p>foo</p><span><p>bar</p></span></div>`,
			"div *",
			[]string{`<p>foo</p>`, `<span><p>bar</p></span>`},
		},
		{
			`<div><p class="hi">foo</p><span><p class="hi">bar</p></span></div>`,
			"div .hi",
			[]string{`<p class="hi">foo</p>`, `<p class="hi">bar</p>`},
		},
		{
			`<p><a id="foo"></a></p>`,
			"p :empty",
			[]string{`<a id="foo"></a>`},
		},
	}
	for i, tt := range tests {
		l, err := newLexer(tt.expr)
		if err != nil {
			t.Errorf("case=%d: could not create lexer %v",i,  err)
			continue
		}
		go l.run()
		c := newCompiler(l)
		sel, err := c.compileSelector()
		if err != nil {
			t.Errorf("case=%d: compilation failed %v", i, compErr(tt.expr, err))
			continue
		}
		if c.peek().typ != typeEOF {
			t.Errorf("case=%d: did not parse entire input", i)
		}
		runTest(t, i, tt.in, sel, tt.want)
	}
}

func TestCompileSimpleSelectorSeq(t *testing.T) {
	tests := []struct {
		in   string
		expr string
		want []string
	}{
		{
			`<p><a></a></p>`,
			"a",
			[]string{"<a></a>"},
		},
		{
			`<p><a class="foo"></a></p>`,
			"a.foo",
			[]string{`<a class="foo"></a>`},
		},
		{
			`<p><a></a></p>`,
			"a.foo",
			[]string{},
		},
		{
			`<p><a id="foo"></a></p>`,
			"a#foo",
			[]string{`<a id="foo"></a>`},
		},
		{
			`<p><a id="foo"></a></p>`,
			"#foo",
			[]string{`<a id="foo"></a>`},
		},
		{
			`<p><a id="foo"></a></p>`,
			"a[id=foo]",
			[]string{`<a id="foo"></a>`},
		},
		{
			`<p><a id="foo"></a></p>`,
			"p:empty",
			[]string{},
		},
		{
			`<p><a id="1"></a><a id="2"></a><a id="3"></a><a id="4"></a></p>`,
			"a:nth-child(odd)",
			[]string{`<a id="1"></a>`, `<a id="3"></a>`},
		},
	}
	for i, tt := range tests {
		l, err := newLexer(tt.expr)
		if err != nil {
			t.Errorf("case=%d: could not create lexer %v", i, err)
			continue
		}
		go l.run()
		c := newCompiler(l)
		sel, err := c.compileSimpleSelectorSeq()
		if err != nil {
			t.Errorf("case=%d: compilation failed %v",i,  err)
			continue
		}
		if c.peek().typ != typeEOF {
			t.Errorf("case=%d: did not parse entire input", i)
		}
		runTest(t, i, tt.in, sel, tt.want)
	}
}

func TestCompileAttr(t *testing.T) {
	tests := []struct {
		in   string
		expr string
		want []string
	}{
		{
			`<p><a id="foo"></a></p>`,
			"[id=foo]",
			[]string{`<a id="foo"></a>`},
		},
		{
			`<p><a id="foo"></a></p>`,
			"[id = 'foo']",
			[]string{`<a id="foo"></a>`},
		},
		{
			`<p><a id="foo"></a></p>`,
			`[id="foo"]`,
			[]string{`<a id="foo"></a>`},
		},
		{
			`<p><a id="hello-world"></a><a id="helloworld"></a></p>`,
			`[id|="hello"]`,
			[]string{`<a id="hello-world"></a>`},
		},
		{
			`<p><a id="hello-world"></a><a id="worldhello"></a></p>`,
			`[id^="hello"]`,
			[]string{`<a id="hello-world"></a>`},
		},
		{
			`<p><a id="hello-world"></a><a id="worldhello"></a></p>`,
			`[id$="hello"]`,
			[]string{`<a id="worldhello"></a>`},
		},
		{
			`<p><a id="hello-world"></a><a id="worldhello"></a></p>`,
			`[id*="hello"]`,
			[]string{`<a id="hello-world"></a>`, `<a id="worldhello"></a>`},
		},
		{
			`<p><a id="hello world"></a><a id="hello-world"></a></p>`,
			`[id~="hello"]`,
			[]string{`<a id="hello world"></a>`},
		},
	}
	for i, tt := range tests {
		l, err := newLexer(tt.expr)
		if err != nil {
			t.Errorf("case=%d: could not create lexer %v", i,err)
			continue
		}
		go l.run()
		c := newCompiler(l)
		m, err := c.compileAttr()
		if err != nil {
			t.Errorf("case=%d: compilation failed %v", i, err)
			continue
		}
		if c.peek().typ != typeEOF {
			t.Errorf("case=%d: did not parse entire input", i)
		}
		runTest(t, i, tt.in, selectorSequence{[]matcher{m}}, tt.want)
	}
}

func TestParthNthArgs(t *testing.T) {
	tests := []struct {
		expr string
		a, b int
	}{
		{"even", 2, 0},
		{"odd", 2, 1},
		{"2n+1", 2, 1},
		{"-2n-1", -2, -1},
		{"2n", 2, 0},
		{"+2n", 2, 0},
		{"-2n", -2, 0},
		{"4", 0, 4},
		{"4n - 3", 4, -3},
	}
	for i, tt := range tests {
		l, err := newLexer(tt.expr)
		if err != nil {
			t.Errorf("case=%d: could not create lexer %v",i,  err)
			continue
		}
		go l.run()
		c := newCompiler(l)
		a, b, err := c.parseNthArgs()
		if err != nil {
			t.Errorf("case='%s': parse failed %v", tt.expr, err)
			continue
		}
		if c.peek().typ != typeEOF {
			t.Errorf("case='%s': did not parse entire input", tt.expr)
		}
		if tt.a != a || tt.b != b {
			t.Errorf("case='%s': want=(a=%d, b=%d), got=(a=%d, b=%d)", tt.expr, tt.a, tt.b, a, b)
		}
	}
}

func TestParseNth(t *testing.T) {
	tests := []struct {
		in     string
		a, b   int
		bFound bool
		ok     bool
	}{
		{"9n", 9, 0, false, true},
		{"-2n+2", -2, 2, true, true},
		{"91n3n", 0, 0, false, false},
	}

	for _, tt := range tests {
		a, b, bFound, err := parseNth(tt.in)
		if err != nil {
			if tt.ok {
				t.Errorf("case=%q: failed to parse", tt.in)
			}
			continue
		}
		if !tt.ok {
			t.Errorf("case=%q: expected parsing error", tt.in)
			continue
		}
		if a != tt.a {
			t.Errorf("case=%q: want (a=%d), got (a=%d)", tt.in, tt.a, a)
		}
		if bFound != tt.bFound {
			t.Errorf("case=%q: want (bFound=%t), got (bFound=%t)", tt.in, tt.bFound, bFound)
		} else if b != tt.b {
			t.Errorf("case=%q: want (b=%d), got (b=%d)", tt.in, tt.b, b)
		}
	}
}

func TestNthRegexp(t *testing.T) {
	tests := []struct {
		in string
		a  string
		b  string
		ok bool
	}{
		{"-2n-2", "-2", "-2", true},
		{"-2n+2", "-2", "+2", true},
		{"-80n+100", "-80", "+100", true},
		{"+80n+100", "+80", "+100", true},
		{"80n+100", "80", "+100", true},
		{" 80n+100 ", "", "", false},
		{"80n+100 ", "", "", false},
		{" 80n+100", "", "", false},
		{"-23n", "-23", "", true},
		{"foobar", "", "", false},
	}

	for _, tt := range tests {
		submatch := nthRegexp.FindStringSubmatch(tt.in)
		if submatch == nil {
			if tt.ok {
				t.Errorf("case=%q: failed to parse", tt.in)
			}
			continue
		}
		if !tt.ok {
			t.Errorf("case=%q: expected to fail to parse", tt.in)
			continue
		}
		if len(submatch) != 3 {
			t.Errorf("case=%q: expected len(submatch)=3, got=%d", tt.in, len(submatch))
			continue
		}
		if got := submatch[1]; got != tt.a {
			t.Errorf("case=%q: expected a=%q, got a=%q", tt.in, tt.a, got)
		}
		if got := submatch[2]; got != tt.b {
			t.Errorf("case=%q: expected b=%q, got b=%q", tt.in, tt.b, got)
		}
	}
}
