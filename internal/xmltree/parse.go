package xmltree

import (
	"encoding/xml"
	"fmt"
	"github.com/tim-riep/go-xslt/internal/fileuri"
	"io"
	"strconv"
	"strings"
	"sync/atomic"
)

// ParseError carries a 1-based line/column for diagnostics.
type ParseError struct {
	Line, Col int
	Msg       string
}

// StripSpaceHook reports whether an xsl:strip-space declaration (net of any
// xsl:preserve-space override, by import precedence and specificity — NOT
// including an xml:space="preserve" scope, which the parser itself tracks
// independently of this hook, nor the schema-aware simple-content-typed
// exemption, which cannot be known until the whole tree is validated) would
// strip a whitespace-only text child of an element with this expanded name.
// A caller that compiled a stylesheet (internal/xslt) builds this from its
// own xsl:strip-space/xsl:preserve-space tables — this package cannot import
// that one (xmltree <- xpath <- xslt is a real cycle constraint), so this is
// an injected hook, mirroring the pattern internal/xpath's Context already
// uses for its own cross-package seams (SchemaTypes, TransformFunc, ...).
// nil (every plain Parse*/ParseLenient11WithBase entry point) means "nothing
// to ask" — no stripping happens at parse time, identical to today.
type StripSpaceHook func(name Name) bool

func (e *ParseError) Error() string { return e.Msg }

// downgradeXMLVersion rewrites a leading `<?xml version="1.1"?>` declaration to
// version 1.0 so Go's encoding/xml (which only accepts 1.0) will parse the
// document. The replacement is length-preserving so byte offsets — and thus the
// line/column diagnostics — are unchanged. Only the version token inside the
// prolog declaration is touched; the XML 1.1/1.0 differences (a few extra
// control characters, NEL line endings) do not affect our node model.
func downgradeXMLVersion(src string) string {
	if !strings.HasPrefix(strings.TrimLeft(src, "\uFEFF \t"), "<?xml") {
		return src
	}
	end := strings.Index(src, "?>")
	if end < 0 {
		return src
	}
	decl := src[:end]
	for _, q := range []string{`version="1.1"`, `version='1.1'`} {
		if i := strings.Index(decl, q); i >= 0 {
			repl := strings.Replace(q, "1.1", "1.0", 1)
			return src[:i] + repl + src[i+len(q):]
		}
	}
	return src
}

// Parse parses an XML document into a tree. The returned node is a Document
// node whose single element child is the root element. XML 1.1 documents are
// parsed with 1.0 rules (see ParseLenient11 for the XSD validator's tolerant
// entry point — fn:parse-xml must NOT get the leniency, the QT3 suite expects
// 1.0-processor errors there).
func Parse(src string) (*Node, error) {
	return parseRaw(src, "", nil)
}

// ParseWithBase parses like Parse, but also resolves a DOCTYPE's external
// SYSTEM subset (a local .dtd file of entity declarations) against baseDir
// when the document's own DOCTYPE references one and doesn't already declare
// the entity internally (copy-1201/1202: a stylesheet's own DOCTYPE naming a
// local .dtd of character entities like &egrave;). baseDir == "" behaves
// exactly like Parse (no external subset is ever fetched).
func ParseWithBase(src, baseDir string) (*Node, error) {
	return parseRaw(src, baseDir, nil)
}

// ParseLenient11 parses like Parse but bridges declared XML 1.1 documents to
// Go's 1.0-4th-edition decoder with two pre-passes: C0 character references
// (legal in 1.1) and 5th-edition-only name characters (e.g. U+0133) are
// substituted with sentinels and restored in the parsed tree (saxon
// XmlVersions xv001/003/004/006/008). Used by the XSD validator for schema and
// instance documents.
func ParseLenient11(src string) (*Node, error) {
	return ParseLenient11WithBase(src, "")
}

// ParseLenient11WithBase is ParseLenient11 with ParseWithBase's external-subset
// resolution — the entry point the XSLT engine uses for stylesheet modules,
// source documents and document()/fn:doc() results, all of which may legally be
// XML 1.1 (the misc/xml-version test set). fn:parse-xml deliberately keeps the
// strict 1.0 Parse.
func ParseLenient11WithBase(src, baseDir string) (*Node, error) {
	return parseLenient11WithBaseStrip(src, baseDir, nil)
}

