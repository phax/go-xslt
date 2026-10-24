package xpath

import "testing"

// TestCaDateTimeLexical checks casting strings to the date/time types (F&O
// 3.1 §19.2, lexical spaces of XSD 1.1 Part 2 §3.3.7–3.3.15) and their
// canonical string forms (F&O §19.1.2.2 / XSD canonical mappings).
func TestCaDateTimeLexical(t *testing.T) {
	caRun(t, "", nil, []caCase{
		{expr: "string(xs:gYear('2000'))", want: "2000"},
		{expr: "string(xs:gYear('-0012'))", want: "-0012"},
		{expr: "string(xs:gYear('2000Z'))", want: "2000Z"},
		{expr: "string(xs:gYear('12345'))", want: "12345"},
		{expr: "string(xs:gYearMonth('2000-02-05:00'))", want: "2000-02-05:00"},
		{expr: "string(xs:gMonth('--05'))", want: "--05"},
		{expr: "string(xs:gMonthDay('--02-29'))", want: "--02-29"},
		{expr: "string(xs:gDay('---31+14:00'))", want: "---31+14:00"},
		{expr: "string(xs:date('2000-01-01-05:30'))", want: "2000-01-01-05:30"},
		{expr: "string(xs:date('2000-01-01+00:00'))", want: "2000-01-01Z"},
		{expr: "string(xs:dateTimeStamp('2000-01-01T00:00:00Z'))", want: "2000-01-01T00:00:00Z"},
		{expr: "string(xs:time('24:00:00'))", want: "00:00:00"},
		{expr: "string(xs:dateTime('2000-12-31T24:00:00'))", want: "2001-01-01T00:00:00"},
		{expr: "string(xs:time('10:00:00.500'))", want: "10:00:00.5"},
		{expr: "xs:time('10:00:00.1234567891') gt xs:time('10:00:00.123')", want: "true"},
		{expr: "string(xs:date(' 2000-01-01 '))", want: "2000-01-01"},
		{expr: "string(xs:date('0000-01-01'))", want: "0000-01-01"},
		{expr: "xs:gMonthDay('--02-30')", code: "FORG0001"},
		{expr: "xs:gDay('---32')", code: "FORG0001"},
		{expr: "xs:gMonth('--13')", code: "FORG0001"},
		{expr: "xs:gYear('02000')", code: "FORG0001"},
		{expr: "xs:gYear('20')", code: "FORG0001"},
		{expr: "xs:gYearMonth('2000')", code: "FORG0001"},
		{expr: "xs:gMonth('05')", code: "FORG0001"},
		{expr: "xs:gMonthDay('--5-1')", code: "FORG0001"},
		{expr: "xs:gDay('--01')", code: "FORG0001"},
		{expr: "xs:date('2000-13-01')", code: "FORG0001"},
		{expr: "xs:date('2001-02-29')", code: "FORG0001"},
		{expr: "xs:date('2000-01-01+14:01')", code: "FORG0001"},
		{expr: "xs:date('2000-01-01+15:00')", code: "FORG0001"},
		{expr: "xs:date('2000-01-01-14:00') lt xs:date('2000-01-02+14:00')", want: "false"},
		{expr: "xs:time('24:30:00')", code: "FORG0001"},
		{expr: "xs:time('10:60:00')", code: "FORG0001"},
		{expr: "xs:time('10:00:60')", code: "FORG0001"},
		{expr: "xs:time('10:00')", code: "FORG0001"},
		{expr: "xs:dateTime('2000-01-01')", code: "FORG0001"},
		{expr: "xs:date('1999-02-29')", code: "FORG0001"},
		{expr: "string(xs:date('2000-02-29'))", want: "2000-02-29"},
		{expr: "string(xs:date('1900-02-28'))", want: "1900-02-28"},
		{expr: "xs:date('1900-02-29')", code: "FORG0001"},
		{expr: "xs:date('2000-04-31')", code: "FORG0001"},
		{expr: "xs:gYear('1234567890')", code: "FODT0001"},
	})
}

