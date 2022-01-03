// Package css implements CSS selectors for HTML elements.
package css

import (
	"errors"
	"fmt"
	"strings"

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
		selected = append(selected, sel.find(n)...)
	}
	return selected
}

func findAll(n *html.Node, fn func(n *html.Node) bool) []*html.Node {
	var m []*html.Node
	if fn(n) {
		m = append(m, n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode {
			continue
		}
		m = append(m, findAll(c, fn)...)
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

	combinators []func(n *html.Node) []*html.Node
}

func (s selector) find(n *html.Node) []*html.Node {
	nodes := findAll(n, s.m.match)
	for _, combinator := range s.combinators {
		var ns []*html.Node
		for _, n := range nodes {
			ns = append(ns, combinator(n)...)
		}
		nodes = ns
	}
	return nodes
}

type descendantCombinator struct {
	m *compoundSelectorMatcher
}

func (c *descendantCombinator) find(n *html.Node) []*html.Node {
	var nodes []*html.Node
	for n := n.FirstChild; n != nil; n = n.NextSibling {
		if n.Type != html.ElementNode {
			continue
		}
		nodes = append(nodes, findAll(n, c.m.match)...)
	}
	return nodes
}

type childCombinator struct {
	m *compoundSelectorMatcher
}

func (c *childCombinator) find(n *html.Node) []*html.Node {
	var nodes []*html.Node
	for n := n.FirstChild; n != nil; n = n.NextSibling {
		if n.Type != html.ElementNode {
			continue
		}
		if c.m.match(n) {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

type adjacentCombinator struct {
	m *compoundSelectorMatcher
}

func (c *adjacentCombinator) find(n *html.Node) []*html.Node {
	var (
		nodes []*html.Node
		prev  *html.Node
		next  *html.Node
	)
	for prev = n.PrevSibling; prev != nil; prev = prev.PrevSibling {
		if prev.Type == html.ElementNode {
			break
		}
	}
	for next = n.NextSibling; next != nil; next = next.NextSibling {
		if next.Type == html.ElementNode {
			break
		}
	}
	if prev != nil && c.m.match(prev) {
		nodes = append(nodes, prev)
	}
	if next != nil && c.m.match(next) {
		nodes = append(nodes, next)
	}
	return nodes
}

type siblingCombinator struct {
	m *compoundSelectorMatcher
}

func (c *siblingCombinator) find(n *html.Node) []*html.Node {
	var nodes []*html.Node
	for n := n.PrevSibling; n != nil; n = n.PrevSibling {
		if n.Type != html.ElementNode {
			continue
		}
		if c.m.match(n) {
			nodes = append(nodes, n)
		}
	}
	for n := n.NextSibling; n != nil; n = n.NextSibling {
		if n.Type != html.ElementNode {
			continue
		}
		if c.m.match(n) {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

func (c *compiler) compile(s *complexSelector) *selector {
	m := &selector{
		m: c.compoundSelector(&s.sel),
	}
	curr := s
	for {
		if curr.next == nil {
			return m
		}
		sel := c.compoundSelector(&curr.next.sel)
		combinator := curr.combinator

		curr = curr.next

		var fn func(n *html.Node) []*html.Node
		switch combinator {
		case "":
			fn = (&descendantCombinator{sel}).find
		case ">":
			fn = (&childCombinator{sel}).find
		case "+":
			fn = (&adjacentCombinator{sel}).find
		case "~":
			fn = (&siblingCombinator{sel}).find
		default:
			c.errorf(curr.pos, "unexpected combinator: %s", combinator)
			continue
		}
		m.combinators = append(m.combinators, fn)
	}
	return m
}

type compoundSelectorMatcher struct {
	m   *typeSelectorMatcher
	scm []subclassSelectorMatcher
}

func (c *compoundSelectorMatcher) match(n *html.Node) bool {
	if c.m != nil {
		if !c.m.match(n) {
			return false
		}
	}
	for _, m := range c.scm {
		if !m.match(n) {
			return false
		}
	}
	return true
}

func (c *compiler) compoundSelector(s *compoundSelector) *compoundSelectorMatcher {
	m := &compoundSelectorMatcher{}
	if s.typeSelector != nil {
		m.m = c.typeSelector(s.typeSelector)
	}
	for _, sc := range s.subClasses {
		scm := c.subclassSelector(&sc)
		if scm != nil {
			m.scm = append(m.scm, *scm)
		}
	}
	if len(s.pseudoSelectors) != 0 {
		if c.errorf(s.pos, "pseudo selector not supported") {
			return nil
		}
	}
	return m
}

type subclassSelectorMatcher struct {
	idSelector        string
	classSelector     string
	attributeSelector *attributeSelectorMatcher
}

func (s *subclassSelectorMatcher) match(n *html.Node) bool {
	if s.idSelector != "" {
		for _, a := range n.Attr {
			if a.Key == "id" && a.Val == s.idSelector {
				return true
			}
		}
		return false
	}

	if s.classSelector != "" {
		for _, a := range n.Attr {
			if a.Key == "class" && a.Val == s.classSelector {
				return true
			}
		}
		return false
	}

	if s.attributeSelector != nil {
		return s.attributeSelector.match(n)
	}
	return false
}

func (c *compiler) subclassSelector(s *subclassSelector) *subclassSelectorMatcher {
	m := &subclassSelectorMatcher{
		idSelector:    s.idSelector,
		classSelector: s.classSelector,
	}
	if s.attributeSelector != nil {
		m.attributeSelector = c.attributeSelector(s.attributeSelector)
	}
	if s.pseudoClassSelector != nil {
		if c.errorf(s.pos, "pseudo class selector not supported") {
			return nil
		}
	}
	return m
}

type attributeSelectorMatcher struct {
	ns namespaceMatcher
	fn func(key, val string) bool
}

func (a *attributeSelectorMatcher) match(n *html.Node) bool {
	for _, attr := range n.Attr {
		if a.ns.match(attr.Namespace) && a.fn(attr.Key, attr.Val) {
			return true
		}
	}
	return false
}

func (c *compiler) attributeSelector(s *attributeSelector) *attributeSelectorMatcher {
	m := &attributeSelectorMatcher{
		ns: newNamespaceMatcher(s.wqName.hasPrefix, s.wqName.prefix),
	}
	key := s.wqName.value
	val := s.val

	if s.modifier {
		key = strings.ToLower(key)
		val = strings.ToLower(val)
	}

	// https://developer.mozilla.org/en-US/docs/Web/CSS/Attribute_selectors
	switch s.matcher {
	case "=":
		m.fn = func(k, v string) bool { return k == key && v == val }
	case "~=":
		m.fn = func(k, v string) bool {
			if k != key {
				return false
			}
			for _, f := range strings.Fields(v) {
				if f == val {
					return true
				}
			}
			return false
		}
	case "|=":
		// "Represents elements with an attribute name of attr whose value can be
		// exactly value or can begin with value immediately followed by a hyphen,
		// - (U+002D). It is often used for language subcode matches."
		m.fn = func(k, v string) bool {
			return k == key && (v == val || strings.HasPrefix(v, val+"-"))
		}
	case "^=":
		m.fn = func(k, v string) bool {
			return k == key && strings.HasPrefix(v, val)
		}
	case "$=":
		m.fn = func(k, v string) bool {
			return k == key && strings.HasSuffix(v, val)
		}
	case "*=":
		m.fn = func(k, v string) bool {
			return k == key && strings.Contains(v, val)
		}
	case "":
		m.fn = func(k, v string) bool { return k == key }
	default:
		c.errorf(s.pos, "unsupported attribute matcher: %s", s.matcher)
		return nil
	}
	if s.modifier {
		fn := m.fn
		m.fn = func(k, v string) bool {
			k = strings.ToLower(k)
			v = strings.ToLower(v)
			return fn(k, v)
		}
	}
	return m
}

// namespaceMatcher performs <ns-prefix> matching for elements and attributes.
type namespaceMatcher struct {
	noNamespace bool
	namespace   string
}

func newNamespaceMatcher(hasPrefix bool, prefix string) namespaceMatcher {
	if !hasPrefix {
		return namespaceMatcher{}
	}
	if prefix == "" {
		return namespaceMatcher{noNamespace: true}
	}
	if prefix == "*" {
		return namespaceMatcher{}
	}
	return namespaceMatcher{namespace: prefix}
}

func (n *namespaceMatcher) match(ns string) bool {
	if n.noNamespace {
		return ns == ""
	}
	if n.namespace == "" {
		return true
	}
	return n.namespace == ns
}

type typeSelectorMatcher struct {
	allAtoms bool
	atom     atom.Atom
	ns       namespaceMatcher
}

func (t *typeSelectorMatcher) match(n *html.Node) (ok bool) {
	if !(t.allAtoms || t.atom == n.DataAtom) {
		return false
	}
	return t.ns.match(n.Namespace)
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
	m.ns = newNamespaceMatcher(s.hasPrefix, s.prefix)
	return m
}
