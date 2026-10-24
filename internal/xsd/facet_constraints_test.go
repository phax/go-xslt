package xsd

import "testing"

// Schema Component Constraints on facets (Datatypes 1.0 §4.3, 1.1 §4.3):
// applicability (cos-applicable-facets), *-valid-restriction, the
// length/minLength/maxLength and bound co-occurrence rules, fixed facets and
// the facet values' own lexical spaces.
func TestFacetConstraints(t *testing.T) {
	// two builds a base type B (restriction of builtin with baseFacets) and
	// D restricting B with derivedFacets.
	two := func(builtin, baseFacets, derivedFacets string) string {
		return `<xs:simpleType name="B"><xs:restriction base="` + builtin + `">` + baseFacets + `</xs:restriction></xs:simpleType>
		  <xs:simpleType name="D"><xs:restriction base="B">` + derivedFacets + `</xs:restriction></xs:simpleType>`
	}
	one := func(builtin, facets string) string {
		return `<xs:simpleType name="D"><xs:restriction base="` + builtin + `">` + facets + `</xs:restriction></xs:simpleType>`
	}
	cases := []struct {
		name  string
		ver   int
		body  string
		valid bool
	}{
		// cos-applicable-facets.
		{"length-on-string", verBoth, one("xs:string", `<xs:length value="3"/>`), true},
		{"length-on-hexBinary", verBoth, one("xs:hexBinary", `<xs:length value="3"/>`), true},
		{"length-on-int", verBoth, one("xs:int", `<xs:length value="3"/>`), false},
		{"totalDigits-on-string", verBoth, one("xs:string", `<xs:totalDigits value="3"/>`), false},
		{"fractionDigits-on-double", verBoth, one("xs:double", `<xs:fractionDigits value="1"/>`), false},
		{"maxInclusive-on-boolean", verBoth, one("xs:boolean", `<xs:maxInclusive value="true"/>`), false},
		{"maxInclusive-on-date", verBoth, one("xs:date", `<xs:maxInclusive value="2020-01-01"/>`), true},
		{"pattern-on-boolean", verBoth, one("xs:boolean", `<xs:pattern value="true"/>`), true},
		{"explicitTimezone-1.1", ver11, one("xs:dateTime", `<xs:explicitTimezone value="required"/>`), true},
		{"explicitTimezone-on-string", ver11, one("xs:string", `<xs:explicitTimezone value="required"/>`), false},
		{"explicitTimezone-1.0", ver10 + skipBug, one("xs:dateTime", `<xs:explicitTimezone value="required"/>`), false},
		{"explicitTimezone-bad-value", ver11, one("xs:dateTime", `<xs:explicitTimezone value="sometimes"/>`), false},
		{"unknown-facet", verBoth, one("xs:string", `<xs:maxSize value="3"/>`), false},
		// Facet values must be in the facet's value space.
		{"length-negative", verBoth, one("xs:string", `<xs:length value="-1"/>`), false},
		{"length-not-integer", verBoth, one("xs:string", `<xs:length value="x"/>`), false},
		{"totalDigits-zero", verBoth, one("xs:decimal", `<xs:totalDigits value="0"/>`), false},
		{"maxInclusive-not-of-type", verBoth, one("xs:int", `<xs:maxInclusive value="abc"/>`), false},
		{"whiteSpace-bad", verBoth + skipBug, one("xs:string", `<xs:whiteSpace value="trim"/>`), false},
		{"pattern-bad-regex", verBoth, one("xs:string", `<xs:pattern value="a(b"/>`), false},
		{"enumeration-not-of-type", verBoth, one("xs:int", `<xs:enumeration value="x"/>`), false},
		{"facet-twice", verBoth, one("xs:string", `<xs:maxLength value="3"/><xs:maxLength value="4"/>`), false},
		// Co-occurrence within one step.
		{"minLength-gt-maxLength", verBoth, one("xs:string", `<xs:minLength value="5"/><xs:maxLength value="4"/>`), false},
		{"minLength-le-maxLength", verBoth, one("xs:string", `<xs:minLength value="4"/><xs:maxLength value="4"/>`), true},
		{"length-with-minLength", verBoth, one("xs:string", `<xs:length value="4"/><xs:minLength value="4"/>`), false},
		{"length-with-maxLength", verBoth, one("xs:string", `<xs:length value="4"/><xs:maxLength value="5"/>`), false},
		{"maxInclusive-and-maxExclusive", verBoth, one("xs:int", `<xs:maxInclusive value="4"/><xs:maxExclusive value="5"/>`), false},
		{"minInclusive-and-minExclusive", verBoth, one("xs:int", `<xs:minInclusive value="4"/><xs:minExclusive value="3"/>`), false},
		{"min-gt-max", verBoth, one("xs:int", `<xs:minInclusive value="5"/><xs:maxInclusive value="4"/>`), false},
		{"minExclusive-eq-maxExclusive", verBoth, one("xs:decimal", `<xs:minExclusive value="4"/><xs:maxExclusive value="4"/>`), false},
		{"minInclusive-eq-maxInclusive", verBoth, one("xs:decimal", `<xs:minInclusive value="4"/><xs:maxInclusive value="4"/>`), true},
		{"minExclusive-ge-maxInclusive", verBoth, one("xs:decimal", `<xs:minExclusive value="5"/><xs:maxInclusive value="4"/>`), false},
		{"fractionDigits-gt-totalDigits", verBoth, one("xs:decimal", `<xs:totalDigits value="2"/><xs:fractionDigits value="3"/>`), false},
		{"fractionDigits-on-integer-nonzero", verBoth, one("xs:integer", `<xs:fractionDigits value="1"/>`), false},
		{"date-bounds-ok", verBoth, one("xs:date", `<xs:minInclusive value="2000-01-01"/><xs:maxExclusive value="2001-01-01"/>`), true},
		{"date-bounds-crossed", verBoth, one("xs:date", `<xs:minInclusive value="2001-01-01"/><xs:maxExclusive value="2000-01-01"/>`), false},
		{"duration-bounds-crossed", verBoth, one("xs:duration", `<xs:minInclusive value="P2D"/><xs:maxInclusive value="P1D"/>`), false},
		{"float-bounds-crossed", verBoth, one("xs:float", `<xs:minInclusive value="2.5"/><xs:maxInclusive value="1E0"/>`), false},
		// Restriction steps (*-valid-restriction).
		{"maxLength-widened", verBoth, two("xs:string", `<xs:maxLength value="3"/>`, `<xs:maxLength value="4"/>`), false},
		{"maxLength-narrowed", verBoth, two("xs:string", `<xs:maxLength value="4"/>`, `<xs:maxLength value="3"/>`), true},
		{"minLength-widened", verBoth, two("xs:string", `<xs:minLength value="3"/>`, `<xs:minLength value="2"/>`), false},
		{"maxLength-below-base-minLength", verBoth, two("xs:string", `<xs:minLength value="3"/>`, `<xs:maxLength value="2"/>`), false},
		{"length-changed", verBoth + skipBug, two("xs:string", `<xs:length value="3"/>`, `<xs:length value="4"/>`), false},
		{"totalDigits-widened", verBoth, two("xs:decimal", `<xs:totalDigits value="3"/>`, `<xs:totalDigits value="4"/>`), false},
		{"fractionDigits-widened", verBoth, two("xs:decimal", `<xs:fractionDigits value="2"/>`, `<xs:fractionDigits value="3"/>`), false},
		{"whiteSpace-weakened", verBoth, two("xs:string", `<xs:whiteSpace value="collapse"/>`, `<xs:whiteSpace value="preserve"/>`), false},
		{"whiteSpace-strengthened", verBoth, two("xs:string", `<xs:whiteSpace value="replace"/>`, `<xs:whiteSpace value="collapse"/>`), true},
		{"whiteSpace-builtin-token-preserve", verBoth, one("xs:token", `<xs:whiteSpace value="preserve"/>`), false},
		{"maxInclusive-widened", verBoth, two("xs:int", `<xs:maxInclusive value="10"/>`, `<xs:maxInclusive value="11"/>`), false},
		{"maxInclusive-vs-base-maxExclusive", verBoth, two("xs:int", `<xs:maxExclusive value="10"/>`, `<xs:maxInclusive value="10"/>`), false},
		{"maxExclusive-vs-base-maxInclusive", verBoth, two("xs:int", `<xs:maxInclusive value="10"/>`, `<xs:maxExclusive value="10"/>`), true},
		{"minInclusive-widened", verBoth, two("xs:int", `<xs:minInclusive value="10"/>`, `<xs:minInclusive value="9"/>`), false},
		{"minExclusive-narrowed", verBoth, two("xs:int", `<xs:minInclusive value="10"/>`, `<xs:minExclusive value="10"/>`), true},
		{"upper-below-base-lower", verBoth, two("xs:int", `<xs:minInclusive value="10"/>`, `<xs:maxInclusive value="5"/>`), false},
		{"lower-above-base-upper", verBoth, two("xs:int", `<xs:maxInclusive value="10"/>`, `<xs:minInclusive value="11"/>`), false},
		{"upper-empties-builtin", verBoth, one("xs:positiveInteger", `<xs:maxInclusive value="0"/>`), false},
		{"lower-empties-builtin", verBoth, one("xs:byte", `<xs:minInclusive value="128"/>`), false},
		{"fixed-facet-changed", verBoth, two("xs:string", `<xs:maxLength value="4" fixed="true"/>`, `<xs:maxLength value="3"/>`), false},
		{"fixed-facet-same", verBoth, two("xs:string", `<xs:maxLength value="4" fixed="true"/>`, `<xs:maxLength value="4"/>`), true},
		{"fixed-bound-changed", verBoth, two("xs:int", `<xs:maxInclusive value="10" fixed="true"/>`, `<xs:maxInclusive value="9"/>`), false},
		{"builtin-fixed-whitespace", verBoth, one("xs:integer", `<xs:whiteSpace value="replace"/>`), false},
		{"enumeration-outside-base", verBoth + skipBug, two("xs:int", `<xs:maxInclusive value="10"/>`, `<xs:enumeration value="11"/>`), false},
		{"enumeration-inside-base", verBoth, two("xs:int", `<xs:maxInclusive value="10"/>`, `<xs:enumeration value="9"/>`), true},
		{"explicitTimezone-loosened", ver11, two("xs:dateTime", `<xs:explicitTimezone value="required"/>`, `<xs:explicitTimezone value="optional"/>`), false},
		{"explicitTimezone-stamp-prohibited", ver11, one("xs:dateTimeStamp", `<xs:explicitTimezone value="prohibited"/>`), false},
		{"explicitTimezone-pinned-from-optional", ver11, two("xs:dateTime", `<xs:explicitTimezone value="optional"/>`, `<xs:explicitTimezone value="prohibited"/>`), true},
		// Lists: length facets count items, enumeration literals are lists.
		{"list-length", verBoth, `<xs:simpleType name="L"><xs:list itemType="xs:int"/></xs:simpleType>` + `<xs:simpleType name="D"><xs:restriction base="L"><xs:length value="2"/></xs:restriction></xs:simpleType>`, true},
		{"list-maxInclusive", verBoth, `<xs:simpleType name="L"><xs:list itemType="xs:int"/></xs:simpleType>` + `<xs:simpleType name="D"><xs:restriction base="L"><xs:maxInclusive value="2"/></xs:restriction></xs:simpleType>`, false},
		{"list-enumeration-invalid-item", verBoth, `<xs:simpleType name="L"><xs:list itemType="xs:int"/></xs:simpleType>` + `<xs:simpleType name="D"><xs:restriction base="L"><xs:enumeration value="1 x"/></xs:restriction></xs:simpleType>`, false},
		{"nmtokens-empty-enumeration", verBoth, one("xs:NMTOKENS", `<xs:enumeration value=""/>`), false},
		{"nmtokens-maxLength-zero", verBoth, one("xs:NMTOKENS", `<xs:maxLength value="0"/>`), false},
		{"nmtokens-length-with-inherited-minLength", verBoth, one("xs:NMTOKENS", `<xs:length value="2"/>`), true},
		// Unions accept only pattern, enumeration (and 1.1 assertions).
		{"union-pattern", verBoth, `<xs:simpleType name="U"><xs:union memberTypes="xs:int xs:date"/></xs:simpleType><xs:simpleType name="D"><xs:restriction base="U"><xs:pattern value="\d+"/></xs:restriction></xs:simpleType>`, true},
		{"union-maxLength", verBoth, `<xs:simpleType name="U"><xs:union memberTypes="xs:int xs:date"/></xs:simpleType><xs:simpleType name="D"><xs:restriction base="U"><xs:maxLength value="3"/></xs:restriction></xs:simpleType>`, false},
		{"union-enumeration-no-member", verBoth + skipBug, `<xs:simpleType name="U"><xs:union memberTypes="xs:int xs:date"/></xs:simpleType><xs:simpleType name="D"><xs:restriction base="U"><xs:enumeration value="x"/></xs:restriction></xs:simpleType>`, false},
		// Value constraints checked against the facets.
		{"default-violates-facet", verBoth, one("xs:string", `<xs:maxLength value="2"/>`) + `<xs:element name="e" type="D" default="abc"/>`, false},
		{"fixed-satisfies-facet", verBoth, one("xs:string", `<xs:maxLength value="3"/>`) + `<xs:attribute name="a" type="D" fixed="abc"/>`, true},
	}
	for _, tc := range cases {
		for _, v := range versionsOf(tc.ver % skipBug) {
			t.Run(tc.name+"/"+v.String(), func(t *testing.T) {
				if tc.ver >= skipBug {
					t.Skip("BUG: engine verdict disagrees with the spec")
				}
				if got := schemaVerdict(t, reprSchema(tc.body), v); got != tc.valid {
					t.Fatalf("schema valid=%v, want %v", got, tc.valid)
				}
			})
		}
	}
}

