package xslt

import (
	"fmt"
	"strings"
	"testing"
)

// BenchmarkTransformFunctionCalls runs a transformation whose templates are
// mostly calls of built-in functions (string, numeric and sequence functions
// over attributes and text), the load the argument checking of
// internal/xpath/fn_argcheck.go sits on.
func BenchmarkTransformFunctionCalls(b *testing.B) {
	var sb strings.Builder
	sb.WriteString("<rows>")
	for i := 0; i < 2000; i++ {
		fmt.Fprintf(&sb, `<row id="%d" price="%d.%02d" name="item number %d" tags="a b c"><note>Some text %d</note></row>`,
			i, i%97, i%100, i, i)
	}
	sb.WriteString("</rows>")
	src := sb.String()
	const sheet = `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="text"/>
  <xsl:template match="/rows"><xsl:apply-templates select="row"/></xsl:template>
  <xsl:template match="row">
    <xsl:value-of select="concat(upper-case(@name), '|', string-length(note), '|', substring(@name, 6, 4), '|',
      format-number(@price, '0.00'), '|', round(@price), '|', abs(@id - 1000), '|',
      contains(note, 'text'), '|', starts-with(@name, 'item'), '|', normalize-space(note), '|',
      string-join(tokenize(@tags, ' '), ','), '|', count(tokenize(@tags, ' ')), '|',
      translate(@name, 'aeiou', 'AEIOU'), '|', substring-before(@name, ' '), '|',
      string(number(@price) * 2), '&#10;')"/>
  </xsl:template>
</xsl:stylesheet>`
	ss, err := Compile(sheet)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := ss.Transform(src, nil); err != nil {
			b.Fatal(err)
		}
	}
}
