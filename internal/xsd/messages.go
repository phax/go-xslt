package xsd

// Custom assertion messages. XSD 1.1 has no message slot on xs:assert, so a
// schema may carry readable diagnostics in its xs:appinfo, in this package's
// own namespace:
//
//	<xs:assert test="...">
//	  <xs:annotation><xs:appinfo>
//	    <msg:message rule="BR-33" xml:lang="en">Reason is missing.</msg:message>
//	  </xs:appinfo></xs:annotation>
//	</xs:assert>
//
// Reading them is outside XSD itself: they affect Error.Message/Rule only,
// never validity, and appinfo content in any other namespace is ignored.

import (
	"strconv"
	"strings"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

// MessagesNS is the namespace of the msg:message convention.
const MessagesNS = "https://github.com/tim-riep/go-xslt/xsd/messages"

type assertMsg struct {
	rule, lang, text string
}

// assertMessages collects the msg:message entries of an xs:assert/xs:assertion.
func assertMessages(a *xmltree.Node) []assertMsg {
	var out []assertMsg
	for _, ann := range a.Children {
		if !isXS(ann, "annotation") {
			continue
		}
		for _, info := range ann.Children {
			if !isXS(info, "appinfo") {
				continue
			}
			for _, m := range info.Children {
				if m.Kind != xmltree.KindElement || m.Name.Space != MessagesNS || m.Name.Local != "message" {
					continue
				}
				am := assertMsg{text: strings.TrimSpace(m.StringValue())}
				am.rule, _ = m.AttrLocal("rule")
				for _, at := range m.Attrs {
					if at.Name.Space == xmlNS && at.Name.Local == "lang" {
						am.lang = strings.TrimSpace(at.Value)
					}
				}
				out = append(out, am)
			}
		}
	}
	return out
}

// assertionFailed builds the diagnostic for assertion a failing on el.
func assertionFailed(a *assertion, el *xmltree.Node) *Error {
	e := invalidf("cvc-assertion", "assertion failed on %s", nameOf(el))
	e.Line, e.Col = int(el.Line), int(el.Col)
	e.Path = elementPath(el)
	e.msgs = a.msgs
	return e
}

// pickMessage replaces the generic text by the best msg:message: the one in
// lang (exact, then primary subtag), else one without xml:lang, else the first.
func (e *Error) pickMessage(lang string) {
	if len(e.msgs) == 0 {
		return
	}
	lang = strings.ToLower(lang)
	pick := func(match func(string) bool) *assertMsg {
		for i := range e.msgs {
			if match(strings.ToLower(e.msgs[i].lang)) {
				return &e.msgs[i]
			}
		}
		return nil
	}
	var m *assertMsg
	if lang != "" {
		primary, _, _ := strings.Cut(lang, "-")
		if m = pick(func(l string) bool { return l == lang }); m == nil {
			m = pick(func(l string) bool { return l == primary || (l != "" && strings.HasPrefix(l, primary+"-")) })
		}
	}
	if m == nil {
		m = pick(func(l string) bool { return l == "" })
	}
	if m == nil {
		m = &e.msgs[0]
	}
	e.Rule = m.rule
	if m.text != "" {
		e.Message = m.text
	}
	e.msgs = nil
}

// elementPath renders el's position as /Root[1]/Child[2] (local names, 1-based
// index among same-named element siblings).
func elementPath(el *xmltree.Node) string {
	var segs []string
	for n := el; n != nil && n.Kind == xmltree.KindElement; n = n.Parent {
		idx := 1
		if n.Parent != nil {
			idx = 0
			for _, sib := range n.Parent.Children {
				if sib.Kind == xmltree.KindElement && sib.Name == n.Name {
					idx++
				}
				if sib == n {
					break
				}
			}
		}
		segs = append(segs, n.Name.Local+"["+strconv.Itoa(idx)+"]")
	}
	var b strings.Builder
	for i := len(segs) - 1; i >= 0; i-- {
		b.WriteByte('/')
		b.WriteString(segs[i])
	}
	return b.String()
}
