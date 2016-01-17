/*
Package css implements CSS selector HTML search.

The syntax is that select accepted is the CSS Selector Level 3 spec which is described
at https://www.w3.org/TR/css3-selectors/. The parsing grammar is defined the grammar.txt
file found within the package.

Selectors compiled by this package search through golang.org/x/net/html nodes and should
be used in conjunction with that package.

	data := `<p>
	  <h2 id="foo">a header</h2>
	  <h2 id="bar">another header</h2>
	</p>`

	sel, err := css.Compile("h2#foo")
	if err != nil {
		// handle error
	}
	node, err := html.Parse(strings.NewReader(data))
	if err != nil {
		// handle error
	}
	elements := sel.Select(node)
*/
package css
