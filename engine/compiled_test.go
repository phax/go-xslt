package engine_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/tim-riep/go-xslt/engine"
)

// A stylesheet that touches most of the state a run builds: keys, an
// accumulator, a memoized and a plain function, grouping, sorting, regex,
// number formatting, maps, try/catch, xsl:number, document() of a file and
// xsl:result-document.
const compiledSheet = `<xsl:stylesheet version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:f="urn:f" xmlns:map="http://www.w3.org/2005/xpath-functions/map"
  exclude-result-prefixes="xs f map">
  <xsl:param name="label" select="'L'"/>
  <xsl:key name="byCat" match="item" use="@cat"/>
  <xsl:accumulator name="sum" as="xs:integer" initial-value="0">
    <xsl:accumulator-rule match="item" select="$value + xs:integer(@n)"/>
  </xsl:accumulator>
  <xsl:mode use-accumulators="sum"/>
  <xsl:decimal-format name="eu" decimal-separator="," grouping-separator="."/>
  <xsl:function name="f:fib" as="xs:integer" new-each-time="no">
    <xsl:param name="n" as="xs:integer"/>
    <xsl:sequence select="if ($n lt 2) then $n else f:fib($n - 1) + f:fib($n - 2)"/>
  </xsl:function>
  <xsl:function name="f:up" as="xs:string"><xsl:param name="s" as="xs:string"/><xsl:sequence select="upper-case($s)"/></xsl:function>
  <xsl:output method="xml" indent="no"/>
  <xsl:template match="/">
    <r label="{$label}">
      <groups>
        <xsl:for-each-group select="data/item" group-by="@cat">
          <xsl:sort select="current-grouping-key()"/>
          <g k="{f:up(current-grouping-key())}" n="{count(current-group())}" keyed="{count(key('byCat', current-grouping-key()))}"/>
        </xsl:for-each-group>
      </groups>
      <fib><xsl:value-of select="f:fib(20)"/></fib>
      <fmt><xsl:value-of select="format-number(1234567.891, '#.##0,00', 'eu')"/></fmt>
      <rx><xsl:analyze-string select="string-join(data/item/@n, ',')" regex="(\d)(\d)?">
        <xsl:matching-substring><m><xsl:value-of select="regex-group(1)"/></m></xsl:matching-substring>
      </xsl:analyze-string></rx>
      <acc><xsl:value-of select="data/item[last()]/accumulator-after('sum')"/></acc>
      <map><xsl:value-of select="let $m := map{ 'a': count(data/item), 'b': 2 } return $m?a + map:get($m, 'b')"/></map>
      <try><xsl:try><xsl:value-of select="1 div 0"/><xsl:catch><caught><xsl:value-of select="$err:code" xmlns:err="http://www.w3.org/2005/xqt-errors"/></caught></xsl:catch></xsl:try></try>
      <doc><xsl:value-of select="document('side.xml')/side/@v"/></doc>
      <xsl:for-each select="data/item"><xsl:sort select="xs:integer(@n)" order="descending"/><i><xsl:number/>:<xsl:value-of select="@n"/></i></xsl:for-each>
      <xsl:result-document href="out-{$label}.xml"><side><xsl:value-of select="$label"/></side></xsl:result-document>
    </r>
  </xsl:template>
</xsl:stylesheet>`

func source(i int) string {
	var b strings.Builder
	b.WriteString("<data>")
	for k := 0; k < 5+i%7; k++ {
		fmt.Fprintf(&b, `<item cat="c%d" n="%d"/>`, (k+i)%3, k*3+i)
	}
	b.WriteString("</data>")
	return b.String()
}

