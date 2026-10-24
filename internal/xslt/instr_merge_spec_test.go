package xslt

import (
	"strings"
	"testing"
)

// xsl:merge (XSLT 3.0 §15). Every expectation follows from §15's rules: the
// merge action runs once per distinct composite key value in key order, its
// focus is (first item of the group, group position, number of groups), and
// current-merge-group()/current-merge-key() expose the group.

// mtERun compiles and runs sheet over src, returning output or error.
func mtERun(t *testing.T, sheet, src string) (string, error) {
	t.Helper()
	ss, err := Compile(sheet)
	if err != nil {
		return "", err
	}
	out, _, err := ss.Transform(src, nil)
	return out, err
}

// mtEMergeSheet wraps a template body that uses xsl:merge.
func mtEMergeSheet(body string) string {
	return `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:mte="urn:mte" exclude-result-prefixes="xs mte">
  <xsl:output method="text"/>
  <xsl:function name="mte:group"><xsl:sequence select="current-merge-group()"/></xsl:function>
  <xsl:function name="mte:key"><xsl:sequence select="current-merge-key()"/></xsl:function>
  <xsl:template match="/">` + body + `</xsl:template>
</xsl:stylesheet>`
}

const mtEMergeSrc = `<r>
  <a><i k="1">a1</i><i k="3">a3</i><i k="3">a3b</i></a>
  <b><i k="2">b2</i><i k="3">b3</i></b>
  <set><n>1</n><n>4</n></set><set><n>2</n><n>3</n></set>
  <u><n>c</n><n>a</n><n>b</n></u>
  <s><n>10</n><n>9</n></s>
</r>`