// ParseLenient11WithBaseStripSpace is ParseLenient11WithBase with a
// StripSpaceHook consulted at parse time: a whitespace-only text run
// directly inside an element the hook (net of this element's own xml:space
// scope, tracked here regardless of whether strip is nil) reports true for
// is never allocated as a Node at all — for a source document with hundreds
// of thousands of such runs, the ones that will certainly be thrown away a
// moment later by the caller's own post-parse applyStripSpace walk are the
// one allocation this skips entirely. strip == nil behaves identically to
// ParseLenient11WithBase. This is additive, not a replacement: a caller
// still runs its own whole-tree stripping pass afterward regardless, as a
// safety net for anything this hook doesn't cover (import-precedence ties
// this engine resolves one way today, a future schema-aware simple-content
// exemption the hook can't see at parse time, ...).
func ParseLenient11WithBaseStripSpace(src, baseDir string, strip StripSpaceHook) (*Node, error) {
	return parseLenient11WithBaseStrip(src, baseDir, strip)
}

func parseLenient11WithBaseStrip(src, baseDir string, strip StripSpaceHook) (*Node, error) {
	src = decodeBOM(src) // UTF-16 sources become UTF-8 before any tokenizing
	if !declaresXML11(src) {
		return parseRaw(src, baseDir, strip)
	}
	prepped, restore := substC0Refs(src)
	tokens := map[string]rune{} // ASCII name-safe token -> original 5e name char
	for attempt := 0; ; attempt++ {
		doc, err := parseRaw(prepped, baseDir, strip)
		if err == nil {
			if len(restore) > 0 || len(tokens) > 0 {
				restoreSentinels(doc, restore, tokens)
			}
			return doc, nil
		}
		if attempt >= 4 {
			return doc, err
		}
		// Error-driven name repair: substitute only runes the 1.1/5e Name
		// production actually allows — a genuinely illegal name char keeps its
		// error (the set's .n twins must stay invalid). The sentinel must be
		// name-legal for the 4e decoder, so it is an ASCII token, not a PUA
		// rune.
		bad, ok := invalidNameIn(err)
		if !ok {
			return doc, err
		}
		added := false
		for _, r := range bad {
			if r >= 0x80 && isName11(r) {
				tok := fmt.Sprintf("Zq5e%XeZ", r)
				if _, dup := tokens[tok]; dup {
					continue
				}
				tokens[tok] = r
				prepped = strings.ReplaceAll(prepped, string(r), tok)
				added = true
			}
		}
		if !added {
			return doc, err
		}
	}
}

// declaresXML11 reports whether the prolog declares version="1.1".
func declaresXML11(src string) bool {
	head := strings.TrimLeft(src, "\uFEFF \t")
	if !strings.HasPrefix(head, "<?xml") {
		return false
	}
	end := strings.Index(head, "?>")
	if end < 0 {
		return false
	}
	decl := head[:end]
	return strings.Contains(decl, `version="1.1"`) || strings.Contains(decl, `version='1.1'`)
}

// substC0Refs replaces character references to C0 controls (0x1-0x1F minus
// tab/LF/CR — legal in XML 1.1 only as references) with literal Private-Use
// sentinels, returning the rewritten source and the restore map.
func substC0Refs(src string) (string, map[rune]rune) {
	restore := map[rune]rune{}
	var b strings.Builder
	for i := 0; i < len(src); {
		j := strings.Index(src[i:], "&#")
		if j < 0 {
			b.WriteString(src[i:])
			break
		}
		j += i
		b.WriteString(src[i:j])
		end := strings.IndexByte(src[j:], ';')
		if end < 0 {
			b.WriteString(src[j:])
			break
		}
		end += j
		body := src[j+2 : end]
		var v int64 = -1
		if strings.HasPrefix(body, "x") || strings.HasPrefix(body, "X") {
			if n, err := strconv.ParseInt(body[1:], 16, 32); err == nil {
				v = n
			}
		} else if n, err := strconv.ParseInt(body, 10, 32); err == nil {
			v = n
		}
		if v >= 0x1 && v <= 0x1F && v != 0x9 && v != 0xA && v != 0xD {
			s := rune(0xE000 + v)
			restore[s] = rune(v)
			b.WriteRune(s)
		} else {
			b.WriteString(src[j : end+1])
		}
		i = end + 1
	}
	return b.String(), restore
}

// invalidNameIn extracts the offending name from a decoder "invalid XML name"
// error.
func invalidNameIn(err error) (string, bool) {
	msg := err.Error()
	const tag = "invalid XML name: "
	if i := strings.Index(msg, tag); i >= 0 {
		return msg[i+len(tag):], true
	}
	return "", false
}

