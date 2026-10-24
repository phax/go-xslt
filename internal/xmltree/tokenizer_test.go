package xmltree

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// The tokenizer must behave exactly like the encoding/xml decoder parseRaw
// used before it (Strict, Entity map, charsetReader, RawToken): the same
// tokens at the same offsets, the same error at the same offset. These tests
// compare the two.

// tokRec is one token as parseRaw consumes it, plus where it started.
type tokRec struct {
	Off    int
	Kind   string
	Name   xml.Name
	Attrs  []xml.Attr
	Text   string // character data, comment, PI instruction, directive
	Target string
}

// tokRun is a whole tokenization: the tokens, then how it ended.
type tokRun struct {
	Toks   []tokRec
	Err    string // "" at a clean io.EOF
	ErrOff int
}

const maxCompareTokens = 1 << 20

func stdTokens(src string, ents map[string]string) tokRun {
	dec := xml.NewDecoder(strings.NewReader(src))
	dec.Strict = true
	dec.CharsetReader = charsetReader
	dec.Entity = ents
	var run tokRun
	for len(run.Toks) < maxCompareTokens {
		off := int(dec.InputOffset())
		tok, err := dec.RawToken()
		if err == io.EOF {
			run.ErrOff = int(dec.InputOffset())
			return run
		}
		if err != nil {
			run.Err, run.ErrOff = err.Error(), int(dec.InputOffset())
			return run
		}
		r := tokRec{Off: off}
		switch t := tok.(type) {
		case xml.StartElement:
			r.Kind, r.Name, r.Attrs = "start", t.Name, t.Attr
			if len(r.Attrs) == 0 {
				r.Attrs = nil
			}
		case xml.EndElement:
			r.Kind, r.Name = "end", t.Name
		case xml.CharData:
			r.Kind, r.Text = "text", string(t)
		case xml.Comment:
			r.Kind, r.Text = "comment", string(t)
		case xml.ProcInst:
			r.Kind, r.Target, r.Text = "pi", t.Target, string(t.Inst)
		case xml.Directive:
			r.Kind, r.Text = "directive", string(t)
		default:
			panic(fmt.Sprintf("unexpected token %T", tok))
		}
		run.Toks = append(run.Toks, r)
	}
	return run
}

func ownTokens(src string, ents map[string]string) tokRun {
	tz := newTokenizer(src, ents)
	var run tokRun
	for len(run.Toks) < maxCompareTokens {
		off := tz.pos
		kind, err := tz.next()
		if err == io.EOF {
			run.ErrOff = tz.pos
			return run
		}
		if err != nil {
			run.Err, run.ErrOff = err.Error(), tz.pos
			return run
		}
		r := tokRec{Off: off}
		switch kind {
		case tokStart:
			r.Kind, r.Name = "start", tz.name
			if len(tz.attrs) > 0 {
				r.Attrs = append([]xml.Attr(nil), tz.attrs...)
			}
		case tokEnd:
			r.Kind, r.Name = "end", tz.name
		case tokText:
			r.Kind = "text"
			if tz.raw {
				r.Text = tz.str
			} else {
				r.Text = string(tz.textBytes())
			}
		case tokComment:
			r.Kind, r.Text = "comment", tz.str
		case tokPI:
			r.Kind, r.Target, r.Text = "pi", tz.target, tz.str
		case tokDirective:
			r.Kind, r.Text = "directive", string(tz.data)
		default:
			panic(fmt.Sprintf("unexpected token kind %d", kind))
		}
		run.Toks = append(run.Toks, r)
	}
	return run
}

// tokenizerSource is the string parseRaw hands the tokenizer, and the entity
// map it uses (no base directory: nothing is read from disk). ok is false
// when one of parseRaw's pre-passes refuses src (entity expansion past
// expansionLimit) — the tokenizer never sees such a document.
//
// With fuzzing set, a document that declares entities skips the expanding
// pre-passes: a billion-laughs shape, which the fuzzer finds within minutes,
// expands to expansionLimit (16 MiB) before it is refused, which would slow
// every such input down for nothing the tokenizer is tested on.
func tokenizerSource(src string, fuzzing bool) (prepared string, ents map[string]string, ok bool) {
	if fuzzing && strings.Contains(src, "<!ENTITY") {
		// The tokenizer only looks values up; any map will do, so take the
		// declared values unexpanded (scanning them is linear) and leave out
		// the expanding pre-passes.
		src = decodeBOM(src)
		var raw map[string]string
		if sub, ok := internalSubset(src); ok {
			raw = map[string]string{}
			scanEntityDecls(sub, raw)
		}
		return downgradeXMLVersion(normalizeAttrWhitespace(src)), raw, true
	}
	src = decodeBOM(src)
	src, _, err := expandExternalEntities(src, "", nil)
	if err != nil {
		return "", nil, false
	}
	src = normalizeAttrWhitespace(src)
	ents, err = documentEntities(src, "", nil)
	if err != nil {
		return "", nil, false
	}
	return downgradeXMLVersion(src), ents, true
}

