# A CSS selector compiler

[![GoDoc](https://godoc.org/github.com/ericchiang/css?status.svg)](https://godoc.org/github.com/ericchiang/css)

This package implements a CSS selector compiler for Go's HTML parsing package [`golang.org/x/net/html`](https://godoc.org/golang.org/x/net/html).

```go
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ericchiang/css"
	"golang.org/x/net/html"
)

var data = `
<p>
  <h2 id="foo">a header</h2>
  <h2 id="bar">another header</h2>
</p>`

func main() {
	sel, err := css.Compile("h2#foo")
	if err != nil {
		panic(err)
	}
	node, err := html.Parse(strings.NewReader(data))
	if err != nil {
		panic(err)
	}
	for _, ele := range sel.Select(node) {
		html.Render(os.Stdout, ele)
	}
	fmt.Println()
}
```

```
$ go run example/css.go
<h2 id="foo">a header</h2>
```

## Details

This package implements the W3 [_Selectors Level 3_](http://www.w3.org/TR/css3-selectors/) specification. The exact grammar can be found in the `grammar.txt` file.