// isName11 reports whether r is an XML 1.1 / 1.0-5th-edition NameChar
// (NameStartChar ∪ the extra tail chars). Kept in sync with the table in
// internal/xpath/regex_class.go (which cannot be imported here — xpath depends
// on xmltree).
func isName11(r rune) bool {
	for _, rng := range [][2]rune{
		{':', ':'}, {'A', 'Z'}, {'_', '_'}, {'a', 'z'},
		{0xC0, 0xD6}, {0xD8, 0xF6}, {0xF8, 0x2FF}, {0x370, 0x37D}, {0x37F, 0x1FFF},
		{0x200C, 0x200D}, {0x2070, 0x218F}, {0x2C00, 0x2FEF}, {0x3001, 0xD7FF},
		{0xF900, 0xFDCF}, {0xFDF0, 0xFFFD}, {0x10000, 0xEFFFF},
		{'-', '-'}, {'.', '.'}, {'0', '9'}, {0xB7, 0xB7},
		{0x300, 0x36F}, {0x203F, 0x2040},
	} {
		if r >= rng[0] && r <= rng[1] {
			return true
		}
	}
	return false
}

// restoreSentinels maps the C0 sentinel runes and the ASCII name tokens back
// to their originals in every name, attribute, and text value of the tree.
func restoreSentinels(n *Node, runes map[rune]rune, tokens map[string]rune) {
	fix := func(s string) string {
		if len(runes) > 0 {
			s = strings.Map(func(r rune) rune {
				if o, ok := runes[r]; ok {
					return o
				}
				return r
			}, s)
		}
		for tok, o := range tokens {
			s = strings.ReplaceAll(s, tok, string(o))
		}
		return s
	}
	var walk func(*Node)
	walk = func(nd *Node) {
		nd.Name.Local = fix(nd.Name.Local)
		// A namespace URI is an attribute value too (xml-version-012 binds a
		// prefix to a URI ending in a U+0346 combining mark).
		nd.Name.Space = fix(nd.Name.Space)
		nd.Value = fix(nd.Value)
		for i := range nd.Attrs {
			nd.Attrs[i].Name.Local = fix(nd.Attrs[i].Name.Local)
			nd.Attrs[i].Name.Space = fix(nd.Attrs[i].Name.Space)
			nd.Attrs[i].Value = fix(nd.Attrs[i].Value)
		}
		for _, ns := range nd.NSDecls() {
			ns.Value = fix(ns.Value)
		}
		for _, ch := range nd.Children {
			walk(ch)
		}
	}
	walk(n)
}