// compareTokenizers reports the first difference between the two
// tokenizations of src, raw and as parseRaw prepares it (fuzzing: see
// tokenizerSource).
func compareTokenizers(src string, fuzzing bool) error {
	type input struct {
		name string
		src  string
		ents map[string]string
	}
	inputs := []input{{"raw", src, nil}}
	if prepared, ents, ok := tokenizerSource(src, fuzzing); ok {
		inputs = append(inputs, input{"raw+entities", src, ents}, input{"prepared", prepared, ents})
	}
	for _, in := range inputs {
		want, got := stdTokens(in.src, in.ents), ownTokens(in.src, in.ents)
		if reflect.DeepEqual(want, got) {
			continue
		}
		n := min(len(want.Toks), len(got.Toks))
		for i := 0; i < n; i++ {
			if !reflect.DeepEqual(want.Toks[i], got.Toks[i]) {
				return fmt.Errorf("%s: token %d differs:\n encoding/xml %+v\n tokenizer    %+v", in.name, i, want.Toks[i], got.Toks[i])
			}
		}
		if len(want.Toks) != len(got.Toks) {
			return fmt.Errorf("%s: %d tokens from encoding/xml (end %q at %d), %d from the tokenizer (end %q at %d)",
				in.name, len(want.Toks), want.Err, want.ErrOff, len(got.Toks), got.Err, got.ErrOff)
		}
		return fmt.Errorf("%s: end differs: encoding/xml %q at %d, tokenizer %q at %d", in.name, want.Err, want.ErrOff, got.Err, got.ErrOff)
	}
	return nil
}

