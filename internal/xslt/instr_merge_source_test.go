package xslt

import (
	"path/filepath"
	"testing"
)

// TestMergeForEachSource reads one input sequence per document named by
// @for-each-source (XSLT 3.0 §15.3), relative URIs resolving against the
// xsl:merge-source element's base URI.
func TestMergeForEachSource(t *testing.T) {
	dir := t.TempDir()
	mtEWrite(t, dir, "a.xml", `<log><e t="1">a1</e><e t="4">a4</e></log>`)
	mtEWrite(t, dir, "b.xml", `<log><e t="2">b2</e><e t="4">b4</e></log>`)
	sheet := mtEMergeSheet(`<xsl:merge>
  <xsl:merge-source name="logs" for-each-source="('a.xml', 'b.xml')" select="log/e"><xsl:merge-key select="xs:integer(@t)"/></xsl:merge-source>
  <xsl:merge-action>[<xsl:value-of select="current-merge-key()"/>:<xsl:value-of select="current-merge-group('logs')"/>]</xsl:merge-action>
</xsl:merge>`)
	ss, err := CompileAt(sheet, filepath.Join(dir, "m.xsl"))
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	rr, err := ss.TransformFull("<x/>", nil, dir)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	// Within a group, items from different input sequences of one source
	// follow the order of their anchors (§15.6.1).
	if want := "[1:a1][2:b2][4:a4 b4]"; rr.Output != want {
		t.Errorf("got %q, want %q", rr.Output, want)
	}
}

// TestMergeCollation compares merge keys under the collation named by
// @collation: with the UCA at primary strength "a" and "A" are equal keys
// (F&O 3.1 §5.3.4), so they form one group.
func TestMergeCollation(t *testing.T) {
	sheet := mtEMergeSheet(`<xsl:merge>
  <xsl:merge-source select="('a', 'A', 'b')"><xsl:merge-key select="." collation="http://www.w3.org/2013/collation/UCA?strength=primary"/></xsl:merge-source>
  <xsl:merge-source select="('B')"><xsl:merge-key select="." collation="http://www.w3.org/2013/collation/UCA?strength=primary"/></xsl:merge-source>
  <xsl:merge-action>[<xsl:value-of select="current-merge-group()"/>]</xsl:merge-action>
</xsl:merge>`)
	got, err := mtERun(t, sheet, "<x/>")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if want := "[a A][b B]"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestMergeStringKeysCompareAsStrings: with no @data-type, merge keys are
// compared "using the rules appropriate to their datatype", xs:untypedAtomic
// cast to xs:string (XSLT 3.0 §13.1.2, applied to merge keys by §15.5), so
// the untyped @k values "10" and "9" are strings and "10" precedes "9".
func TestMergeStringKeysCompareAsStrings(t *testing.T) {
	for _, key := range []string{"@k", "xs:string(@k)"} {
		sheet := mtEMergeSheet(`<xsl:merge>
  <xsl:merge-source select="r/i"><xsl:merge-key select="` + key + `"/></xsl:merge-source>
  <xsl:merge-action>[<xsl:value-of select="current-merge-key()"/>]</xsl:merge-action>
</xsl:merge>`)
		got, err := mtERun(t, sheet, `<r><i k="10"/><i k="9"/></r>`)
		if err != nil {
			t.Fatalf("%s: run: %v", key, err)
		}
		if want := "[10][9]"; got != want {
			t.Errorf("%s: got %q, want %q", key, got, want)
		}
	}
	// Numeric keys still compare as numbers.
	sheet := mtEMergeSheet(`<xsl:merge>
  <xsl:merge-source select="r/i"><xsl:merge-key select="xs:integer(@k)"/></xsl:merge-source>
  <xsl:merge-action>[<xsl:value-of select="current-merge-key()"/>]</xsl:merge-action>
</xsl:merge>`)
	if got, err := mtERun(t, sheet, `<r><i k="9"/><i k="10"/></r>`); err != nil || got != "[9][10]" {
		t.Errorf("integer keys: got %q, %v; want [9][10]", got, err)
	}
}

// TestMergeCurrentKeyIsAtomized: current-merge-key() returns "the merge key
// of the first item in the group, after atomization and casting of
// xs:untypedAtomic values to xs:string" (XSLT 3.0 §15.6.2).
func TestMergeCurrentKeyIsAtomized(t *testing.T) {
	sheet := mtEMergeSheet(`<xsl:merge>
  <xsl:merge-source select="r/i"><xsl:merge-key select="@k"/></xsl:merge-source>
  <xsl:merge-action>[<xsl:value-of select="current-merge-key() instance of xs:string, current-merge-key() instance of node()"/>]</xsl:merge-action>
</xsl:merge>`)
	got, err := mtERun(t, sheet, `<r><i k="a"/></r>`)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if want := "[true false]"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	// An atomic merge item keeps its own type as the key.
	sheet = mtEMergeSheet(`<xsl:merge>
  <xsl:merge-source select="(1, 2)"><xsl:merge-key select="."/></xsl:merge-source>
  <xsl:merge-action>[<xsl:value-of select="current-merge-key() instance of xs:integer"/>]</xsl:merge-action>
</xsl:merge>`)
	if got, err := mtERun(t, sheet, `<x/>`); err != nil || got != "[true][true]" {
		t.Errorf("atomic items: got %q, %v; want [true][true]", got, err)
	}
}