func parseRaw(src, baseDir string, strip StripSpaceHook) (*Node, error) {
	// A byte-order mark is an encoding signature, not document content: Go's
	// decoder would otherwise hand back a U+FEFF character-data token ahead of
	// the root element, which then rides through the transform into the result
	// (conflict-resolution-1701: a BOM-prefixed source document).
	src = decodeBOM(src)
	// External parsed general entities are spliced in textually before
	// tokenizing (see expandExternalEntities) — encoding/xml can only supply
	// an entity's replacement as character data, never as markup.
	src, entSpans := expandExternalEntities(src, baseDir)
	src = normalizeAttrWhitespace(src)
	lines := newLinePos(src)
	dec := xml.NewDecoder(strings.NewReader(downgradeXMLVersion(src)))
	dec.Strict = true
	dec.CharsetReader = charsetReader
	dec.Entity = documentEntities(src, baseDir)
	idKinds := idAttrKinds(src)
	attrDefaults := attlistDefaults(src)
	elemOnly := elementOnlyDecls(src, baseDir)

	doc := &Node{Kind: KindDocument}
	if u := unparsedEntityDecls(src, baseDir); len(u) > 0 {
		doc.SetUnparsed(u)
	}
	stack := []*Node{doc}
	// namespace scope stack: each frame maps prefix -> uri (default prefix "").
	nsStack := []map[string]string{{
		"xml": "http://www.w3.org/XML/1998/namespace",
		"":    "",
	}}
	// rawNames holds each open element's name AS WRITTEN (prefix in Space),
	// for the end-tag check RawToken leaves to its caller.
	var rawNames []xml.Name
	// nameIntern dedupes element Local/Prefix strings across the document: the
	// decoder hands back a freshly allocated string for every single tag, but
	// a structured document typically has only a handful of DISTINCT element
	// names repeated many thousands of times (a UBL invoice's cac:InvoiceLine,
	// cbc:ID, ...). Scoped to this one parse, not a package-level table —
	// each parse gets its own, so there's no cross-document retention and no
	// concurrency concern.
	nameIntern := make(map[string]string)
	// preserveSpace mirrors stack 1:1 (one entry per open element, plus the
	// document-level seed): xml:space="preserve" on an element forces true
	// for it and every descendant until overridden; any other explicit
	// xml:space value (including "default") resets to false; absent, it's
	// inherited unchanged. Needed here — not just by internal/xslt's own
	// post-parse stripWalk, which does the identical thing — because the
	// strip hook below must net out against it at the exact point a
	// whitespace-only run would otherwise be discarded, before stripWalk
	// ever gets a tree to walk.
	preserveSpace := []bool{false}

	// pendingText/pendingOpen/pendingLine/pendingCol accumulate consecutive
	// CharData tokens (plain text and/or CDATA sections — the decoder reports
	// a CDATA section as a separate CharData token, but in the XML data
	// model it forms ONE text node with surrounding chardata: the existing
	// coalescing comment below, now in flushPending, gives the exact case)
	// into one run, deferring the decision of whether it becomes a Node (and
	// with what content) until the run actually closes. This matters
	// specifically for the strip hook: a run's early tokens looking
	// whitespace-only is not the same as the WHOLE run being whitespace-only
	// once it closes (e.g. "<a>   <![CDATA[x]]></a>" opens looking
	// strippable and only three tokens later turns out to be " x") — an
	// eager per-token decision would permanently lose content a deferred one
	// never sees as anything but the complete, correct run.
	// pendingBuf is reused across runs (a strings.Builder was Reset — and so
	// regrown from nothing — for every run, after string(t) had already copied
	// the token once). The run's string is materialized exactly once, at the
	// flush; whitespace-only runs (the indentation between elements, the bulk
	// of a pretty-printed document's text nodes) are interned so identical
	// "\n    " runs share one backing string.
	var pendingBuf []byte
	wsRuns := map[string]string{}
	var pendingOpen bool
	var pendingLine, pendingCol int
	flushPending := func() {
		if !pendingOpen {
			return
		}
		pendingOpen = false
		s := internText(&pendingBuf, wsRuns)
		top := stack[len(stack)-1]
		// Whitespace outside the document element (prolog/epilog "Misc") is
		// not character data in the infoset: a document node has no
		// whitespace text children (fn-innermost-029: //text() counts).
		if top == doc && strings.Trim(s, " \t\r\n") == "" {
			return
		}
		if top.Kind == KindElement {
			// A validating processor treats whitespace-only text directly in
			// an element the DTD declares with ELEMENT CONTENT (no #PCDATA
			// alternative) as ignorable whitespace, stripped by default even
			// with no xsl:strip-space in the stylesheet (id-003/id-036).
			if elemOnly[top.Name.Local] && strings.Trim(s, " \t\r\n") == "" {
				return
			}
			// xsl:strip-space, net of this element's xml:space scope. Uses
			// strings.TrimSpace, not the 4-char XML-whitespace Trim above —
			// deliberately, to match internal/xslt's own stripWalk check
			// byte-for-byte: that function still runs over the finished tree
			// regardless of this early-skip (a safety net, not a
			// replacement), and the two must agree on what "whitespace"
			// means here or a run they disagree about would be stripped by
			// one and survive the other depending purely on which happened
			// to run first.
			if strip != nil && !preserveSpace[len(preserveSpace)-1] &&
				strip(top.Name) && strings.TrimSpace(s) == "" {
				return
			}
		}
		top.Append(&Node{Kind: KindText, Value: s, Line: int32(pendingLine), Col: int32(pendingCol)})
	}

	for {
		startOff := dec.InputOffset()
		// RawToken rather than Token: Token resolves every name to its
		// namespace URI and discards the prefix the author wrote, so a copied
		// element could only ever be given SOME prefix bound to its URI —
		// <Part> under xmlns="some1" plus xmlns:ns1="some1" came out as
		// ns1:Part (copy-4901), and two prefixes sharing one URI collapsed
		// into one (output-0138). The namespace frame stack below already
		// does the resolution; Token's other two services — end-tag matching
		// and no autoclose (Strict) — are re-established here.
		tok, err := dec.RawToken()
		if err == io.EOF {
			break
		}
		if err != nil {
			line, col := lines.at(int(dec.InputOffset()))
			return nil, &ParseError{Line: line, Col: col, Msg: err.Error()}
		}
		line, col := lines.at(int(startOff))
		top := stack[len(stack)-1]

		switch t := tok.(type) {
		case xml.StartElement:
			// A new child starting is a run boundary: any pending text run
			// belongs to the PARENT (top, before this element is pushed) and
			// must close now, not merge with anything that follows.
			flushPending()
			// Intern the tag's own name before anything retains it: Local and
			// the written prefix (Space, in RawToken's naming) both end up
			// stored verbatim into the element's Name by resolveRawName below,
			// and interning here is what makes every occurrence of the same
			// name across the document share one backing byte array instead
			// of each tag paying for its own copy.
			t.Name.Local = intern(nameIntern, t.Name.Local)
			t.Name.Space = intern(nameIntern, t.Name.Space)
			// Build this element's namespace frame from the parent's, applying
			// its own xmlns decls if any — but most elements in a real document
			// declare none, so cloning the parent's map unconditionally (the
			// map is Go-allocated and sized to its exact entry count) wastes an
			// allocation+copy per element for no reason: clone lazily, only on
			// the first decl actually found, and otherwise push the SAME map
			// reference the parent used. Safe because every write below goes
			// through this clone-before-first-write gate, so a shared frame is
			// never mutated — a deeper element that itself needs to add a decl
			// clones its own copy at that point instead.
			frame := nsStack[len(nsStack)-1]
			cloned := false
			var decls []*Node
			if d := attrDefaults[t.Name.Local]; len(d) > 0 {
				t.Attr = withNSDefaults(t.Attr, d) // DTD-defaulted xmlns[:p] declarations
			}
			for _, a := range t.Attr {
				local, isNS := "", false
				switch {
				case a.Name.Space == "xmlns":
					local, isNS = a.Name.Local, true
				case a.Name.Space == "" && a.Name.Local == "xmlns":
					local, isNS = "", true
				}
				if !isNS {
					continue
				}
				if !cloned {
					frame = cloneNS(frame)
					cloned = true
				}
				frame[local] = a.Value
				decls = append(decls, &Node{Kind: KindNamespace, Name: Name{Local: local}, Value: a.Value})
			}
			el := &Node{
				Kind: KindElement,
				Name: resolveRawName(frame, t.Name, false),
				Line: int32(line), Col: int32(col),
			}
			el.SetNSDecls(decls)
			for _, d := range decls {
				d.Parent = el
			}
			for _, a := range t.Attr {
				if a.Name.Space == "xmlns" || (a.Name.Space == "" && a.Name.Local == "xmlns") {
					continue
				}
				an := &Node{
					Kind:  KindAttribute,
					Name:  resolveRawName(frame, a.Name, true),
					Value: a.Value,
					Line:  int32(line), Col: int32(col),
				}
				if m := idKinds[t.Name.Local]; m != nil {
					if k := m[a.Name.Local]; k != 0 {
						an.SetIDKind(k)
					}
				}
				an.Parent = el
				el.Attrs = append(el.Attrs, an)
			}
			if defaults := attrDefaults[t.Name.Local]; len(defaults) > 0 {
				// DTD ATTLIST default/#FIXED values (attribute-0501): supply
				// them for any declared attribute absent from the instance —
				// per the XML infoset, such an attribute IS present with its
				// declared default, exactly as if the author had typed it.
				for name, d := range defaults {
					if isNSDecl(name) {
						continue // bound above, not an attribute
					}
					present := false
					for _, a := range el.Attrs {
						if a.Name.Space == "" && a.Name.Local == name {
							present = true
							break
						}
					}
					if present {
						continue
					}
					el.Attrs = append(el.Attrs, &Node{
						Kind:   KindAttribute,
						Name:   Name{Local: name},
						Value:  d.value,
						Parent: el,
						Line:   int32(line), Col: int32(col),
					})
				}
			}
			top.Append(el)
			stack = append(stack, el)
			nsStack = append(nsStack, frame)
			rawNames = append(rawNames, t.Name)
			// xml:space="preserve" forces true for this element and its
			// descendants until overridden; any other explicit value
			// (including "default") resets to false; absent, inherit the
			// parent's scope unchanged. Checked via the RESOLVED namespace
			// URI (el.Attr), exactly like internal/xslt's own stripWalk, not
			// the written prefix — "xml" is reserved and always bound to
			// this URI, but matching the same way the rest of this engine
			// already does keeps the two checks provably in agreement.
			preserve := preserveSpace[len(preserveSpace)-1]
			if v, ok := el.Attr(xmlNS, "space"); ok {
				preserve = v == "preserve"
			}
			preserveSpace = append(preserveSpace, preserve)

		case xml.EndElement:
			// The closing element's own pending text run must close before
			// it does — flushPending reads stack's CURRENT top, so this must
			// run before stack/preserveSpace are popped below.
			flushPending()
			if len(rawNames) == 0 {
				return nil, &ParseError{Line: line, Col: col, Msg: "unexpected end element </" + rawQName(t.Name) + ">"}
			}
			if open := rawNames[len(rawNames)-1]; open != t.Name {
				return nil, &ParseError{Line: line, Col: col, Msg: "element <" + rawQName(open) + "> closed by </" + rawQName(t.Name) + ">"}
			}
			rawNames = rawNames[:len(rawNames)-1]
			stack = stack[:len(stack)-1]
			nsStack = nsStack[:len(nsStack)-1]
			preserveSpace = preserveSpace[:len(preserveSpace)-1]

		case xml.CharData:
			// Accumulate; the decision of whether/how this becomes a Node is
			// deferred to flushPending, called at the next run boundary (see
			// its own doc comment on pendingText above for why eagerly
			// deciding per-token would be wrong).
			if !pendingOpen {
				pendingOpen = true
				pendingLine, pendingCol = line, col
			}
			pendingBuf = append(pendingBuf, t...)

		case xml.Comment:
			flushPending()
			top.Append(&Node{Kind: KindComment, Value: string(t), Line: int32(line), Col: int32(col)})

		case xml.ProcInst:
			flushPending()
			if t.Target == "xml" {
				continue // XML declaration, not a PI node
			}
			top.Append(&Node{Kind: KindPI, Name: Name{Local: t.Target}, Value: string(t.Inst), Line: int32(line), Col: int32(col)})
		}
	}
	flushPending()

	if len(stack) != 1 {
		return nil, &ParseError{Msg: "unexpected end of document: unclosed elements"}
	}
	assignOrder(doc)
	if len(entSpans) > 0 {
		// Only an external-parsed-entity document needs the reverse (line,col)
		// -> offset lookup applyEntityBases does, so only build its line-start
		// table (one int per line — real memory on a large document) in that
		// rare case, rather than unconditionally for every parse.
		applyEntityBases(doc, entSpans, lineStartOffsets(src))
	}
	return doc, nil
}