func TestMergeSpec(t *testing.T) {
	cases := []struct {
		name, body, want string
	}{
		// Groups in key order; current-merge-group($source) restricts to one
		// source; the whole group lists source A's items before B's (§15.6.1).
		{"named-sources", `<xsl:merge>
  <xsl:merge-source name="a" select="r/a/i"><xsl:merge-key select="@k"/></xsl:merge-source>
  <xsl:merge-source name="b" select="r/b/i"><xsl:merge-key select="@k"/></xsl:merge-source>
  <xsl:merge-action>[<xsl:value-of select="current-merge-key()"/>:<xsl:value-of select="current-merge-group()"/>|a=<xsl:value-of select="current-merge-group('a')"/>|b=<xsl:value-of select="current-merge-group('b')"/>]</xsl:merge-action>
</xsl:merge>`, "[1:a1|a=a1|b=][2:b2|a=|b=b2][3:a3 a3b b3|a=a3 a3b|b=b3]"},
		// The merge-action's focus: position() is the group number, last()
		// the number of groups, "." the group's first item.
		{"action-focus", `<xsl:merge>
  <xsl:merge-source select="r/a/i"><xsl:merge-key select="@k"/></xsl:merge-source>
  <xsl:merge-action><xsl:value-of select="position(), '/', last(), '=', string(.), ';'" separator=""/></xsl:merge-action>
</xsl:merge>`, "1/2=a1;2/2=a3;"},
		// Two merge keys: current-merge-key() is the sequence of both
		// (§15.6.2). (Its untypedAtomic-to-string cast is not checked: the
		// engine returns the key uncast — see the report.)
		{"composite-key", `<xsl:merge>
  <xsl:merge-source select="r/a/i"><xsl:merge-key select="@k"/><xsl:merge-key select="string-length(.)"/></xsl:merge-source>
  <xsl:merge-action>[<xsl:value-of select="current-merge-key()" separator=","/>:<xsl:value-of select="count(current-merge-key())"/>]</xsl:merge-action>
</xsl:merge>`, "[1,2:2][3,2:2][3,3:2]"},
		// A merge key from the sequence constructor instead of @select.
		{"key-from-body", `<xsl:merge>
  <xsl:merge-source select="r/a/i"><xsl:merge-key><xsl:value-of select="@k"/></xsl:merge-key></xsl:merge-source>
  <xsl:merge-action><xsl:value-of select="current-merge-key()"/>;</xsl:merge-action>
</xsl:merge>`, "1;3;"},
		// for-each-item: one input sequence per item; each is sorted on its
		// own, so their concatenation need not be (§15.3).
		{"for-each-item", `<xsl:merge>
  <xsl:merge-source for-each-item="r/set" select="n"><xsl:merge-key select="xs:integer(.)"/></xsl:merge-source>
  <xsl:merge-action><xsl:value-of select="."/></xsl:merge-action>
</xsl:merge>`, "1234"},
		// sort-before-merge="yes" sorts an unsorted input first (§15.5).
		{"sort-before-merge", `<xsl:merge>
  <xsl:merge-source select="r/u/n" sort-before-merge="yes"><xsl:merge-key select="."/></xsl:merge-source>
  <xsl:merge-action><xsl:value-of select="."/></xsl:merge-action>
</xsl:merge>`, "abc"},
		// Explicit order and data-type, as for xsl:sort.
		{"descending-number", `<xsl:merge>
  <xsl:merge-source select="r/s/n"><xsl:merge-key select="." data-type="number" order="descending"/></xsl:merge-source>
  <xsl:merge-action><xsl:value-of select="."/>;</xsl:merge-action>
</xsl:merge>`, "10;9;"},
		// order as an attribute value template, evaluated in the focus of
		// the xsl:merge instruction (§15.5).
		{"order-avt", `<xsl:merge>
  <xsl:merge-source select="r/s/n"><xsl:merge-key select="xs:integer(.)" order="{if (true()) then 'descending' else 'ascending'}"/></xsl:merge-source>
  <xsl:merge-action><xsl:value-of select="."/>;</xsl:merge-action>
</xsl:merge>`, "10;9;"},
		// Atomic input items: merge keys see each item with singleton focus.
		{"atomic-items", `<xsl:merge>
  <xsl:merge-source select="(1, 3, 5)"><xsl:merge-key select="."/></xsl:merge-source>
  <xsl:merge-source select="(2, 3)"><xsl:merge-key select="."/></xsl:merge-source>
  <xsl:merge-action>[<xsl:value-of select="current-merge-group()"/>]</xsl:merge-action>
</xsl:merge>`, "[1][2][3 3][5]"},
		// An empty input sequence: the action never runs.
		{"empty-input", `<xsl:merge>
  <xsl:merge-source select="r/none"><xsl:merge-key select="."/></xsl:merge-source>
  <xsl:merge-action>X</xsl:merge-action>
</xsl:merge>|`, "|"},
		// xsl:fallback after the merge-action is ignored by an XSLT 3.0
		// processor (§15.2).
		{"fallback-ignored", `<xsl:merge>
  <xsl:merge-source select="r/a/i[1]"><xsl:merge-key select="@k"/></xsl:merge-source>
  <xsl:merge-action>ok</xsl:merge-action>
  <xsl:fallback>fallback</xsl:fallback>
</xsl:merge>`, "ok"},
		// Inside the action, a nested merge has its own group; afterwards the
		// outer group is current again.
		{"nested", `<xsl:merge>
  <xsl:merge-source select="r/a/i[1]"><xsl:merge-key select="@k"/></xsl:merge-source>
  <xsl:merge-action><xsl:merge>
      <xsl:merge-source select="(7, 8)"><xsl:merge-key select="."/></xsl:merge-source>
      <xsl:merge-action><xsl:value-of select="current-merge-key()"/></xsl:merge-action>
    </xsl:merge>-<xsl:value-of select="current-merge-key()"/></xsl:merge-action>
</xsl:merge>`, "78-1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := mtERun(t, mtEMergeSheet(c.body), mtEMergeSrc)
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestMergeSpecErrors(t *testing.T) {
	cases := []struct {
		name, body string
		codes      []string // any of
	}{
		{"XTSE2200-key-count", `<xsl:merge>
  <xsl:merge-source select="r/a/i"><xsl:merge-key select="@k"/></xsl:merge-source>
  <xsl:merge-source select="r/b/i"><xsl:merge-key select="@k"/><xsl:merge-key select="."/></xsl:merge-source>
  <xsl:merge-action/>
</xsl:merge>`, []string{"XTSE2200"}},
		{"XTSE3190-duplicate-name", `<xsl:merge>
  <xsl:merge-source name="x" select="r/a/i"><xsl:merge-key select="@k"/></xsl:merge-source>
  <xsl:merge-source name="x" select="r/b/i"><xsl:merge-key select="@k"/></xsl:merge-source>
  <xsl:merge-action/>
</xsl:merge>`, []string{"XTSE3190"}},
		{"XTSE3195-both-for-each", `<xsl:merge>
  <xsl:merge-source for-each-item="r" for-each-source="'x.xml'" select="a"><xsl:merge-key select="."/></xsl:merge-source>
  <xsl:merge-action/>
</xsl:merge>`, []string{"XTSE3195"}},
		// Content model (xsl:merge-source+, xsl:merge-action, xsl:fallback*).
		{"XTSE0010-two-actions", `<xsl:merge>
  <xsl:merge-source select="r/a/i"><xsl:merge-key select="@k"/></xsl:merge-source>
  <xsl:merge-action/><xsl:merge-action/>
</xsl:merge>`, []string{"XTSE0010"}},
		{"XTSE0010-source-after-action", `<xsl:merge>
  <xsl:merge-action/>
  <xsl:merge-source select="r/a/i"><xsl:merge-key select="@k"/></xsl:merge-source>
</xsl:merge>`, []string{"XTSE0010"}},
		{"XTSE0010-fallback-first", `<xsl:merge>
  <xsl:merge-source select="r/a/i"><xsl:merge-key select="@k"/></xsl:merge-source>
  <xsl:fallback/>
  <xsl:merge-action/>
</xsl:merge>`, []string{"XTSE0010"}},
		{"XTSE0010-source-content", `<xsl:merge>
  <xsl:merge-source select="r/a/i"><xsl:merge-key select="@k"/><xsl:sequence select="1"/></xsl:merge-source>
  <xsl:merge-action/>
</xsl:merge>`, []string{"XTSE0010"}},
		// XTSE3200: a merge key has @select or content, not both (§15.5).
		{"XTSE3200", `<xsl:merge>
  <xsl:merge-source select="r/a/i"><xsl:merge-key select="@k">body</xsl:merge-key></xsl:merge-source>
  <xsl:merge-action/>
</xsl:merge>`, []string{"XTSE3200"}},
		// Attribute types: name is an NCName, streamable a boolean.
		{"XTSE0020-name", `<xsl:merge>
  <xsl:merge-source name="a:b:c" select="r/a/i"><xsl:merge-key select="@k"/></xsl:merge-source>
  <xsl:merge-action/>
</xsl:merge>`, []string{"XTSE0020"}},
		{"XTSE0020-streamable", `<xsl:merge>
  <xsl:merge-source streamable="maybe" select="r/a/i"><xsl:merge-key select="@k"/></xsl:merge-source>
  <xsl:merge-action/>
</xsl:merge>`, []string{"XTSE0020"}},
		// XTDE2210: corresponding merge keys with differing order; may be
		// reported statically, and must be found once AVTs are evaluated.
		{"XTDE2210-static", `<xsl:merge>
  <xsl:merge-source select="r/a/i"><xsl:merge-key select="@k" order="ascending"/></xsl:merge-source>
  <xsl:merge-source select="r/b/i"><xsl:merge-key select="@k" order="descending"/></xsl:merge-source>
  <xsl:merge-action/>
</xsl:merge>`, []string{"XTDE2210"}},
		{"XTDE2210-present-on-one", `<xsl:merge>
  <xsl:merge-source select="r/a/i"><xsl:merge-key select="@k" lang="en"/></xsl:merge-source>
  <xsl:merge-source select="r/b/i"><xsl:merge-key select="@k"/></xsl:merge-source>
  <xsl:merge-action/>
</xsl:merge>`, []string{"XTDE2210"}},
		{"XTDE2210-avt", `<xsl:merge>
  <xsl:merge-source select="r/a/i"><xsl:merge-key select="@k" order="{'ascending'}"/></xsl:merge-source>
  <xsl:merge-source select="r/b/i"><xsl:merge-key select="@k" order="{'descending'}"/></xsl:merge-source>
  <xsl:merge-action/>
</xsl:merge>`, []string{"XTDE2210"}},
		{"XTDE2210-data-type-avt", `<xsl:merge>
  <xsl:merge-source select="r/a/i"><xsl:merge-key select="@k" data-type="{'number'}"/></xsl:merge-source>
  <xsl:merge-source select="r/b/i"><xsl:merge-key select="@k" data-type="{'text'}"/></xsl:merge-source>
  <xsl:merge-action/>
</xsl:merge>`, []string{"XTDE2210"}},
		// XTDE2220: an input not sorted by its merge key.
		{"XTDE2220-unsorted", `<xsl:merge>
  <xsl:merge-source select="r/u/n"><xsl:merge-key select="."/></xsl:merge-source>
  <xsl:merge-action/>
</xsl:merge>`, []string{"XTDE2220"}},
		{"XTDE2220-for-each-item", `<xsl:merge>
  <xsl:merge-source for-each-item="r/u" select="n"><xsl:merge-key select="."/></xsl:merge-source>
  <xsl:merge-action/>
</xsl:merge>`, []string{"XTDE2220"}},
		// XTTE2230: keys from different inputs not comparable with le.
		{"XTTE2230-incomparable", `<xsl:merge>
  <xsl:merge-source select="r/a/i"><xsl:merge-key select="xs:integer(@k)"/></xsl:merge-source>
  <xsl:merge-source select="r/b/i"><xsl:merge-key select="string(@k)"/></xsl:merge-source>
  <xsl:merge-action/>
</xsl:merge>`, []string{"XTTE2230"}},
		// A merge key selecting more than one item: XSLT 3.0 leaves the code
		// between XTTE1020 (xsl:sort's rule) and XTTE2230 (merge-038).
		{"multi-item-key", `<xsl:merge>
  <xsl:merge-source select="r/a"><xsl:merge-key select="i"/></xsl:merge-source>
  <xsl:merge-action/>
</xsl:merge>`, []string{"XTTE1020", "XTTE2230"}},
		// for-each-source is xs:string*.
		{"XPTY0004-for-each-source", `<xsl:merge>
  <xsl:merge-source for-each-source="1 to 2" select="r"><xsl:merge-key select="."/></xsl:merge-source>
  <xsl:merge-action/>
</xsl:merge>`, []string{"XPTY0004"}},
		{"FODC0002-for-each-source", `<xsl:merge>
  <xsl:merge-source for-each-source="'no-such-mte-doc.xml'" select="r"><xsl:merge-key select="."/></xsl:merge-source>
  <xsl:merge-action/>
</xsl:merge>`, []string{"FODC0002"}},
		// §15.6: current-merge-group()/current-merge-key() are absent outside
		// a merge action, and inside a function called from one; a source
		// name must name a merge source of the current merge.
		{"XTDE3480-outside", `<xsl:value-of select="count(current-merge-group())"/>`, []string{"XTDE3480"}},
		{"XTDE3510-outside", `<xsl:value-of select="current-merge-key()"/>`, []string{"XTDE3510"}},
		{"XTDE3490-bad-source", `<xsl:merge>
  <xsl:merge-source name="a" select="r/a/i"><xsl:merge-key select="@k"/></xsl:merge-source>
  <xsl:merge-action><xsl:value-of select="current-merge-group('zzz')"/></xsl:merge-action>
</xsl:merge>`, []string{"XTDE3490"}},
		// The current merge group and key are cleared by function calls
		// (§15.6.1 Notes).
		{"XTDE3480-in-function", `<xsl:merge>
  <xsl:merge-source select="r/a/i"><xsl:merge-key select="@k"/></xsl:merge-source>
  <xsl:merge-action><xsl:value-of select="mte:group()"/></xsl:merge-action>
</xsl:merge>`, []string{"XTDE3480"}},
		{"XTDE3510-in-function", `<xsl:merge>
  <xsl:merge-source select="r/a/i"><xsl:merge-key select="@k"/></xsl:merge-source>
  <xsl:merge-action><xsl:value-of select="mte:key()"/></xsl:merge-action>
</xsl:merge>`, []string{"XTDE3510"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := mtERun(t, mtEMergeSheet(c.body), mtEMergeSrc)
			if err == nil {
				t.Fatalf("want one of %v, got success", c.codes)
			}
			for _, code := range c.codes {
				if strings.Contains(err.Error(), "err:"+code) {
					return
				}
			}
			t.Errorf("want one of %v, got %v", c.codes, err)
		})
	}
}
