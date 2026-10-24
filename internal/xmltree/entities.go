package xmltree

import (
	"fmt"
	"github.com/tim-riep/go-xslt/internal/fsaccess"
	"path/filepath"
	"strconv"
	"strings"
)

// internalEntities extracts the general entity declarations from a document's
// DOCTYPE internal subset, so encoding/xml (which does not process DTDs) can
// still resolve references to them via Decoder.Entity.
//
// Two XML rules matter here and are both implemented: the FIRST declaration of
// a name wins (later ones are ignored, not overrides), and an entity's
// replacement text may itself contain entity references, which are expanded
// here because encoding/xml substitutes Decoder.Entity values verbatim.
//
// Only internal general entities with a literal value are handled; parameter
// entities (<!ENTITY % …>) and external ones (SYSTEM/PUBLIC) are skipped, and
// replacement text is inserted as character data, not as markup.
// UnparsedEntities returns the names of the unparsed (NDATA) entities declared
// in the document's DOCTYPE internal subset — the xs:ENTITY value space.
func UnparsedEntities(src string) map[string]bool {
	decls := unparsedEntityDecls(src, "", nil)
	if len(decls) == 0 {
		return nil
	}
	out := make(map[string]bool, len(decls))
	for name := range decls {
		out[name] = true
	}
	return out
}

// UnparsedEntity is one <!ENTITY … NDATA …> declaration's external identifier:
// the (possibly relative) system identifier and the public identifier, if any.
type UnparsedEntity struct {
	SystemID string
	PublicID string
}