// TestCaDurationLexical checks casting to the duration types and their
// canonical forms (F&O 3.1 §8.1, XSD 1.1 Part 2 §3.3.6 and §E.? duration
// canonical mapping: largest units first, zero is PT0S / P0M).
func TestCaDurationLexical(t *testing.T) {
	caRun(t, "", nil, []caCase{
		{expr: "string(xs:duration('-P1Y2M3DT4H'))", want: "-P1Y2M3DT4H"},
		{expr: "string(xs:duration('P12M'))", want: "P1Y"},
		{expr: "string(xs:duration('P1M'))", want: "P1M"},
		{expr: "string(xs:duration('P0Y'))", want: "PT0S"},
		{expr: "string(xs:duration('PT0.5S'))", want: "PT0.5S"},
		{expr: "string(xs:duration('-PT1S'))", want: "-PT1S"},
		{expr: "string(xs:duration('P1YT36H'))", want: "P1Y1DT12H"},
		{expr: "string(xs:duration(' P1D '))", want: "P1D"},
		{expr: "string(xs:yearMonthDuration('-P13M'))", want: "-P1Y1M"},
		{expr: "string(xs:yearMonthDuration('P0Y'))", want: "P0M"},
		{expr: "string(xs:yearMonthDuration('P2Y'))", want: "P2Y"},
		{expr: "string(xs:dayTimeDuration('-PT90M'))", want: "-PT1H30M"},
		{expr: "string(xs:dayTimeDuration('PT0S'))", want: "PT0S"},
		{expr: "string(xs:dayTimeDuration('P1D'))", want: "P1D"},
		{expr: "string(xs:dayTimeDuration('PT61.25S'))", want: "PT1M1.25S"},
		{expr: "string(xs:dayTimeDuration('P200000D'))", want: "P200000D"},
		{expr: "string(xs:dayTimeDuration('P200000DT1H2M3.5S'))", want: "P200000DT1H2M3.5S"},
		{expr: "string(xs:dayTimeDuration('P200000DT3S'))", want: "P200000DT3S"},
		{expr: "string(xs:dayTimeDuration('-P200000DT2M'))", want: "-P200000DT2M"},
		{expr: "string(xs:dayTimeDuration('PT200000000000S'))", want: "P2314814DT19H33M20S"},
		{expr: "xs:duration('P')", code: "FORG0001"},
		{expr: "xs:duration('PT')", code: "FORG0001"},
		{expr: "xs:duration('P1YT')", code: "FORG0001"},
		{expr: "xs:duration('P1H')", code: "FORG0001"},
		{expr: "xs:duration('1Y')", code: "FORG0001"},
		{expr: "xs:duration('P-1Y')", code: "FORG0001"},
		{expr: "xs:duration('P1.5Y')", code: "FORG0001"},
	})
}

// TestCaBinaryLexical checks casting to xs:hexBinary/xs:base64Binary (XSD 1.1
// Part 2 §3.3.16–17: hex digits in pairs, canonical upper case; base64 with
// the padding rules of RFC 2045 as restricted by XSD's lexical grammar).
func TestCaBinaryLexical(t *testing.T) {
	caRun(t, "", nil, []caCase{
		{expr: "string(xs:hexBinary('0f'))", want: "0F"},
		{expr: "string(xs:hexBinary(' 0fA0 '))", want: "0FA0"},
		{expr: "xs:hexBinary('0')", code: "FORG0001"},
		{expr: "xs:hexBinary('0G')", code: "FORG0001"},
		{expr: "xs:base64Binary('AQID BA==') eq xs:base64Binary('AQIDBA==')", want: "true"},
		{expr: "string(xs:base64Binary('AQIDBA=='))", want: "AQIDBA=="},
		{expr: "string(xs:base64Binary(xs:hexBinary('010203')))", want: "AQID"},
		{expr: "string(xs:hexBinary(xs:base64Binary('AQID')))", want: "010203"},
		{expr: "xs:base64Binary('AR==')", code: "FORG0001"},
		{expr: "xs:base64Binary('A')", code: "FORG0001"},
		{expr: "xs:base64Binary('AQ=')", code: "FORG0001"},
	})
}