// tokenizerSeeds are hand-written edge cases.
var tokenizerSeeds = []string{
	"",
	"<a/>",
	"<a>text</a>",
	"\uFEFF<a/>",
	"\xEF\xBB\xBF<?xml version='1.0'?><a/>",
	"\xFF\xFE<\x00a\x00/\x00>\x00",
	"\xFE\xFF\x00<\x00a\x00/\x00>",
	`<?xml version="1.0" encoding="UTF-8"?><a/>`,
	`<?xml version="1.0" encoding="utf-8"?><a/>`,
	`<?xml version="1.0" encoding="ISO-8859-1"?><a>caf` + "\xe9" + `</a>`,
	`<?xml version="1.0" encoding="latin1"?><a b="` + "\xe9\xff" + `">` + "\xe9" + `</a>`,
	`<?xml version="1.0" encoding="windows-1252"?><a>` + "\x80\x9f" + `</a>`,
	`<?xml version="1.0" encoding="Shift_JIS"?><a>` + "\x82\xa0" + `</a>`,
	`<?xml version="1.0" encoding="EUC-JP"?><a>` + "\xa4\xa2" + `</a>`,
	`<?xml version="1.0" encoding="UTF-16"?><a/>`,
	`<?xml version="1.0" encoding="bogus-charset"?><a/>`,
	`<?xml version="1.0" encoding="latin1"?><a/><?xml encoding="latin1"?><b>` + "\xe9" + `</b>`,
	`<?xml version="1.1"?><a/>`,
	`<?xml version='1.1' encoding='latin1'?><a/>`,
	`<?xml version="2.0"?><a/>`,
	`<?xml encoding="latin1" version="1.0"?><a>` + "\xc3" + `</a>`,
	`<?xml version="1.0" encoding=latin1?><a/>`,
	`<?xml?><a/>`,
	`<?xml version="1.0"?>`,
	"<a>\r\n</a>",
	// Found by fuzzing: a byte right after a \r skips the fast path, and
	// must still get the character check.
	"<![CDATA[\r\xe9]]>",
	"<a>\r\x01</a>",
	"<a b='\r\xff'/>",
	"<a>x\ry\r\r\nz\n\r</a>",
	"<a b='x\r\ny\rz'>\r</a>",
	"<a b=\"x\ty\nz\">\t</a>",
	"<a><![CDATA[x\r\ny]]></a>",
	"<a><![CDATA[]]></a>",
	"<a><![CDATA[]]]]></a>",
	"<a><![CDATA[a]]b]]]>c</a>",
	"<a><![CDATA[x",
	"<a><![CDAT[x]]></a>",
	"<a>]]></a>",
	"<a>]]]></a>",
	"<a>] ]></a>",
	"<a b=']]>'/>",
	"<a>&lt;&gt;&amp;&apos;&quot;</a>",
	"<a b='&lt;&gt;&amp;&apos;&quot;'/>",
	"<a>&#65;&#x41;&#X41;&#x;&#;&#65</a>",
	"<a>&#0;</a>",
	"<a>&#x110000;</a>",
	"<a>&#xD800;</a>",
	"<a>&#xFFFE;</a>",
	"<a>&#99999999999999999999;</a>",
	"<a>&#9;&#10;&#13;</a>",
	"<a>&unknown;</a>",
	"<a>&amp</a>",
	"<a>& b</a>",
	"<a>&;</a>",
	"<a>&#</a>",
	"<a>&",
	"<a>&am",
	"<a>&1a;</a>",
	"<a>&a:b;</a>",
	"<a>&\u00e9;</a>",
	"<a b='&unknown;'/>",
	"<!DOCTYPE a [<!ENTITY e 'v'><!ENTITY f '&#60;b/&#62;'>]><a>&e;&f;</a>",
	"<!DOCTYPE a [<!ENTITY e 'x&#0;y'>]><a>&e;</a>",
	"<!DOCTYPE a [<!ENTITY e '\xc3'>]><a>&e;\xa9</a>",
	"<!DOCTYPE a [<!ENTITY e '\xa9'>]><a>\xc3&e;</a>",
	"<!DOCTYPE a [<!ENTITY e 'x'>]><a b='&e;'/>",
	"<!DOCTYPE a [<!ATTLIST a b CDATA 'd'>]><a/>",
	"<!DOCTYPE a SYSTEM 'a.dtd'><a/>",
	"<!DOCTYPE a PUBLIC '-//x//y' \"a.dtd\" [<!-- c > -->]><a/>",
	"<!DOCTYPE a [<!ELEMENT a (#PCDATA)> <!-- a -- b --> <? pi ?>]><a/>",
	"<!DOCTYPE a [ '<' \"]>\" ]><a/>",
	"<!DOCTYPE a [<!-",
	"<!DOCTYPE[<!ENTITY0 '<']", // expandExternalEntities panics on it (see tokenizerSource)
	"<!DOCTYPE",
	"<!>",
	"<!x>",
	"<!-- c -->",
	"<!---->",
	"<!----->",
	"<!-- a -- b -->",
	"<!-- a --->",
	"<!-- a ->",
	"<!- x -->",
	"<!--",
	"<!--x--",
	"<!-- \xff\x00 -->",
	"<?pi?>",
	"<?pi data?>",
	"<?pi   data  ? >?>",
	"<?pi",
	"<?pi ",
	"<??>",
	"<? pi?>",
	"<?xml-stylesheet href='a'?><a/>",
	"<?XML version='1.0'?><a/>",
	"<?pi \xff\x00?>",
	"<a></b>",
	"<a></a b>",
	"<a></a >",
	"<a></ a>",
	"</a>",
	"<a/><b/>",
	"<a b='1' b='2'/>",
	"<a b='1'c='2'/>",
	"<a b = '1' />",
	"<a b/>",
	"<a b=1/>",
	"<a b='1/>",
	"<a b=\"1'/>",
	"<a b='<'/>",
	"<a b='>'/>",
	"<a b='\"'/>",
	"<a b=\"'\"/>",
	"<a / >",
	"<a/ >",
	"<a",
	"<a ",
	"<a b",
	"<a b=",
	"<a b='",
	"<a b='x",
	"<a b='x'",
	"<a b='\xff",
	"<",
	"< a/>",
	"<1a/>",
	"<-a/>",
	"<.a/>",
	"<a.b-c_d:e/>",
	"<:a/>",
	"<a:/>",
	"<a:b:c/>",
	"<p:a xmlns:p='u'/>",
	"<p:a/>",
	"<a xmlns='u' xmlns:p='v' p:b='1' xmlns:q=''/>",
	"<a xmlns:xml='http://www.w3.org/XML/1998/namespace'/>",
	"<a xmlns:='u'/>",
	"<xmlns:a/>",
	"<a\u00e9/>",
	"<\u00e9/>",
	"<\u00b7/>",
	"<a\u00b7/>",
	"<\u4e00/>",
	"<\U00010000/>",
	"<a\U00010000/>",
	"<\xff/>",
	"<a\xff/>",
	"<a\xc3/>",
	"<a>\xff</a>",
	"<a>\xc3</a>",
	"<a>\xed\xa0\x80</a>",
	"<a>\xef\xbf\xbe</a>",
	"<a>\xef\xbf\xbd</a>",
	"<a>\x00</a>",
	"<a>\x01\x08\x0b\x0c\x1f</a>",
	"<a>\x7f\u0085\u2028</a>",
	"<a>\U0010FFFF\U0001F600</a>",
	"<a b='\x01'/>",
	"<a b='\xff'/>",
	"text",
	"text<a/>",
	"<a/>text",
	"<a/>\xff",
	"  \n<a/>\n  ",
	"<a>x</a>]]>",
	strings.Repeat("<a>", MaxDepth) + strings.Repeat("</a>", MaxDepth),
	strings.Repeat("<a>", MaxDepth+1) + strings.Repeat("</a>", MaxDepth+1),
	"<a>" + strings.Repeat("x", 70000) + "&amp;" + strings.Repeat("y", 70000) + "</a>",
	"<a b='" + strings.Repeat("v", 5000) + "'/>",
}