// RootElement returns the first element child of a document node (or n itself
// if it is already an element).
func RootElement(n *Node) *Node {
	if n.Kind == KindElement {
		return n
	}
	for _, c := range n.Children {
		if c.Kind == KindElement {
			return c
		}
	}
	return nil
}

// intern returns the single shared copy of s held in m, recording s as that
// copy on its first occurrence.
func intern(m map[string]string, s string) string {
	if v, ok := m[s]; ok {
		return v
	}
	m[s] = s
	return s
}

func cloneNS(m map[string]string) map[string]string {
	out := make(map[string]string, len(m)+2)
	for k, v := range m {
		out[k] = v
	}
	return out
}

// normalizeAttrWhitespace applies the XML attribute-value normalization step
// encoding/xml skips: a LITERAL tab, LF or CR inside a quoted attribute value
// becomes a space. It has to happen on the raw text, because once the decoder
// has resolved &#10; into the very same LF the two are indistinguishable —
// and the character reference must stay a newline (boolean-082: a stylesheet
// attribute broken across two lines reads "font-weight: bold"). Comments,
// CDATA sections, processing instructions and the DOCTYPE are left alone;
// every replacement is byte-for-byte so line/column offsets are unaffected
// (a CRLF pair inside a value becomes two spaces rather than one).
//
// The initial ContainsAny check alone doesn't avoid much: it only screens out
// a document with NO tab/LF/CR anywhere, but every pretty-printed document
// has plenty of newlines between tags, tripping it true, while the actual
// target — a literal control character INSIDE a quoted attribute value — is
// rare. So the []byte copy itself is deferred to the first replacement
// actually found, not done unconditionally up front: scanning stays on src
// throughout (byte-indexing a string costs nothing), and only the specific
// in-quote control-character positions ever touch the lazily-allocated b.
func normalizeAttrWhitespace(src string) string {
	if !strings.ContainsAny(src, "\t\n\r") {
		return src
	}
	var b []byte
	n := len(src)
	i := 0
	for i < n {
		if src[i] != '<' {
			i++
			continue
		}
		rest := src[i:]
		switch {
		case strings.HasPrefix(rest, "<!--"):
			if j := strings.Index(rest[4:], "-->"); j >= 0 {
				i += 4 + j + 3
			} else {
				i = n
			}
			continue
		case strings.HasPrefix(rest, "<![CDATA["):
			if j := strings.Index(rest[9:], "]]>"); j >= 0 {
				i += 9 + j + 3
			} else {
				i = n
			}
			continue
		case strings.HasPrefix(rest, "<?"):
			if j := strings.Index(rest[2:], "?>"); j >= 0 {
				i += 2 + j + 2
			} else {
				i = n
			}
			continue
		case strings.HasPrefix(rest, "<!"):
			// DOCTYPE (with a possible internal subset in [...]) or any other
			// declaration: skip to its closing '>' outside brackets/quotes.
			depth := 0
			var q byte
			j := i + 2
			for j < n {
				c := src[j]
				if q != 0 {
					if c == q {
						q = 0
					}
				} else if c == '"' || c == '\'' {
					q = c
				} else if c == '[' {
					depth++
				} else if c == ']' {
					depth--
				} else if c == '>' && depth <= 0 {
					break
				}
				j++
			}
			i = j + 1
			continue
		}
		// A start or end tag: whitespace inside quoted attribute values.
		var q byte
		j := i + 1
		for j < n {
			c := src[j]
			if q != 0 {
				if c == q {
					q = 0
				} else if c == '\n' || c == '\t' || c == '\r' {
					if b == nil {
						b = []byte(src)
					}
					b[j] = ' '
				}
			} else if c == '"' || c == '\'' {
				q = c
			} else if c == '>' {
				break
			}
			j++
		}
		i = j + 1
	}
	if b == nil {
		return src
	}
	return string(b)
}

