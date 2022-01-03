// Package css implements CSS selectors for HTML elements.
package css

import (
	"errors"
	"fmt"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// ParseError is returned indicating an lex, parse, or compilation error with
// the associated position in the string the error occurred.
type ParseError struct {
	Pos int
	Msg string
}

// Error returns a formatted version of the error.
func (p *ParseError) Error() string {
	return fmt.Sprintf("css: %s at position %d", p.Msg, p.Pos)
}

func errorf(pos int, msg string, v ...interface{}) error {
	return &ParseError{pos, fmt.Sprintf(msg, v...)}
}

// Selector is a compiled CSS selector.
type Selector struct {
	s []*selector
}

// Select returns any matches from a parsed HTML document.
func (s *Selector) Select(n *html.Node) []*html.Node {
	selected := []*html.Node{}
	for _, sel := range s.s {
		selected = append(selected, match(n, sel.match)...)
	}
	return selected
}

func match(n *html.Node, fn func(n *html.Node) bool) []*html.Node {
	if fn(n) {
		return []*html.Node{n}
	}
	var m []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		m = append(m, match(c, fn)...)
	}
	return m
}

// MustParse is like Parse but panics on errors.
func MustParse(s string) *Selector {
	sel, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return sel
}

// Parse compiles a complex selector list from a string. The parser supports
// Selectors Level 4.
//
// Multiple selectors are supported through comma separated values. For example
// "h1, h2".
//
// Parse reports the first error hit when compiling.
func Parse(s string) (*Selector, error) {
	p := newParser(s)
	list, err := p.parse()
	if err != nil {
		var perr *parseErr
		if errors.As(err, &perr) {
			return nil, &ParseError{perr.t.pos, perr.msg}
		}
		var lerr *lexErr
		if errors.As(err, &lerr) {
			return nil, &ParseError{lerr.last, lerr.msg}
		}
		return nil, err
	}
	sel := &Selector{}

	c := compiler{maxErrs: 1}
	for _, s := range list {
		m := c.compile(&s)
		if m == nil {
			continue
		}
		sel.s = append(sel.s, m)
	}
	if err := c.err(); err != nil {
		return nil, err
	}
	return sel, nil
}

type compiler struct {
	sels    []complexSelector
	maxErrs int
	errs    []error
}

func (c *compiler) err() error {
	if len(c.errs) == 0 {
		return nil
	}
	return c.errs[0]
}

func (c *compiler) errorf(pos int, msg string, v ...interface{}) bool {
	err := &ParseError{pos, fmt.Sprintf(msg, v...)}
	c.errs = append(c.errs, err)
	if len(c.errs) >= c.maxErrs {
		return true
	}
	return false
}

type selector struct {
	m *compoundSelectorMatcher
}

func (s selector) match(n *html.Node) bool {
	if s.m != nil {
		return s.m.match(n)
	}
	return false
}

func (c *compiler) compile(s *complexSelector) *selector {
	m := &selector{c.compoundSelector(&s.sel)}
	if s.combinator != "" {
		if c.errorf(s.pos, "combinator not supported") {
			return nil
		}
	}
	return m
}

type compoundSelectorMatcher struct {
	m *typeSelectorMatcher
}

func (c *compoundSelectorMatcher) match(n *html.Node) bool {
	if c.m != nil {
		return c.m.match(n)
	}
	return false
}

func (c *compiler) compoundSelector(s *compoundSelector) *compoundSelectorMatcher {
	m := &compoundSelectorMatcher{}
	if s.typeSelector != nil {
		m.m = c.typeSelector(s.typeSelector)
	}
	if len(s.subClasses) != 0 {
		if c.errorf(s.pos, "subclass selector not supported") {
			return nil
		}
	}
	if len(s.pseudoSelectors) != 0 {
		if c.errorf(s.pos, "pseudo selector not supported") {
			return nil
		}
	}
	return m
}

type typeSelectorMatcher struct {
	allAtoms    bool
	atom        atom.Atom
	noNamespace bool
	namespace   string
}

func (t *typeSelectorMatcher) match(n *html.Node) bool {
	return (t.allAtoms || t.atom == n.DataAtom) &&
		((t.noNamespace && n.Namespace == "") ||
			(t.namespace == "") ||
			(t.namespace == n.Namespace))
}

func (c *compiler) typeSelector(s *typeSelector) *typeSelectorMatcher {
	m := &typeSelectorMatcher{}
	if s.value == "*" {
		m.allAtoms = true
	} else {
		a := atom.Lookup([]byte(s.value))
		if a == 0 {
			if c.errorf(s.pos, "unrecognized node name: %s", s.value) {
				return nil
			}
		}
		m.atom = a
	}
	if !s.hasPrefix {
		return m
	}
	switch s.prefix {
	case "":
		m.noNamespace = true
	case "*":
	default:
		m.namespace = s.prefix
	}
	return m
}