func TestTokenizerSeeds(t *testing.T) {
	for i, s := range tokenizerSeeds {
		if err := compareTokenizers(s, false); err != nil {
			t.Errorf("seed %d %q: %v", i, abbreviate(s), err)
		}
	}
}

func abbreviate(s string) string {
	if len(s) > 120 {
		return s[:120] + "…"
	}
	return s
}

// suiteFiles lists the XML files of the W3C suites (absent in a fresh clone).
func suiteFiles(maxSize int64) []string {
	root := filepath.Join("..", "..", "test", "conformance")
	var out []string
	for _, dir := range []string{"xslt30-test", "qt3tests", "xsdtests", "xmlconf"} {
		base := filepath.Join(root, dir)
		if r, err := filepath.EvalSymlinks(base); err == nil {
			base = r
		}
		_ = filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			switch strings.ToLower(filepath.Ext(p)) {
			case ".xml", ".xsl", ".xslt", ".xsd", ".dtd", ".ent":
			default:
				if dir != "xmlconf" { // its documents have every extension
					return nil
				}
			}
			if info, err := d.Info(); err == nil && info.Size() <= maxSize {
				out = append(out, p)
			}
			return nil
		})
	}
	return out
}

// TestTokenizerMatchesEncodingXMLOnSuites runs every file of the W3C suites
// through both tokenizers.
func TestTokenizerMatchesEncodingXMLOnSuites(t *testing.T) {
	if testing.Short() {
		t.Skip("walks the W3C suites")
	}
	files := suiteFiles(1 << 62)
	if len(files) == 0 {
		t.Skip("W3C suites not present")
	}
	bad := 0
	for _, p := range files {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if err := compareTokenizers(string(b), false); err != nil {
			bad++
			if bad <= 20 {
				t.Errorf("%s: %v", p, err)
			}
		}
	}
	t.Logf("%d files compared, %d differ", len(files), bad)
}

// FuzzTokenizer compares the tokenizer with encoding/xml on arbitrary input.
// XMLTREE_FUZZ_SUITES=1 adds every W3C suite file up to 256 KB to the seeds.
//
//	go test ./internal/xmltree/ -run '^$' -fuzz FuzzTokenizer -fuzztime 30m
func FuzzTokenizer(f *testing.F) {
	for _, s := range tokenizerSeeds {
		if len(s) <= 1<<16 {
			f.Add(s)
		}
	}
	if os.Getenv("XMLTREE_FUZZ_SUITES") != "" {
		for _, p := range suiteFiles(256 << 10) {
			if b, err := os.ReadFile(p); err == nil {
				f.Add(string(b))
			}
		}
	}
	f.Fuzz(func(t *testing.T, s string) {
		if err := compareTokenizers(s, true); err != nil {
			t.Fatal(err)
		}
	})
}

// TestNameTablesMatchEncodingXML checks isXMLName against encoding/xml for
// every rune, as a name's first character and as a later one.
func TestNameTablesMatchEncodingXML(t *testing.T) {
	accepts := func(name string) bool {
		d := xml.NewDecoder(strings.NewReader("<" + name + "/>"))
		for {
			if _, err := d.RawToken(); err != nil {
				return err == io.EOF
			}
		}
	}
	step := rune(1)
	if testing.Short() {
		step = 7
	}
	for r := rune(0); r <= 0x10FFFF; r += step {
		if r >= 0xD800 && r <= 0xDFFF || r == '<' || r == '/' || r == '>' {
			continue
		}
		if r > 0xFFFF && r%97 != 0 { // the tables are BMP-only; sample the rest
			continue
		}
		for _, name := range []string{string(r), "a" + string(r), "a" + string(r) + "b"} {
			if want, got := accepts(name), isXMLName(name) && !strings.ContainsAny(name, " \t\r\n=\"'"); want != got {
				// isXMLName judges a whole name; the decoder's readName stops
				// at a non-name ASCII byte first, so only compare where both
				// see the same bytes.
				if r < 0x80 && !nameByte[r] {
					continue
				}
				t.Errorf("name %q (U+%04X): encoding/xml %v, isXMLName %v", name, r, want, got)
			}
		}
	}
}