// resolveRawName resolves a name as RawToken reports it (the written prefix
// in Space) against the namespace frame in scope: an unprefixed element takes
// the default namespace, an unprefixed attribute none, and a prefix with no
// binding is left as the namespace itself — exactly what Token() used to do
// with it, so the rest of the engine sees the same shape as before.
func resolveRawName(frame map[string]string, raw xml.Name, attr bool) Name {
	if raw.Space == "" {
		if attr {
			return Name{Local: raw.Local}
		}
		return Name{Local: raw.Local, Space: frame[""]}
	}
	if uri, ok := frame[raw.Space]; ok {
		return Name{Local: raw.Local, Space: uri, Prefix: raw.Space}
	}
	return Name{Local: raw.Local, Space: raw.Space}
}

// rawQName spells a RawToken name back out as written.
func rawQName(n xml.Name) string {
	if n.Space != "" {
		return n.Space + ":" + n.Local
	}
	return n.Local
}

// AssignOrder (re)numbers every node in root's subtree in document order, the
// same way parsing a real document does. Constructed/cloned content (an
// xsl:variable/param/function/xsl:sequence body's scratch collector, an
// xsl:copy-of/fn:copy-of clone) is built entirely out of fresh Node structs
// whose zero-value order (0) ties every one of them together — harmless for
// simple traversal/serialization, but it breaks any later document-order-
// dependent operation applied to that tree as a standalone value: sorting by
// Order() can't actually reorder tied nodes into true document position, and
// NodeSet dedup logic that assumes a real sort brings duplicates adjacent
// silently keeps repeats. Call this once a temporary tree is finalized as an
// independently-reachable value (see internal/xslt), before it can reach a
// union/intersect/except, "is"/"<<"/">>" , or a sort/dedup over its content.
func AssignOrder(root *Node) { assignOrder(root) }