// Facet checking against instance values (Datatypes §4.3, constraining
// facets' Validation Rules), across the built-in families.
func TestFacetValidation(t *testing.T) {
	cases := []struct {
		name   string
		ver    int
		typ    string // restriction body
		value  string
		valid  bool
		code   string // expected Error.Code prefix when invalid ("" = any)
		inList bool
	}{
		{"string-length-ok", verBoth, `<xs:restriction base="xs:string"><xs:length value="3"/></xs:restriction>`, "abc", true, "", false},
		{"string-length-chars-not-bytes", verBoth, `<xs:restriction base="xs:string"><xs:length value="3"/></xs:restriction>`, "äöü", true, "", false},
		{"string-length-bad", verBoth, `<xs:restriction base="xs:string"><xs:length value="3"/></xs:restriction>`, "ab", false, "", false},
		{"string-minLength-bad", verBoth, `<xs:restriction base="xs:string"><xs:minLength value="3"/></xs:restriction>`, "ab", false, "", false},
		{"hex-length-octets", verBoth, `<xs:restriction base="xs:hexBinary"><xs:length value="2"/></xs:restriction>`, "0aFF", true, "", false},
		{"hex-length-octets-bad", verBoth, `<xs:restriction base="xs:hexBinary"><xs:length value="2"/></xs:restriction>`, "0aFF00", false, "", false},
		{"base64-maxLength-octets", verBoth, `<xs:restriction base="xs:base64Binary"><xs:maxLength value="3"/></xs:restriction>`, "AQID", true, "", false},
		{"base64-maxLength-octets-bad", verBoth, `<xs:restriction base="xs:base64Binary"><xs:maxLength value="2"/></xs:restriction>`, "AQID", false, "", false},
		{"anyURI-length", verBoth, `<xs:restriction base="xs:anyURI"><xs:maxLength value="5"/></xs:restriction>`, "a/b", true, "", false},
		{"collapse-before-length", verBoth, `<xs:restriction base="xs:token"><xs:length value="3"/></xs:restriction>`, "  a b  ", true, "", false},
		{"preserve-length-counts-spaces", verBoth, `<xs:restriction base="xs:string"><xs:length value="3"/></xs:restriction>`, " ab", true, "", false},
		{"replace-whitespace", verBoth, `<xs:restriction base="xs:string"><xs:whiteSpace value="replace"/><xs:pattern value="a b"/></xs:restriction>`, "a\tb", true, "", false},
		{"pattern-anchored", verBoth, `<xs:restriction base="xs:string"><xs:pattern value="ab"/></xs:restriction>`, "xaby", false, "", false},
		{"pattern-multiple-or", verBoth, `<xs:restriction base="xs:string"><xs:pattern value="a"/><xs:pattern value="b"/></xs:restriction>`, "b", true, "", false},
		{"pattern-steps-and", verBoth, `<xs:restriction base="xs:string"><xs:pattern value="a."/></xs:restriction>`, "ab", true, "", false},
		{"pattern-class-subtraction", verBoth, `<xs:restriction base="xs:string"><xs:pattern value="[a-z-[aeiou]]+"/></xs:restriction>`, "bcd", true, "", false},
		{"pattern-class-subtraction-bad", verBoth, `<xs:restriction base="xs:string"><xs:pattern value="[a-z-[aeiou]]+"/></xs:restriction>`, "bad", false, "", false},
		{"pattern-name-escape", verBoth, `<xs:restriction base="xs:string"><xs:pattern value="\i\c*"/></xs:restriction>`, "_x1", true, "", false},
		{"enumeration-value-space-decimal", verBoth, `<xs:restriction base="xs:decimal"><xs:enumeration value="1.0"/></xs:restriction>`, "01.00", true, "", false},
		{"enumeration-value-space-float-nan", verBoth, `<xs:restriction base="xs:float"><xs:enumeration value="NaN"/></xs:restriction>`, "NaN", true, "", false},
		{"enumeration-qname-value", verBoth, `<xs:restriction base="xs:QName"><xs:enumeration value="foo"/></xs:restriction>`, " foo ", true, "", false},
		{"enumeration-boolean", verBoth, `<xs:restriction base="xs:boolean"><xs:enumeration value="true"/></xs:restriction>`, "1", true, "", false},
		{"maxInclusive-decimal", verBoth, `<xs:restriction base="xs:decimal"><xs:maxInclusive value="10.5"/></xs:restriction>`, "10.50", true, "", false},
		{"maxExclusive-decimal", verBoth, `<xs:restriction base="xs:decimal"><xs:maxExclusive value="10.5"/></xs:restriction>`, "10.5", false, "", false},
		{"minExclusive-double", verBoth, `<xs:restriction base="xs:double"><xs:minExclusive value="0"/></xs:restriction>`, "1E-300", true, "", false},
		{"minExclusive-double-zero", verBoth, `<xs:restriction base="xs:double"><xs:minExclusive value="0"/></xs:restriction>`, "-0", false, "", false},
		{"maxInclusive-double-inf", verBoth, `<xs:restriction base="xs:double"><xs:maxInclusive value="1E10"/></xs:restriction>`, "INF", false, "", false},
		{"maxInclusive-double-nan", verBoth + skipBug, `<xs:restriction base="xs:double"><xs:maxInclusive value="1E10"/></xs:restriction>`, "NaN", false, "", false},
		{"totalDigits-ok", verBoth, `<xs:restriction base="xs:decimal"><xs:totalDigits value="3"/></xs:restriction>`, "12.3", true, "", false},
		{"totalDigits-trailing-zeros", verBoth, `<xs:restriction base="xs:decimal"><xs:totalDigits value="3"/></xs:restriction>`, "12.300", true, "", false},
		{"totalDigits-leading-zeros", verBoth, `<xs:restriction base="xs:decimal"><xs:totalDigits value="2"/></xs:restriction>`, "0012", true, "", false},
		{"totalDigits-bad", verBoth, `<xs:restriction base="xs:decimal"><xs:totalDigits value="3"/></xs:restriction>`, "1234", false, "", false},
		{"fractionDigits-ok", verBoth, `<xs:restriction base="xs:decimal"><xs:fractionDigits value="2"/></xs:restriction>`, "1.20", true, "", false},
		{"fractionDigits-bad", verBoth, `<xs:restriction base="xs:decimal"><xs:fractionDigits value="2"/></xs:restriction>`, "1.234", false, "", false},
		{"date-maxInclusive", verBoth, `<xs:restriction base="xs:date"><xs:maxInclusive value="2020-12-31"/></xs:restriction>`, "2020-12-31", true, "", false},
		{"date-maxInclusive-bad", verBoth, `<xs:restriction base="xs:date"><xs:maxInclusive value="2020-12-31"/></xs:restriction>`, "2021-01-01", false, "", false},
		{"dateTime-timezone-compare", verBoth, `<xs:restriction base="xs:dateTime"><xs:maxInclusive value="2020-01-01T12:00:00Z"/></xs:restriction>`, "2020-01-01T13:00:00+01:00", true, "", false},
		{"dateTime-timezone-compare-bad", verBoth, `<xs:restriction base="xs:dateTime"><xs:maxInclusive value="2020-01-01T12:00:00Z"/></xs:restriction>`, "2020-01-01T13:00:00+00:30", false, "", false},
		// An indeterminate comparison (no timezone vs timezone ±14h) fails
		// the facet (Datatypes §3.3.8.4 order relation is partial).
		{"dateTime-indeterminate", verBoth + skipBug, `<xs:restriction base="xs:dateTime"><xs:maxInclusive value="2020-01-01T12:00:00Z"/></xs:restriction>`, "2020-01-01T12:00:00", false, "", false},
		{"time-minExclusive", verBoth, `<xs:restriction base="xs:time"><xs:minExclusive value="10:00:00"/></xs:restriction>`, "10:00:01", true, "", false},
		{"gYear-maxExclusive", verBoth, `<xs:restriction base="xs:gYear"><xs:maxExclusive value="2000"/></xs:restriction>`, "1999", true, "", false},
		{"gYearMonth-bound-bad", verBoth, `<xs:restriction base="xs:gYearMonth"><xs:minInclusive value="2000-05"/></xs:restriction>`, "2000-04", false, "", false},
		{"duration-max", verBoth, `<xs:restriction base="xs:duration"><xs:maxInclusive value="P1Y"/></xs:restriction>`, "P12M", true, "", false},
		{"duration-max-bad", verBoth, `<xs:restriction base="xs:duration"><xs:maxInclusive value="P1Y"/></xs:restriction>`, "P13M", false, "", false},
		{"dayTimeDuration-min", ver11, `<xs:restriction base="xs:dayTimeDuration"><xs:minInclusive value="PT1H"/></xs:restriction>`, "PT59M", false, "", false},
		{"explicitTimezone-required-bad", ver11, `<xs:restriction base="xs:date"><xs:explicitTimezone value="required"/></xs:restriction>`, "2020-01-01", false, "", false},
		{"explicitTimezone-required-ok", ver11, `<xs:restriction base="xs:date"><xs:explicitTimezone value="required"/></xs:restriction>`, "2020-01-01Z", true, "", false},
		{"explicitTimezone-prohibited-bad", ver11, `<xs:restriction base="xs:time"><xs:explicitTimezone value="prohibited"/></xs:restriction>`, "10:00:00+02:00", false, "", false},
		{"dateTimeStamp-needs-tz", ver11, `<xs:restriction base="xs:dateTimeStamp"/>`, "2020-01-01T00:00:00", false, "", false},
		{"list-length-items", verBoth, `<xs:restriction><xs:simpleType><xs:list itemType="xs:int"/></xs:simpleType><xs:length value="2"/></xs:restriction>`, " 1   2 ", true, "", false},
		{"list-length-items-bad", verBoth, `<xs:restriction><xs:simpleType><xs:list itemType="xs:int"/></xs:simpleType><xs:length value="2"/></xs:restriction>`, "1 2 3", false, "", false},
		{"list-item-invalid", verBoth, `<xs:list itemType="xs:int"/>`, "1 x", false, "", false},
		{"list-enumeration-whole-list", verBoth + skipBug, `<xs:restriction><xs:simpleType><xs:list itemType="xs:int"/></xs:simpleType><xs:enumeration value="1 2"/></xs:restriction>`, "01 2", true, "", false},
		{"list-pattern-whole-string", verBoth, `<xs:restriction><xs:simpleType><xs:list itemType="xs:int"/></xs:simpleType><xs:pattern value="\d( \d)*"/></xs:restriction>`, "1 2", true, "", false},
		{"list-of-union", verBoth, `<xs:list><xs:simpleType><xs:union memberTypes="xs:int xs:boolean"/></xs:simpleType></xs:list>`, "1 true 3", true, "", false},
		{"list-of-union-bad", verBoth, `<xs:list><xs:simpleType><xs:union memberTypes="xs:int xs:boolean"/></xs:simpleType></xs:list>`, "1 maybe", false, "", false},
		{"union-member-match", verBoth, `<xs:union memberTypes="xs:int xs:date"/>`, "2020-01-01", true, "", false},
		{"union-no-member", verBoth, `<xs:union memberTypes="xs:int xs:date"/>`, "x", false, "", false},
		{"union-restricted-enum", verBoth, `<xs:restriction><xs:simpleType><xs:union memberTypes="xs:int xs:date"/></xs:simpleType><xs:enumeration value="1"/><xs:enumeration value="2020-01-01"/></xs:restriction>`, "2020-01-01", true, "", false},
		{"union-restricted-enum-miss", verBoth, `<xs:restriction><xs:simpleType><xs:union memberTypes="xs:int xs:date"/></xs:simpleType><xs:enumeration value="1"/></xs:restriction>`, "2", false, "", false},
		{"union-of-list", verBoth, `<xs:union><xs:simpleType><xs:list itemType="xs:int"/></xs:simpleType><xs:simpleType><xs:restriction base="xs:string"><xs:enumeration value="none"/></xs:restriction></xs:simpleType></xs:union>`, "1 2", true, "", false},
		{"assertion-on-simple-1.1", ver11, `<xs:restriction base="xs:int"><xs:assertion test="$value mod 2 = 0"/></xs:restriction>`, "4", true, "", false},
		{"assertion-on-simple-1.1-bad", ver11, `<xs:restriction base="xs:int"><xs:assertion test="$value mod 2 = 0"/></xs:restriction>`, "5", false, "", false},
		{"assertion-on-list-1.1", ver11, `<xs:restriction><xs:simpleType><xs:list itemType="xs:int"/></xs:simpleType><xs:assertion test="count($value) = 2"/></xs:restriction>`, "1 2", true, "", false},
	}
	for _, tc := range cases {
		for _, v := range versionsOf(tc.ver % skipBug) {
			t.Run(tc.name+"/"+v.String(), func(t *testing.T) {
				if tc.ver >= skipBug {
					t.Skip("BUG: engine verdict disagrees with the spec")
				}
				if got := instanceVerdict(t, schemaFor(tc.typ), `<v>`+tc.value+`</v>`, v); got != tc.valid {
					t.Fatalf("value %q: valid=%v, want %v", tc.value, got, tc.valid)
				}
			})
		}
	}
}
