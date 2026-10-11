package xslt

import (
	"strings"
	"testing"
)

// TestStripSpaceParseTimeSanity targets exactly the hazards the design
// discussion flagged: an xml:space="preserve" override, and a CDATA section
// splitting a would-be-whitespace-only run so its true content is only known
// once the run closes.
func TestStripSpaceParseTimeSanity(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			"plain whitespace stripped",
			`<root><a>  </a><a>x</a></root>`,
			"x",
		},
		{
			"xml:space=preserve overrides strip-space",
			`<root><a xml:space="preserve">  </a><a>x</a></root>`,
			"  x",
		},
		{
			"xml:space=preserve inherited by descendant",
			`<root xml:space="preserve"><a>  </a><a>x</a></root>`,
			"  x",
		},
		{
			"xml:space=default resets inherited preserve",
			`<root xml:space="preserve"><a xml:space="default">  </a><a>x</a></root>`,
			"x",
		},
		{
			"CDATA splits a whitespace-looking run into non-whitespace",
			"<root><a>  <![CDATA[y]]>  </a><a>x</a></root>",
			"  y  x",
		},
		{
			"CDATA-only empty run inside a strippable element",
			"<root><a><![CDATA[]]></a><a>x</a></root>",
			"x",
		},
	}
	sheet := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:strip-space elements="a root"/>
  <xsl:output method="text"/>
  <xsl:template match="/"><xsl:value-of select="string-join(//text(), '')"/></xsl:template>
</xsl:stylesheet>`
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ss, err := CompileFrom(sheet, "")
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			res, err := ss.TransformEntry(tc.src, Entry{}, "")
			if err != nil {
				t.Fatalf("transform: %v", err)
			}
			got := strings.TrimRight(res.Output, "\n")
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