// SetOrder stamps one node with an explicit document-order index and tree
// identity. assignOrder numbers a FINISHED tree in one walk; an incremental
// builder (stream.go's StreamReader) never has a finished tree to walk, so it
// stamps each node as it is created instead — in the same sequence, and thus
// to the same effect. See StreamReader.stamp for why leaving order at its
// zero value is a correctness bug rather than a missing optimization.
func SetOrder(n *Node, order int, tree uint64) {
	n.order, n.tree = int32(order), tree
}

// treeSeq hands out Node.tree stamps in creation order (atomic: trees are
// built from many goroutines under the conformance harnesses).
var treeSeq uint64

// ephemeralTreeBit sorts every CONSTRUCTED tree after every retrieved one in
// cross-tree document order, which the spec leaves implementation-defined and
// only requires to be consistent. Creation order alone is not enough here: a
// global variable's temporary tree is built when the variable is evaluated,
// and this engine evaluates globals EAGERLY — so "$var union /a/b" would put
// the variable's nodes before a source document that the stylesheet opened
// later, where a lazy processor puts them after. Sorting by (retrieved before
// constructed, then creation order) gives the lazy processor's answer for a
// source document without changing the relative order of two constructed
// trees (sx-union-012/017/022/035/102, si-fork-118).
const ephemeralTreeBit = uint64(1) << 63