// TestCompiledStylesheetIsConcurrencySafe runs one compiled stylesheet from many
// goroutines at once and requires every result to equal what a fresh
// Transform of the same inputs gives. Run it with -race.
func TestCompiledStylesheetIsConcurrencySafe(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "side.xml"), []byte(`<side v="42"/>`), 0o644); err != nil {
		t.Fatal(err)
	}
	sheet, diags := engine.Compile(engine.Request{Stylesheet: compiledSheet, BaseDir: dir})
	if sheet == nil {
		t.Fatalf("compile: %v", diags)
	}
	const inputs = 12
	want := make([]string, inputs)
	for i := range want {
		r := engine.Transform(engine.Request{Stylesheet: compiledSheet, BaseDir: dir, Source: source(i), Params: map[string]string{"label": fmt.Sprint("L", i)}})
		if r.HasErrors() {
			t.Fatalf("reference run %d: %v", i, r.Diagnostics)
		}
		want[i] = r.Output + "|" + fmt.Sprint(len(r.SecondaryOutputs))
	}
	var wg sync.WaitGroup
	errs := make(chan string, 1000)
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for n := 0; n < 15; n++ {
				i := (g + n) % inputs
				r := sheet.Transform(engine.TransformRequest{Source: source(i), Params: map[string]string{"label": fmt.Sprint("L", i)}})
				got := r.Output + "|" + fmt.Sprint(len(r.SecondaryOutputs))
				if r.HasErrors() || got != want[i] {
					errs <- fmt.Sprintf("goroutine %d input %d: %v\n got  %s\n want %s", g, i, r.Diagnostics, got, want[i])
					return
				}
			}
		}(g)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
}

func TestCompileReportsErrorsWithPosition(t *testing.T) {
	sheet, diags := engine.Compile(engine.Request{Stylesheet: "<xsl:stylesheet version=\"3.0\" xmlns:xsl=\"http://www.w3.org/1999/XSL/Transform\">\n  <xsl:template match=\"/\">\n    <xsl:value-of select=\"1 +\"/>\n  </xsl:template>\n</xsl:stylesheet>"})
	if sheet != nil || len(diags) == 0 || diags[0].Line == 0 {
		t.Fatalf("want a compile diagnostic with a line, got sheet=%v diags=%+v", sheet, diags)
	}
}

func TestCompiledTransformToAndReader(t *testing.T) {
	sheet, diags := engine.Compile(engine.Request{Stylesheet: `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"><xsl:output method="text"/><xsl:template match="/"><xsl:value-of select="count(//i)"/></xsl:template></xsl:stylesheet>`})
	if sheet == nil {
		t.Fatal(diags)
	}
	var sb strings.Builder
	if r := sheet.TransformTo(&sb, engine.TransformRequest{Source: "<r><i/><i/></r>"}); r.HasErrors() || sb.String() != "2" {
		t.Fatalf("TransformTo: %q %v", sb.String(), r.Diagnostics)
	}
	if r := sheet.TransformFromReader(strings.NewReader("<r><i/><i/><i/></r>"), engine.TransformRequest{}); r.HasErrors() || r.Output != "3" {
		t.Fatalf("TransformFromReader: %q %v", r.Output, r.Diagnostics)
	}
}

// TestRuntimeErrorsCarryTheStylesheetPositionAndCode: a host that generates the
// stylesheet from a template must be able to map a failure back to the element
// that raised it (line/column) and to branch on the W3C error code.
func TestRuntimeErrorsCarryTheStylesheetPositionAndCode(t *testing.T) {
	ss := "<xsl:stylesheet version=\"3.0\" xmlns:xsl=\"http://www.w3.org/1999/XSL/Transform\" xmlns:xs=\"http://www.w3.org/2001/XMLSchema\">\n" + // line 1
		"  <xsl:template match=\"/\">\n" + // 2
		"    <out>\n" + // 3
		"      <xsl:value-of select=\"xs:integer('abc')\"/>\n" + // 4
		"    </out>\n" +
		"  </xsl:template>\n" +
		"  <xsl:template name=\"t\">\n" + // 7
		"    <xsl:message terminate=\"yes\">boom</xsl:message>\n" + // 8
		"  </xsl:template>\n" +
		"</xsl:stylesheet>"
	sheet, diags := engine.Compile(engine.Request{Stylesheet: ss})
	if sheet == nil {
		t.Fatal(diags)
	}
	r := sheet.Transform(engine.TransformRequest{Source: "<x/>"})
	if len(r.Diagnostics) != 1 {
		t.Fatalf("diagnostics: %+v", r.Diagnostics)
	}
	if d := r.Diagnostics[0]; d.Line != 4 || d.Code != "FORG0001" || d.Phase != "run" {
		t.Errorf("value-of error: line %d code %q phase %q (%s), want line 4 FORG0001 run", d.Line, d.Code, d.Phase, d.Message)
	}
	r = sheet.Transform(engine.TransformRequest{InitialTemplate: "t"})
	if len(r.Diagnostics) != 1 || r.Diagnostics[0].Line != 8 || r.Diagnostics[0].Code != "XTMM9000" {
		t.Errorf("xsl:message error: %+v, want line 8 XTMM9000", r.Diagnostics)
	}
}
