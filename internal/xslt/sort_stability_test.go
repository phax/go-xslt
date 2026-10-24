package xslt

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// TestSortIsStableOnTotalKeys checks the unstable-sort path of sortNodes
// (sortKeysTotal) against an independent stable sort: thousands of records
// with heavily duplicated text and number keys (non-numeric values included,
// which sort first), under every combination of order and case-order the path
// takes, must come out exactly as sort.SliceStable orders them by the same
// keys — equal keys in document order.
func TestSortIsStableOnTotalKeys(t *testing.T) {
	type rec struct {
		id   int
		name string
		num  string
	}
	rng := rand.New(rand.NewSource(7))
	names := []string{"a", "B", "b", "A", "ab", "aB", "", "z"}
	nums := []string{"1", "2", "2.0", "-0", "0", "x", "10", "NaN", "3"}
	recs := make([]rec, 3000)
	var src strings.Builder
	src.WriteString("<r>")
	for i := range recs {
		recs[i] = rec{i, names[rng.Intn(len(names))], nums[rng.Intn(len(nums))]}
		fmt.Fprintf(&src, `<e i="%d" n="%s" v="%s"/>`, i, recs[i].name, recs[i].num)
	}
	src.WriteString("</r>")
	num := func(s string) float64 {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil || s == "NaN" {
			return math.NaN()
		}
		return f
	}
	cmpNum := func(a, b float64) int {
		switch an, bn := math.IsNaN(a), math.IsNaN(b); {
		case an && bn:
			return 0
		case an:
			return -1
		case bn:
			return 1
		case a < b:
			return -1
		case a > b:
			return 1
		}
		return 0
	}
	cmpText := func(a, b, caseOrder string) int {
		if caseOrder == "" {
			return strings.Compare(a, b)
		}
		if c := strings.Compare(strings.ToLower(a), strings.ToLower(b)); c != 0 {
			return c
		}
		c := strings.Compare(a, b)
		if caseOrder == "lower-first" {
			return -c
		}
		return c
	}
	for _, textOrder := range []string{"ascending", "descending"} {
		for _, numOrder := range []string{"ascending", "descending"} {
			for _, caseOrder := range []string{"", "upper-first", "lower-first"} {
				name := textOrder + "/" + numOrder + "/" + caseOrder
				t.Run(name, func(t *testing.T) {
					co := ""
					if caseOrder != "" {
						co = ` case-order="` + caseOrder + `"`
					}
					sheet := `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="text"/>
  <xsl:template match="/"><xsl:for-each select="r/e"><xsl:sort select="@n" order="` + textOrder + `"` + co + `/><xsl:sort select="@v" data-type="number" order="` + numOrder + `"/><xsl:value-of select="@i"/>,</xsl:for-each></xsl:template>
</xsl:stylesheet>`
					ss, err := CompileFrom(sheet, "")
					if err != nil {
						t.Fatal(err)
					}
					res, err := ss.TransformFull(src.String(), nil, "")
					if err != nil {
						t.Fatal(err)
					}
					want := append([]rec(nil), recs...)
					sort.SliceStable(want, func(i, j int) bool {
						a, b := want[i], want[j]
						if c := cmpText(a.name, b.name, caseOrder); c != 0 {
							if textOrder == "descending" {
								c = -c
							}
							return c < 0
						}
						c := cmpNum(num(a.num), num(b.num))
						if numOrder == "descending" {
							c = -c
						}
						return c < 0
					})
					var exp strings.Builder
					for _, r := range want {
						fmt.Fprintf(&exp, "%d,", r.id)
					}
					if res.Output != exp.String() {
						t.Errorf("sorted order differs from the stable reference\n got %.200s\nwant %.200s", res.Output, exp.String())
					}
				})
			}
		}
	}
}