// unparsedEntityDecls collects the unparsed (NDATA) general entities declared
// by a document: its DOCTYPE internal subset first and — when baseDir is
// non-empty — the external SYSTEM subset after it, so the internal declaration
// of a name wins. Within one subset the FIRST declaration of a name is binding
// (XML §4.2, unparsed-entity-50).
func unparsedEntityDecls(src, baseDir string, files *fsaccess.Guard) map[string]UnparsedEntity {
	out := map[string]UnparsedEntity{}
	if sub, ok := internalSubset(src); ok {
		scanUnparsedEntityDecls(sub, out)
	}
	if baseDir != "" {
		if sysID, ok := externalDTDSystemID(src); ok {
			path := sysID
			if !filepath.IsAbs(path) {
				path = filepath.Join(baseDir, path)
			}
			if data, err := files.ReadFile(path); err == nil {
				scanUnparsedEntityDecls(string(data), out)
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// scanUnparsedEntityDecls records every NDATA entity declaration found in one
// DTD subset into out, never overriding a name already present.
func scanUnparsedEntityDecls(sub string, out map[string]UnparsedEntity) {
	for i := 0; ; {
		j := strings.Index(sub[i:], "<!ENTITY")
		if j < 0 {
			break
		}
		p := i + j + len("<!ENTITY")
		i = p
		p = skipSpace(sub, p)
		if p >= len(sub) || sub[p] == '%' {
			continue // parameter entity
		}
		start := p
		for p < len(sub) && !isXMLSpace(sub[p]) && sub[p] != '>' {
			p++
		}
		name := sub[start:p]
		end := strings.IndexByte(sub[p:], '>')
		if end < 0 {
			break
		}
		decl := sub[p : p+end]
		i = p + end + 1
		if name == "" || !strings.Contains(decl, "NDATA") {
			continue
		}
		if _, seen := out[name]; seen {
			continue // first declaration is binding
		}
		var ent UnparsedEntity
		if k := strings.Index(decl, "PUBLIC"); k >= 0 {
			pub, rest, ok := firstQuotedLiteral(decl[k+len("PUBLIC"):])
			if ok {
				ent.PublicID = strings.Join(strings.Fields(pub), " ")
				if sys, _, ok := firstQuotedLiteral(rest); ok {
					ent.SystemID = sys
				}
			}
		} else if k := strings.Index(decl, "SYSTEM"); k >= 0 {
			if sys, _, ok := firstQuotedLiteral(decl[k+len("SYSTEM"):]); ok {
				ent.SystemID = sys
			}
		}
		out[name] = ent
	}
}

func internalEntities(src string) map[string]string {
	ents, _ := documentEntities(src, "", nil)
	return ents
}

// entityExpansionFloor is the least text entity expansion may produce; see
// expansionLimit.
const entityExpansionFloor = 16 << 20

// expansionLimit bounds how many bytes entity expansion may produce for a
// document of inputLen bytes: eight times the input, but at least 16 MiB.
// Without it a few hundred bytes of mutually referring entity declarations
// ("billion laughs") expand exponentially and take the whole process down
// with them; no ordinary document comes near the bound.
func expansionLimit(inputLen int) int {
	if l := 8 * inputLen; l > entityExpansionFloor {
		return l
	}
	return entityExpansionFloor
}

// errEntityExpansion reports a document whose entities expand past
// expansionLimit.
func errEntityExpansion(limit int) error {
	return &ParseError{Msg: fmt.Sprintf("entity expansion exceeds %d bytes (the limit for this document)", limit)}
}

// documentEntities extracts a document's general entity declarations: the
// DOCTYPE internal subset (if any), plus — when baseDir is non-empty — the
// external subset named by a SYSTEM identifier, resolved against baseDir and
// read from the filesystem (copy-1201/1202: a stylesheet's own DOCTYPE
// referencing a local .dtd file of named character entities, e.g. &egrave;).
// An unreadable/absent external file is silently skipped — the processor does
// not fail a document over an entity set it cannot fetch; any reference to an
// entity it declared just stays unresolved, as it always did before baseDir
// support existed. Internal declarations take precedence over external ones
// (scanEntityDecls never overrides an already-seen name), matching the XML
// rule that the internal subset wins over the external one.
func documentEntities(src, baseDir string, files *fsaccess.Guard) (map[string]string, error) {
	ents, _, err := documentEntitiesRec(src, baseDir, files)
	return ents, err
}

// documentEntitiesRec is documentEntities that also returns the names of the
// declared entities it dropped as recursive (see dropRecursiveEntities), so a
// reference to one can be reported as what it is.
func documentEntitiesRec(src, baseDir string, files *fsaccess.Guard) (map[string]string, map[string]bool, error) {
	ents := map[string]string{}
	inputLen := len(src)
	if sub, ok := internalSubset(src); ok {
		decls, err := expandParamEntities(sub, baseDir, files, expansionLimit(len(src)))
		if err != nil {
			return nil, nil, err
		}
		scanEntityDecls(decls, ents)
	}
	if baseDir != "" {
		if sysID, ok := externalDTDSystemID(src); ok {
			path := sysID
			if !filepath.IsAbs(path) {
				path = filepath.Join(baseDir, path)
			}
			if data, err := files.ReadFile(path); err == nil {
				inputLen += len(data)
				scanEntityDecls(string(data), ents)
			}
		}
	}
	if len(ents) == 0 {
		return nil, nil, nil
	}
	// WFC: No Recursion (XML 1.0 §4.1, on EntityRef): "A parsed entity MUST
	// NOT contain a recursive reference to itself, either directly or
	// indirectly." An entity whose expansion reaches a cycle has no
	// replacement text; it is dropped, so a reference to it in the document
	// fails like an undeclared one (an unreferenced declaration stays
	// harmless — the constraint is on the reference). Dropping them first also
	// keeps the pass loop below from leaving a literal "&a;" in a value.
	rec := dropRecursiveEntities(ents)
	if err := expandEntityValues(ents, expansionLimit(inputLen)); err != nil {
		return nil, nil, err
	}
	// encoding/xml inserts a Decoder.Entity value verbatim, without rescanning it
	// for references, so character references and the five predefined entities
	// have to be resolved here — exactly once, and only after general-entity
	// expansion has reached its fixpoint, so that "&amp;#37;" stays "&#37;".
	for name, val := range ents {
		ents[name] = expandBuiltinRefs(val)
	}
	return ents, rec, nil
}

// entityRefVolume is an upper bound on the text the references to ents in
// body expand to (a reference inside a comment or CDATA section is counted
// too). It stops counting once the total passes limit.
func entityRefVolume(body string, ents map[string]string, limit int) int {
	total := 0
	for i := 0; total <= limit; {
		j := strings.IndexByte(body[i:], '&')
		if j < 0 {
			break
		}
		j += i
		k := strings.IndexByte(body[j:], ';')
		if k < 0 {
			break
		}
		if v, ok := ents[body[j+1:j+k]]; ok {
			total += len(v)
		}
		i = j + 1
	}
	return total
}

// scanEntityDecls scans text for <!ENTITY name "value"> declarations (skipping
// parameter entities and ones with an external id) and records each
// first-seen name into ents without overriding an entry already present —
// letting a caller seed ents with higher-priority declarations first (see
// documentEntities: internal-subset entities must win over external-subset
// ones of the same name).
func scanEntityDecls(text string, ents map[string]string) {
	for i := 0; ; {
		j := strings.Index(text[i:], "<!ENTITY")
		if j < 0 {
			break
		}
		p := i + j + len("<!ENTITY")
		i = p
		p = skipSpace(text, p)
		if p >= len(text) || text[p] == '%' {
			continue // parameter entity
		}
		start := p
		for p < len(text) && !isXMLSpace(text[p]) && text[p] != '>' {
			p++
		}
		name := text[start:p]
		p = skipSpace(text, p)
		if name == "" || p >= len(text) || (text[p] != '"' && text[p] != '\'') {
			continue // external id, or malformed — leave it to the parser
		}
		q := text[p]
		p++
		start = p
		for p < len(text) && text[p] != q {
			p++
		}
		if p >= len(text) {
			break
		}
		if _, seen := ents[name]; !seen {
			ents[name] = text[start:p]
		}
		i = p + 1
	}
}

// externalDTDSystemID extracts the SYSTEM literal from a document's DOCTYPE
// declaration's external ID (<!DOCTYPE root SYSTEM "uri"> or <!DOCTYPE root
// PUBLIC "public-id" "uri">). PUBLIC identifiers are not resolved — only the
// accompanying system literal is used, matching how browsers/processors
// commonly fall back for a PUBLIC id they don't otherwise recognize.
func externalDTDSystemID(src string) (string, bool) {
	d := strings.Index(src, "<!DOCTYPE")
	if d < 0 {
		return "", false
	}
	i := d + len("<!DOCTYPE")
	end := i
	for end < len(src) && src[end] != '[' && src[end] != '>' {
		end++
	}
	decl := src[i:end]
	if j := strings.Index(decl, "SYSTEM"); j >= 0 {
		lit, _, ok := firstQuotedLiteral(decl[j+len("SYSTEM"):])
		return lit, ok
	}
	if j := strings.Index(decl, "PUBLIC"); j >= 0 {
		_, rest, ok := firstQuotedLiteral(decl[j+len("PUBLIC"):])
		if !ok {
			return "", false
		}
		lit, _, ok := firstQuotedLiteral(rest)
		return lit, ok
	}
	return "", false
}

// firstQuotedLiteral returns the content of the first '...'/"..." literal in
// s, plus the remainder of s following its closing quote.
func firstQuotedLiteral(s string) (lit, rest string, ok bool) {
	i := skipSpace(s, 0)
	if i >= len(s) || (s[i] != '"' && s[i] != '\'') {
		return "", "", false
	}
	q := s[i]
	i++
	start := i
	for i < len(s) && s[i] != q {
		i++
	}
	if i >= len(s) {
		return "", "", false
	}
	return s[start:i], s[i+1:], true
}

// expandEntityValues resolves entity references inside replacement text. The
// pass count bounds the recursion depth, so a circular declaration leaves the
// unresolved reference in place instead of looping.
//
// The values together may not grow past limit: declarations that refer to
// each other can double a value on every pass.
func expandEntityValues(ents map[string]string, limit int) error {
	total := 0
	for _, val := range ents {
		total += len(val)
	}
	for pass := 0; pass < 40; pass++ {
		changed := false
		for name, val := range ents {
			// The room this value may grow into: the limit less every other
			// value. Checked while substituting, not after, so a value with
			// many references to a large one is never built past the limit.
			out, hit, ok := substituteEntities(val, ents, limit-(total-len(val)))
			if !ok {
				return errEntityExpansion(limit)
			}
			if hit {
				total += len(out) - len(val)
				ents[name] = out
				changed = true
			}
		}
		if !changed {
			return nil
		}
	}
	return nil
}

// dropRecursiveEntities deletes (and returns the names of) every entity whose replacement
// text refers, directly or through other entities, to an entity on a cycle
// of references. The walk is iterative (a declaration chain is input, and
// nothing may overflow the Go stack) and linear in the total value length.
func dropRecursiveEntities(ents map[string]string) map[string]bool {
	refs := func(val string) []string {
		var out []string
		for i := 0; ; {
			j := strings.IndexByte(val[i:], '&')
			if j < 0 {
				return out
			}
			j += i
			k := strings.IndexByte(val[j:], ';')
			if k < 0 {
				return out
			}
			if name := val[j+1 : j+k]; name != "" {
				if _, ok := ents[name]; ok {
					out = append(out, name)
				}
			}
			i = j + 1
		}
	}
	const (
		unseen = iota
		onPath
		clean
		bad
	)
	state := make(map[string]int, len(ents))
	type frame struct {
		name string
		next []string
	}
	for root := range ents {
		if state[root] != unseen {
			continue
		}
		state[root] = onPath
		stack := []frame{{root, refs(ents[root])}}
		for len(stack) > 0 {
			top := &stack[len(stack)-1]
			if len(top.next) == 0 {
				if state[top.name] == onPath {
					state[top.name] = clean
				}
				stack = stack[:len(stack)-1]
				continue
			}
			n := top.next[0]
			top.next = top.next[1:]
			switch state[n] {
			case unseen:
				state[n] = onPath
				stack = append(stack, frame{n, refs(ents[n])})
			case onPath, bad:
				// A back edge closes a cycle, and every entity on the path
				// reaches it; reaching a bad entity is as bad. Marking stops
				// at the first bad frame: everything below it is bad already,
				// which keeps the walk linear.
				for i := len(stack) - 1; i >= 0 && state[stack[i].name] != bad; i-- {
					state[stack[i].name] = bad
				}
			}
		}
	}
	var dropped map[string]bool
	for name, s := range state {
		if s == bad {
			delete(ents, name)
			if dropped == nil {
				dropped = map[string]bool{}
			}
			dropped[name] = true
		}
	}
	return dropped
}

// expandBuiltinRefs resolves character references and the five predefined
// entities in text, leaving anything else untouched.
func expandBuiltinRefs(s string) string {
	predef := map[string]string{"lt": "<", "gt": ">", "amp": "&", "apos": "'", "quot": `"`}
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] != '&' {
			b.WriteByte(s[i])
			i++
			continue
		}
		end := strings.IndexByte(s[i:], ';')
		if end < 0 {
			b.WriteString(s[i:])
			break
		}
		name := s[i+1 : i+end]
		switch {
		case strings.HasPrefix(name, "#x"), strings.HasPrefix(name, "#X"):
			if r, err := strconv.ParseInt(name[2:], 16, 32); err == nil && r > 0 {
				b.WriteRune(rune(r))
			} else {
				b.WriteString(s[i : i+end+1])
			}
		case strings.HasPrefix(name, "#"):
			if r, err := strconv.ParseInt(name[1:], 10, 32); err == nil && r > 0 {
				b.WriteRune(rune(r))
			} else {
				b.WriteString(s[i : i+end+1])
			}
		default:
			if rep, ok := predef[name]; ok {
				b.WriteString(rep)
			} else {
				b.WriteString(s[i : i+end+1])
			}
		}
		i += end + 1
	}
	return b.String()
}

// substituteEntities replaces the references to ents in s. It stops as
// soon as the result would grow past max — before writing the replacement
// that would cross it — and then reports ok false: the caller must refuse the
// document, and no string longer than max has been built.
func substituteEntities(s string, ents map[string]string, max int) (out string, hit, ok bool) {
	if !strings.Contains(s, "&") {
		return s, false, true
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] != '&' {
			b.WriteByte(s[i])
			i++
			continue
		}
		end := strings.IndexByte(s[i:], ';')
		if end < 0 {
			b.WriteString(s[i:])
			break
		}
		name := s[i+1 : i+end]
		if rep, ok := ents[name]; ok {
			if b.Len()+len(rep) > max {
				return "", hit, false
			}
			b.WriteString(rep)
			hit = true
		} else {
			b.WriteString(s[i : i+end+1])
		}
		i += end + 1
	}
	return b.String(), hit, true
}

// internalSubset returns the text between the '[' and matching ']' of a DOCTYPE
// declaration, skipping over comments and quoted strings so a ']' inside either
// does not end the subset early.
func internalSubset(src string) (string, bool) {
	d := strings.Index(src, "<!DOCTYPE")
	if d < 0 {
		return "", false
	}
	open := strings.IndexByte(src[d:], '[')
	if open < 0 {
		return "", false
	}
	i := d + open + 1
	start := i
	for i < len(src) {
		switch {
		case strings.HasPrefix(src[i:], "<!--"):
			e := strings.Index(src[i:], "-->")
			if e < 0 {
				return "", false
			}
			i += e + 3
		case src[i] == '"' || src[i] == '\'':
			q := src[i]
			i++
			for i < len(src) && src[i] != q {
				i++
			}
			i++
		case src[i] == ']':
			return src[start:i], true
		default:
			i++
		}
	}
	return "", false
}

func skipSpace(s string, i int) int {
	for i < len(s) && isXMLSpace(s[i]) {
		i++
	}
	return i
}

func isXMLSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\r' || c == '\n' }

// externalGeneralEntityDecls returns the EXTERNAL parsed general entities a
// document declares (name -> the file their replacement text lives in),
// resolved against the subset they were declared in. NDATA (unparsed)
// entities are excluded — they are never expanded — as are parameter
// entities and internal ones (handled by documentEntities).
func externalGeneralEntityDecls(src, baseDir string, files *fsaccess.Guard) map[string]string {
	if baseDir == "" {
		return nil
	}
	out := map[string]string{}
	if sub, ok := internalSubset(src); ok {
		scanExternalEntityDecls(sub, baseDir, files, out)
	}
	if sysID, ok := externalDTDSystemID(src); ok {
		path := sysID
		if !filepath.IsAbs(path) {
			path = filepath.Join(baseDir, path)
		}
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
		if data, err := files.ReadFile(path); err == nil {
			scanExternalEntityDecls(string(data), filepath.Dir(path), files, out)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func scanExternalEntityDecls(text, baseDir string, files *fsaccess.Guard, out map[string]string) {
	for i := 0; ; {
		j := strings.Index(text[i:], "<!ENTITY")
		if j < 0 {
			return
		}
		p := i + j + len("<!ENTITY")
		i = p
		p = skipSpace(text, p)
		if p >= len(text) || text[p] == '%' {
			continue // parameter entity
		}
		start := p
		for p < len(text) && !isXMLSpace(text[p]) && text[p] != '>' {
			p++
		}
		name := text[start:p]
		end := strings.IndexByte(text[p:], '>')
		if end < 0 {
			return
		}
		decl := text[p : p+end]
		i = p + end + 1
		if name == "" || strings.Contains(decl, "NDATA") {
			continue
		}
		if _, seen := out[name]; seen {
			continue // first declaration binds
		}
		sys := ""
		if k := strings.Index(decl, "PUBLIC"); k >= 0 {
			if _, rest, ok := firstQuotedLiteral(decl[k+len("PUBLIC"):]); ok {
				sys, _, _ = firstQuotedLiteral(rest)
			}
		} else if k := strings.Index(decl, "SYSTEM"); k >= 0 {
			sys, _, _ = firstQuotedLiteral(decl[k+len("SYSTEM"):])
		}
		if sys == "" {
			continue
		}
		if !filepath.IsAbs(sys) {
			sys = filepath.Join(baseDir, sys)
		}
		// The recorded path becomes the entity's BASE URI ("file://"+path —
		// see applyEntityBases), so it has to be absolute: a relative baseDir
		// (the conformance harness passes one) otherwise produced a nonsense
		// "file://../../…" that no relative xml:base could resolve against
		// (base-uri-051).
		if abs, err := filepath.Abs(sys); err == nil {
			sys = abs
		}
		out[name] = sys
	}
}

// expandExternalEntities textually substitutes references to external parsed
// general entities in a document's BODY (everything after the DOCTYPE
// declaration). Go's encoding/xml can only supply an entity's replacement as
// character data, so an external entity whose content is markup — the normal
// case — has to be spliced into the source before tokenizing. Each
// replacement's own text declaration is stripped, and expansion repeats so an
// entity referencing another one resolves too.
//
// The spliced document may not grow past expansionLimit of the input (the
// document plus the entity files read).
func expandExternalEntities(src, baseDir string, files *fsaccess.Guard) (string, []entitySpan, error) {
	paths := externalGeneralEntityDecls(src, baseDir, files)
	reps := map[string]string{}
	inputLen := len(src)
	for name, path := range paths {
		b, err := files.ReadFile(path)
		if err != nil {
			continue
		}
		inputLen += len(b)
		reps[name] = stripTextDecl(string(b))
	}
	// An INTERNAL general entity whose replacement text is markup gets the
	// same textual splice, for the same reason — the decoder would hand
	// "<t:text>" back as character data (whitespace-011's &br;). Its literal
	// is spliced RAW: a general-entity reference inside an EntityValue is
	// bypassed at declaration time (XML §4.4.7) and only resolved where the
	// entity is included, so the &quot; in streamable-137's macro must reach
	// the decoder as a reference, not as a bare quote that would end the
	// attribute early. Plain-text entities keep going through Decoder.Entity
	// (whose values ARE pre-expanded, as verbatim substitution needs).
	if sub, ok := internalSubset(src); ok {
		raw := map[string]string{}
		decls, err := expandParamEntities(sub, baseDir, files, expansionLimit(inputLen))
		if err != nil {
			return "", nil, err
		}
		scanEntityDecls(decls, raw)
		for name, val := range raw {
			if _, ok := reps[name]; ok {
				continue
			}
			if strings.Contains(val, "<") {
				reps[name] = val
			}
		}
	}
	if len(reps) == 0 {
		return src, nil, nil
	}
	head := doctypeEnd(src)
	var spans []entitySpan
	budget := expansionLimit(inputLen)
	body := spliceEntities(src[head:], reps, paths, head, 0, &spans, &budget)
	if budget < 0 {
		return "", nil, errEntityExpansion(expansionLimit(inputLen))
	}
	return src[:head] + body, spans, nil
}

// spliceEntities replaces every reference to a known external entity in text
// with that entity's replacement, recursively, recording where each
// replacement landed. offset is text's absolute position in the document being
// built, so the recorded spans are absolute. A nested entity's span is
// appended BEFORE its container's, which makes the first matching span the
// innermost one. Every replacement spliced in is charged to budget; once it
// goes negative the splicing stops (the caller reports the error).
func spliceEntities(text string, reps, paths map[string]string, offset, depth int, spans *[]entitySpan, budget *int) string {
	if depth > 8 || *budget < 0 || !strings.Contains(text, "&") {
		return text
	}
	var b strings.Builder
	i := 0
	for {
		j := strings.IndexByte(text[i:], '&')
		if j < 0 {
			b.WriteString(text[i:])
			break
		}
		j += i
		k := strings.IndexByte(text[j:], ';')
		if k < 0 {
			b.WriteString(text[i:])
			break
		}
		name := text[j+1 : j+k]
		rep, ok := reps[name]
		if !ok {
			b.WriteString(text[i : j+k+1])
			i = j + k + 1
			continue
		}
		b.WriteString(text[i:j])
		begin := offset + b.Len()
		if *budget -= len(rep); *budget < 0 {
			return ""
		}
		b.WriteString(spliceEntities(rep, reps, paths, begin, depth+1, spans, budget))
		if path := paths[name]; path != "" {
			// Only an EXTERNAL entity's text carries its own base URI.
			*spans = append(*spans, entitySpan{start: begin, end: offset + b.Len(), path: path})
		}
		i = j + k + 1
	}
	return b.String()
}

// entitySpan records where one external entity's replacement text landed in
// the expanded source, so the nodes parsed out of it can be given the
// entity's own base URI.
type entitySpan struct {
	start, end int
	path       string
}

// doctypeEnd returns the index just past a document's DOCTYPE declaration
// (0 when there is none), so entity expansion never touches the DTD itself.
func doctypeEnd(src string) int {
	d := strings.Index(src, "<!DOCTYPE")
	if d < 0 {
		return 0
	}
	i := d + len("<!DOCTYPE")
	for i < len(src) {
		switch {
		case src[i] == '[':
			// Skip the internal subset, then the closing '>'.
			if sub, ok := internalSubset(src); ok {
				i = strings.Index(src, sub) + len(sub)
			}
			for i < len(src) && src[i] != '>' {
				i++
			}
			if i >= len(src) {
				return len(src) // unterminated: the tokenizer reports it
			}
			return i + 1
		case src[i] == '"' || src[i] == '\'':
			q := src[i]
			i++
			for i < len(src) && src[i] != q {
				i++
			}
			i++
		case src[i] == '>':
			return i + 1
		default:
			i++
		}
	}
	return 0
}

// stripTextDecl removes a leading XML/text declaration from an external
// entity's replacement text (an external parsed entity may begin with one,
// and it is not part of the replacement).
func stripTextDecl(s string) string {
	t := strings.TrimLeft(s, "\uFEFF \t\r\n")
	if !strings.HasPrefix(t, "<?xml") {
		return s
	}
	if end := strings.Index(t, "?>"); end >= 0 {
		return t[end+2:]
	}
	return s
}

// expandParamEntities resolves parameter-entity references (%name;) inside a
// DTD subset: an internal parameter entity contributes its literal value, an
// external one the contents of the file it names (resolved against baseDir).
// This is what makes declarations living in a separate .dtd reachable from the
// internal subset (whitespace-011: "<!ENTITY % ext SYSTEM 'entity.ent'>%ext;").
func expandParamEntities(sub, baseDir string, files *fsaccess.Guard, limit int) (string, error) {
	if !strings.Contains(sub, "%") {
		return sub, nil
	}
	vals := map[string]string{}
	for i := 0; ; {
		j := strings.Index(sub[i:], "<!ENTITY")
		if j < 0 {
			break
		}
		p := i + j + len("<!ENTITY")
		i = p
		p = skipSpace(sub, p)
		if p >= len(sub) || sub[p] != '%' {
			continue // a general entity
		}
		p = skipSpace(sub, p+1)
		start := p
		for p < len(sub) && !isXMLSpace(sub[p]) && sub[p] != '>' {
			p++
		}
		name := sub[start:p]
		end := strings.IndexByte(sub[p:], '>')
		if end < 0 {
			break
		}
		decl := sub[p : p+end]
		i = p + end + 1
		if name == "" {
			continue
		}
		if lit, _, ok := firstQuotedLiteral(decl); ok && !strings.Contains(decl, "SYSTEM") && !strings.Contains(decl, "PUBLIC") {
			vals[name] = lit
			continue
		}
		if baseDir == "" {
			continue
		}
		sys := ""
		if k := strings.Index(decl, "PUBLIC"); k >= 0 {
			if _, rest, ok := firstQuotedLiteral(decl[k+len("PUBLIC"):]); ok {
				sys, _, _ = firstQuotedLiteral(rest)
			}
		} else if k := strings.Index(decl, "SYSTEM"); k >= 0 {
			sys, _, _ = firstQuotedLiteral(decl[k+len("SYSTEM"):])
		}
		if sys == "" {
			continue
		}
		path := sys
		if !filepath.IsAbs(path) {
			path = filepath.Join(baseDir, path)
		}
		if data, err := files.ReadFile(path); err == nil {
			vals[name] = stripTextDecl(string(data))
		}
	}
	if len(vals) == 0 {
		return sub, nil
	}
	for pass := 0; pass < 4; pass++ {
		changed := false
		for name, val := range vals {
			ref := "%" + name + ";"
			if n := strings.Count(sub, ref); n > 0 {
				// The size of the result is known before it is built.
				if len(sub)+n*(len(val)-len(ref)) > limit {
					return "", errEntityExpansion(limit)
				}
				sub = strings.ReplaceAll(sub, ref, val)
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	return sub, nil
}
