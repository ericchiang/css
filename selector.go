package css

import (
	"strings"

	"golang.org/x/net/html"
)

type Selector struct {
	selectorsGroup []selector
}

func (s *Selector) Select(n *html.Node) []*html.Node {
	var matched []*html.Node
	for _, sel := range s.selectorsGroup {
		matched = append(matched, sel.Select(n)...)
	}
	return matched
}

type matcher interface {
	matches(n *html.Node) bool
}

type selector struct {
	selSeq selectorSequence
	combs  []combinatorSelector
}

func (s selector) Select(n *html.Node) []*html.Node {
	matched := s.selSeq.Select(n)
	for _, comb := range s.combs {
		var combMatched []*html.Node
		for _, n := range matched {
			combMatched = append(combMatched, comb.Select(n)...)
		}
		matched = combMatched
	}
	return matched
}

type selectorSequence struct {
	matchers []matcher
}

func (s selectorSequence) Select(n *html.Node) []*html.Node {
	if s.matches(n) {
		return []*html.Node{n}
	}
	var selected []*html.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		selected = append(selected, s.Select(c)...)
	}
	return selected
}

func (s selectorSequence) matches(n *html.Node) bool {
	for _, m := range s.matchers {
		if !m.matches(n) {
			return false
		}
	}
	return true
}

type combinatorSelector struct {
	combinator tokenType
	selSeq     selectorSequence
}

func (c combinatorSelector) Select(n *html.Node) []*html.Node {
	var matched []*html.Node
	switch c.combinator {
	case typeGreater:
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			if c.selSeq.matches(child) {
				matched = append(matched, child)
			}
		}
	case typeTilde:
		for sibl := n.NextSibling; sibl != nil; sibl = sibl.NextSibling {
			if c.selSeq.matches(sibl) {
				matched = append(matched, sibl)
			}
		}
	case typePlus:
		for sibl := n.NextSibling; sibl != nil; sibl = sibl.NextSibling {
			if c.selSeq.matches(sibl) {
				matched = append(matched, sibl)
			}
			// check matches against only the first element node
			if sibl.Type == html.ElementNode {
				break
			}
		}
	default:
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			matched = append(matched, c.selSeq.Select(child)...)
		}
	}
	return matched
}

type universal struct{}

func (u universal) matches(n *html.Node) bool {
	return true
}

type typeSelector struct {
	ele string
}

func (s typeSelector) matches(n *html.Node) bool {
	return n.Type == html.ElementNode && n.Data == s.ele
}

type attrSelector struct {
	key string
}

func (s attrSelector) matches(n *html.Node) bool {
	for _, attr := range n.Attr {
		if attr.Key == s.key {
			return true
		}
	}
	return false
}

type attrMatcher struct {
	key string
	val string
}

func (m attrMatcher) matches(n *html.Node) bool {
	for _, attr := range n.Attr {
		if attr.Key == m.key {
			return attr.Val == m.val
		}
	}
	return false
}

type attrCompMatcher struct {
	key  string
	val  string
	comp func(got, want string) bool
}

func (m attrCompMatcher) matches(n *html.Node) bool {
	for _, attr := range n.Attr {
		if attr.Key == m.key {
			return m.comp(attr.Val, m.val)
		}
	}
	return false
}

var (
	prefixMatcher = strings.HasPrefix
	suffixMatcher = strings.HasSuffix
	subStrMatcher = strings.Contains
)

func includesMatcher(got, want string) bool {
	for _, s := range strings.Fields(got) {
		if s == want {
			return true
		}
	}
	return false
}

func dashMatcher(got, want string) bool {
	for _, s := range strings.Split(got, "-") {
		if s == want {
			return true
		}
	}
	return false
}

type negation struct {
	m matcher
}

func (neg negation) matches(n *html.Node) bool {
	return !neg.m.matches(n)
}

// matcherFunc for pseudo classes
type matcherFunc func(n *html.Node) bool

func (f matcherFunc) matches(n *html.Node) bool {
	return f(n)
}

func empty(n *html.Node) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.CommentNode {
			return false
		}
	}
	return true
}

func firstChild(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	for s := n.PrevSibling; s != nil; s = s.PrevSibling {
		if s.Type == html.ElementNode {
			return false
		}
	}
	return true
}

func firstOfType(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	for s := n.PrevSibling; s != nil; s = s.PrevSibling {
		if s.Type == html.ElementNode && s.Data == n.Data {
			return false
		}
	}
	return true
}

func lastChild(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	for s := n.NextSibling; s != nil; s = s.NextSibling {
		if s.Type == html.ElementNode {
			return false
		}
	}
	return true
}

func lastOfType(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	for s := n.NextSibling; s != nil; s = s.NextSibling {
		if s.Type == html.ElementNode && s.Data == n.Data {
			return false
		}
	}
	return true
}

func onlyChild(n *html.Node) bool {
	return firstChild(n) && lastChild(n)
}

func onlyOfType(n *html.Node) bool {
	return firstOfType(n) && lastOfType(n)
}

func root(n *html.Node) bool {
	return n.Parent == nil
}

type nthChild struct {
	a, b int
}

func (nth nthChild) matches(n *html.Node) bool {
	pos := 0
	for s := n.PrevSibling; s != nil; s = s.PrevSibling {
		if s.Type == html.ElementNode {
			pos++
		}
	}
	return posMatches(nth.a, nth.b, pos)
}

func posMatches(a, b, pos int) bool {
	n := (pos - b + 1)
	return (a == 0 && n == 0) || (n%a == 0 && n/a >= 0)
}
