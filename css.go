package css

import (
	"errors"
	"fmt"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type ParseError struct {
	Pos int
	Msg string
}

func (p *ParseError) Error() string {
	return fmt.Sprintf("css: %s at position %d", p.Msg, p.Pos)
}

func errorf(pos int, msg string, v ...interface{}) error {
	return &ParseError{pos, fmt.Sprintf(msg, v...)}
}

type Selector struct {
	s []complexSelector
}

func (s *Selector) Select(n *html.Node) []*html.Node {
	return nil
}

func MustParse(s string) *Selector {
	sel, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return sel
}

func Parse(s string) (*Selector, error) {
	p := newParser(s)
	sel, err := p.parse()
	if err == nil {
		return &Selector{sel}, nil
	}
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

func (t *typeSelector) compile() (*typeSelectorMatcher, error) {
	m := &typeSelectorMatcher{}
	if t.value == "*" {
		m.allAtoms = true
	} else {
		a := atom.Lookup([]byte(t.value))
		if a == 0 {
			return nil, errorf(t.pos, "unrecognized node name: %s", t.value)
		}
		m.atom = a
	}
	if !t.hasPrefix {
		return m, nil
	}
	switch t.prefix {
	case "":
		m.noNamespace = true
	case "*":
	default:
		m.namespace = t.prefix
	}
	return m, nil
}

type selector struct {
}