// assignOrder walks the tree in document order assigning indices. Order is:
// node, its namespace nodes, its attribute nodes, then its children.
func assignOrder(root *Node) {
	counter := int32(0)
	tree := atomic.AddUint64(&treeSeq, 1)
	if root != nil && root.Ephemeral() {
		tree |= ephemeralTreeBit
	}
	var walk func(n *Node)
	walk = func(n *Node) {
		n.order = counter
		n.tree = tree
		counter++
		for _, ns := range n.NSDecls() {
			ns.order = counter
			ns.tree = tree
			counter++
		}
		for _, a := range n.Attrs {
			a.order = counter
			a.tree = tree
			counter++
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
}

// linePos converts byte offsets to 1-based (line, col) incrementally, as the
// decoder advances, instead of precomputing every line-start offset up front
// (an []int with one entry per line — tens of MB on a large, heavily
// line-broken document for an array that exists only to serve this lookup).
// Every caller in this package only ever asks about offsets in non-decreasing
// order (parseRaw queries dec.InputOffset() once per token, which only moves
// forward), so each byte of src needs scanning at most once across the
// whole parse, with O(1) extra state instead of an O(line count) array and
// no binary search per lookup.
type linePos struct {
	src       string
	scanned   int // offset scanned up to so far (exclusive)
	line      int // 1-based line number as of `scanned`
	lineStart int // byte offset where `line` began
}

func newLinePos(src string) *linePos {
	return &linePos{src: src, line: 1}
}

// at maps a byte offset to a 1-based (line, col). off must be >= every off
// given to a previous call on the same linePos.
func (p *linePos) at(off int) (int, int) {
	if off > len(p.src) {
		off = len(p.src)
	}
	for p.scanned < off {
		if p.src[p.scanned] == '\n' {
			p.line++
			p.lineStart = p.scanned + 1
		}
		p.scanned++
	}
	return p.line, off - p.lineStart + 1
}

var _ = fmt.Sprintf

// applyEntityBases gives every element and processing instruction that came
// out of an external parsed entity that entity's own location as its intrinsic
// base URI (XML Base / the infoset's [base URI] property). Only the outermost
// such node needs it — NodeBaseURI stops climbing at the first ancestor that
// carries one.
func applyEntityBases(doc *Node, spans []entitySpan, lines []int) {
	if len(spans) == 0 {
		return
	}
	var walk func(n *Node)
	walk = func(n *Node) {
		if n.Kind == KindElement || n.Kind == KindPI {
			off := offsetOf(lines, int(n.Line), int(n.Col))
			for _, sp := range spans {
				if off >= sp.start && off < sp.end {
					if n.EntityBase() == "" {
						n.SetEntityBase(fileuri.FromPath(sp.path))
					}
					break
				}
			}
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(doc)
}

// lineStartOffsets builds the one-int-per-line table offsetOf needs for its
// reverse (line,col) -> offset lookup. Only called for a document that
// actually contains a spliced external parsed entity (applyEntityBases'
// caller) — the common case never pays for this array at all.
func lineStartOffsets(s string) []int {
	offs := []int{0}
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			offs = append(offs, i+1)
		}
	}
	return offs
}

// offsetOf converts a 1-based line/column back to a byte offset.
func offsetOf(lines []int, line, col int) int {
	if line-1 < 0 || line-1 >= len(lines) {
		return -1
	}
	return lines[line-1] + col - 1
}

// maxInternedWS bounds the whitespace runs worth sharing: indentation is short,
// and a long run is rare enough that a map entry for it would only cost memory.
const maxInternedWS = 64

// internText returns the accumulated run as a string and empties the buffer
// for reuse. A short run made only of XML whitespace is looked up in (and added
// to) runs, so repeated indentation allocates once per distinct run, not once
// per text node.
func internText(buf *[]byte, runs map[string]string) string {
	b := *buf
	*buf = b[:0]
	if len(b) == 0 || len(b) > maxInternedWS {
		return string(b)
	}
	for _, c := range b {
		if c != ' ' && c != '\t' && c != '\r' && c != '\n' {
			return string(b)
		}
	}
	if s, ok := runs[string(b)]; ok { // no allocation: map lookup by []byte conversion
		return s
	}
	s := string(b)
	runs[s] = s
	return s
}
