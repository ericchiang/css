package css

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/net/html"
)

func (s *Selector) String() string {
	var b strings.Builder
	formatValue(reflect.ValueOf(s), &b, "")
	return b.String()
}

func formatValue(v reflect.Value, b *strings.Builder, ident string) {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fmt.Fprintf(b, "%d", v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		fmt.Fprintf(b, "%d", v.Uint())
	case reflect.Float32, reflect.Float64:
		fmt.Fprintf(b, "%f", v.Float())
	case reflect.Bool:
		fmt.Fprintf(b, "%t", v.Bool())
	case reflect.Array, reflect.Slice:
		if v.IsNil() {
			b.WriteString("[]")
			return
		}
		fmt.Fprintf(b, "[\n")
		for i := 0; i < v.Len(); i++ {
			b.WriteString(ident)
			b.WriteString("\t")
			formatValue(v.Index(i), b, ident+"\t")
			fmt.Fprintf(b, ",\n")
		}
		b.WriteString(ident)
		b.WriteString("]")
	case reflect.Func:
		if v.IsNil() {
			b.WriteString("<nil>")
			return
		}
		fmt.Fprintf(b, "<func()>")
	case reflect.Interface:
		if v.IsNil() {
			b.WriteString("<nil>")
			return
		}
		formatValue(v.Elem(), b, ident)
	case reflect.Map:
		if v.IsNil() {
			b.WriteString("<nil>")
			return
		}
		iter := v.MapRange()
		fmt.Fprintf(b, "{\n")
		for iter.Next() {
			b.WriteString(ident)
			formatValue(iter.Key(), b, ident+"\n")
			fmt.Fprintf(b, ", ")
			formatValue(iter.Value(), b, ident)
		}
		fmt.Fprintf(b, "}")
	case reflect.Ptr:
		if v.IsNil() {
			b.WriteString("<nil>")
			return
		}
		fmt.Fprintf(b, "*")
		formatValue(reflect.Indirect(v), b, ident)
	case reflect.String:
		fmt.Fprintf(b, "%q", v.String())
	case reflect.Struct:
		t := v.Type()
		fmt.Fprintf(b, "%s{\n", t.Name())
		for i := 0; i < v.NumField(); i++ {
			b.WriteString(ident + "\t")
			b.WriteString(t.Field(i).Name)
			b.WriteString(": ")
			formatValue(v.Field(i), b, ident+"\t")
			b.WriteString(",\n")
		}
		b.WriteString(ident)
		b.WriteString("}")
	}
}

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
			"body",
			`<h1><a></a></h1>`,
			[]string{`<body><h1><a></a></h1></body>`},
		},
		{
			"body *",
			`<h1><a></a></h1>`,
			[]string{`<h1><a></a></h1>`, `<a></a>`},
		},
		{
			"body > *",
			`<h1><a></a></h1>`,
			[]string{`<h1><a></a></h1>`},
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
		{
			":root",
			`<html><head></head><body></body></html>`,
			[]string{`<html><head></head><body></body></html>`},
		},
		{
			"div:first-child",
			`
			<p></p>
			<div>
				<div class="foo"><p></p></div>
				<div class="bar"><div class="spam"></div></div>
			</div>
			<p></p>
			`,
			[]string{
				`<div class="foo"><p></p></div>`,
				`<div class="spam"></div>`,
			},
		},
		{
			"div:last-child",
			`
			<p></p>
			<div>
				<div class="foo"><p></p></div>
				<div class="bar"><div class="spam"></div></div>
			</div>
			<p></p>
			`,
			[]string{
				`<div class="bar"><div class="spam"></div></div>`,
				`<div class="spam"></div>`,
			},
		},
		{
			"div:only-child",
			`
			<p></p>
			<div>
				<div class="foo"><p></p></div>
				<div class="bar"><div class="spam"></div></div>
			</div>
			<p></p>
			`,
			[]string{
				`<div class="spam"></div>`,
			},
		},
		{
			".test:first-of-type",
			`
			<p></p>
			<div>
				<p class="test" id="foo"></p>
				<div class="test" id="foo"></div>
				<div class="test" id="bar"></div>
				<p class="test" id="bar"></p>
				<h1 class="test" id="bar"></h1>
			</div>
			<p></p>
			`,
			[]string{
				`<p class="test" id="foo"></p>`,
				`<div class="test" id="foo"></div>`,
				`<h1 class="test" id="bar"></h1>`,
			},
		},
		{
			".test:last-of-type",
			`
			<p></p>
			<div>
				<p class="test" id="foo"></p>
				<div class="test" id="foo"></div>
				<div class="test" id="bar"></div>
				<p class="test" id="bar"></p>
				<h1 class="test" id="bar"></h1>
			</div>
			<p></p>
			`,
			[]string{
				`<div class="test" id="bar"></div>`,
				`<p class="test" id="bar"></p>`,
				`<h1 class="test" id="bar"></h1>`,
			},
		},
		{
			".test:only-of-type",
			`
			<p></p>
			<div>
				<p class="test" id="foo"></p>
				<div class="test" id="foo"></div>
				<div class="test" id="bar"></div>
				<p class="test" id="bar"></p>
				<h1 class="test" id="bar"></h1>
			</div>
			<p></p>
			`,
			[]string{
				`<h1 class="test" id="bar"></h1>`,
			},
		},
		{
			"li:nth-child(2)",
			`
			<ul>
				<li>1</li>
				<li>2</li>
				<li>3</li>
				<li>4</li>
				<li>5</li>
				<li>6</li>
				<li>7</li>
				<li>8</li>
			</ul>
			`,
			[]string{
				`<li>2</li>`,
			},
		},
		{
			"li:nth-child(1n+2)",
			`
			<ul>
				<li>1</li>
				<li>2</li>
				<li>3</li>
				<li>4</li>
				<li>5</li>
				<li>6</li>
				<li>7</li>
				<li>8</li>
			</ul>
			`,
			[]string{
				`<li>2</li>`,
				`<li>3</li>`,
				`<li>4</li>`,
				`<li>5</li>`,
				`<li>6</li>`,
				`<li>7</li>`,
				`<li>8</li>`,
			},
		},
		{
			"li:nth-child(3n)",
			`
			<ul>
				<li>1</li>
				<li>2</li>
				<li>3</li>
				<li>4</li>
				<li>5</li>
				<li>6</li>
				<li>7</li>
				<li>8</li>
			</ul>
			`,
			[]string{
				`<li>3</li>`,
				`<li>6</li>`,
			},
		},
		{
			"li:nth-child(3n+2)",
			`
			<ul>
				<li>1</li>
				<li>2</li>
				<li>3</li>
				<li>4</li>
				<li>5</li>
				<li>6</li>
				<li>7</li>
				<li>8</li>
			</ul>
			`,
			[]string{
				`<li>2</li>`,
				`<li>5</li>`,
				`<li>8</li>`,
			},
		},
		{
			"li:nth-child(3n+ 2)",
			`
			<ul>
				<li>1</li>
				<li>2</li>
				<li>3</li>
				<li>4</li>
				<li>5</li>
				<li>6</li>
				<li>7</li>
				<li>8</li>
			</ul>
			`,
			[]string{
				`<li>2</li>`,
				`<li>5</li>`,
				`<li>8</li>`,
			},
		},
		{
			"li:nth-child(3n - 2)",
			`
			<ul>
				<li>1</li>
				<li>2</li>
				<li>3</li>
				<li>4</li>
				<li>5</li>
				<li>6</li>
				<li>7</li>
				<li>8</li>
			</ul>
			`,
			[]string{
				`<li>1</li>`,
				`<li>4</li>`,
				`<li>7</li>`,
			},
		},
		{
			"li:nth-child(even)",
			`
			<ul>
				<li>1</li>
				<li>2</li>
				<li>3</li>
				<li>4</li>
				<li>5</li>
				<li>6</li>
				<li>7</li>
				<li>8</li>
			</ul>
			`,
			[]string{
				`<li>2</li>`,
				`<li>4</li>`,
				`<li>6</li>`,
				`<li>8</li>`,
			},
		},
		{
			"li:nth-child(odd)",
			`
			<ul>
				<li>1</li>
				<li>2</li>
				<li>3</li>
				<li>4</li>
				<li>5</li>
				<li>6</li>
				<li>7</li>
				<li>8</li>
			</ul>
			`,
			[]string{
				`<li>1</li>`,
				`<li>3</li>`,
				`<li>5</li>`,
				`<li>7</li>`,
			},
		},
		{
			"li:nth-last-child(2)",
			`
			<ul>
				<li>1</li>
				<li>2</li>
				<li>3</li>
				<li>4</li>
				<li>5</li>
				<li>6</li>
				<li>7</li>
				<li>8</li>
			</ul>
			`,
			[]string{
				`<li>7</li>`,
			},
		},
		{
			"li:nth-last-child(1n+2)",
			`
			<ul>
				<li>1</li>
				<li>2</li>
				<li>3</li>
				<li>4</li>
				<li>5</li>
				<li>6</li>
				<li>7</li>
				<li>8</li>
			</ul>
			`,
			[]string{
				`<li>1</li>`,
				`<li>2</li>`,
				`<li>3</li>`,
				`<li>4</li>`,
				`<li>5</li>`,
				`<li>6</li>`,
				`<li>7</li>`,
			},
		},
		{
			"li:nth-last-child(3n)",
			`
			<ul>
				<li>1</li>
				<li>2</li>
				<li>3</li>
				<li>4</li>
				<li>5</li>
				<li>6</li>
				<li>7</li>
				<li>8</li>
			</ul>
			`,
			[]string{
				`<li>3</li>`,
				`<li>6</li>`,
			},
		},
		{
			"li:nth-last-child(3n+2)",
			`
			<ul>
				<li>1</li>
				<li>2</li>
				<li>3</li>
				<li>4</li>
				<li>5</li>
				<li>6</li>
				<li>7</li>
				<li>8</li>
			</ul>
			`,
			[]string{
				`<li>1</li>`,
				`<li>4</li>`,
				`<li>7</li>`,
			},
		},
		{
			"li:nth-last-child(3n+2)",
			`
			<ul>
				<li>1</li>
				<li>2</li>
				<li>3</li>
				<li>4</li>
				<li>5</li>
				<li>6</li>
				<li>7</li>
				<li>8</li>
			</ul>
			`,
			[]string{
				`<li>1</li>`,
				`<li>4</li>`,
				`<li>7</li>`,
			},
		},
		{
			"ul :nth-of-type(3n+2)",
			`
			<ul>
				<p></p>
				<li>1</li>
				<p></p>
				<li>2</li>
				<li>3</li>
				<li>4</li>
				<li>5</li>
				<li>6</li>
				<li>7</li>
				<li>8</li>
			</ul>
			`,
			[]string{
				`<p></p>`,
				`<li>2</li>`,
				`<li>5</li>`,
				`<li>8</li>`,
			},
		},
		{
			"ul :nth-last-of-type(3n+2)",
			`
			<ul>
				<p></p>
				<li>1</li>
				<p></p>
				<li>2</li>
				<li>3</li>
				<li>4</li>
				<li>5</li>
				<li>6</li>
				<li>7</li>
				<li>8</li>
			</ul>
			`,
			[]string{
				`<p></p>`,
				`<li>1</li>`,
				`<li>4</li>`,
				`<li>7</li>`,
			},
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
			t.Errorf("Selecting %q (%s) from %s returned diff (-want, +got): %s", test.sel, s, in, diff)
		}
	}
}

func TestParseFuzz(t *testing.T) {
	strs := []string{
		"\xaa",
		":rLU((",
	}
	for _, s := range strs {
		Parse(s)
	}
}

func TestBadSelector(t *testing.T) {
	tests := []struct {
		sel string
		pos int
	}{
		{":nth-child(3+4n)", 0},
	}
	for _, test := range tests {
		_, err := Parse(test.sel)
		if err == nil {
			t.Errorf("Expected parsing %s to return error", test.sel)
			continue
		}
		var perr *ParseError
		if !errors.As(err, &perr) {
			t.Errorf("Expected parsing %s to return error of type *ParseError, got %T: %v", test.sel, err, err)
			continue
		}
		if test.pos != perr.Pos {
			t.Errorf("Parsing %s returned unexpected position, got=%d, want=%d", test.sel, perr.Pos, test.pos)
		}
	}
}
