package xpath

import (
	"strings"
	"testing"

	"github.com/tim-riep/go-xslt/internal/xmltree"
)

// Expected values in this file come from F&O 3.1 §4.4 (abs, ceiling, floor,
// round, round-half-to-even), §4.6.1 (format-integer), §4.7 (format-number)
// and §4.8.14 (math:atan2) — mostly the REC's own examples, otherwise derived
// directly from the numbered rules of those sections.

// numNS is a static namespace context for the decimal-format-name tests.
type numNS map[string]string

func (m numNS) ResolveNS(p string) (string, bool) { u, ok := m[p]; return u, ok }

// numEval parses and evaluates expr against <r/> in a context prepared by
// setup (may be nil), returning the result or the first error.
func numEval(expr string, setup func(*Context)) (Object, error) {
	p, err := Parse(expr)
	if err != nil {
		return nil, err
	}
	doc, _ := xmltree.Parse("<r/>")
	ctx := &Context{Node: doc, Pos: 1, Size: 1}
	if setup != nil {
		setup(ctx)
	}
	return p.Eval(ctx)
}

type numCase struct{ expr, want string }

func numRun(t *testing.T, cases []numCase, setup func(*Context)) {
	t.Helper()
	for _, c := range cases {
		v, err := numEval(c.expr, setup)
		if err != nil {
			t.Errorf("%s: unexpected error %v", c.expr, err)
			continue
		}
		if got := ToString(v); got != c.want {
			t.Errorf("%s = %q, want %q", c.expr, got, c.want)
		}
	}
}

type numErrCase struct{ expr, code string }

func numRunErr(t *testing.T, cases []numErrCase, setup func(*Context)) {
	t.Helper()
	for _, c := range cases {
		v, err := numEval(c.expr, setup)
		if err == nil {
			t.Errorf("%s = %q, want error %s", c.expr, ToString(v), c.code)
			continue
		}
		if !strings.Contains(err.Error(), c.code) {
			t.Errorf("%s: error %v, want %s", c.expr, err, c.code)
		}
	}
}

func TestNumRoundingFamilySpec(t *testing.T) {
	numRun(t, []numCase{
		// §4.4.1 fn:abs examples + type preservation.
		{"abs(10.5)", "10.5"},
		{"abs(-10.5)", "10.5"},
		{"abs(-3)", "3"},
		{"abs(-3) instance of xs:integer", "true"},
		{"abs(-10.5) instance of xs:decimal", "true"},
		{"abs(xs:float('-1.5'))", "1.5"},
		{"abs(xs:float('-1.5')) instance of xs:float", "true"},
		{"abs(-2.5e0)", "2.5"},
		{"abs(-2.5e0) instance of xs:double", "true"},
		{"empty(abs(()))", "true"},
		// §4.4.2 fn:ceiling examples; -1 < x < 0 gives negative zero.
		{"ceiling(10.5)", "11"},
		{"ceiling(-10.5)", "-10"},
		{"ceiling(7)", "7"},
		{"ceiling(xs:float('1.2'))", "2"},
		{"ceiling(xs:float('1.2')) instance of xs:float", "true"},
		{"ceiling(1.2e0)", "2"},
		{"ceiling(-0.5e0)", "-0"},
		{"empty(ceiling(()))", "true"},
		// §4.4.3 fn:floor examples.
		{"floor(10.5)", "10"},
		{"floor(-10.5)", "-11"},
		{"floor(7)", "7"},
		{"floor(xs:float('-1.2'))", "-2"},
		{"floor(xs:float('-1.2')) instance of xs:float", "true"},
		{"floor(-1.2e0)", "-2"},
		{"floor(-0e0)", "-0"},
		{"empty(floor(()))", "true"},
		// §4.4.4 fn:round examples.
		{"round(2.5)", "3"},
		{"round(2.4999)", "2"},
		{"round(-2.5)", "-2"},
		{"round(1.125, 2)", "1.13"},
		{"round(8452, -2)", "8500"},
		{"round(8452, -2) instance of xs:integer", "true"},
		{"round(8452, 2)", "8452"},
		{"round(3.1415e0, 2)", "3.14"},
		{"round(3.1415e0, 2) instance of xs:double", "true"},
		{"round(xs:float('2.5'))", "3"},
		{"round(xs:float('2.5')) instance of xs:float", "true"},
		// A zero result keeps the sign of the argument (§4.4.4 rules).
		{"round(-0.4e0)", "-0"},
		{"round(xs:double('NaN'))", "NaN"},
		{"round(xs:double('-INF'))", "-INF"},
		// Huge precisions: more places than the value has leaves it unchanged,
		// a power of ten above any value gives zero.
		{"round(1.5, 5000)", "1.5"},
		{"round(12345, -5000)", "0"},
		{"round(1.5e0, 400)", "1.5"},
		{"round(1.5e300, -400)", "0"},
		{"empty(round(()))", "true"},
		// xs:untypedAtomic is cast to xs:double (§4.4 general rules).
		{"round(xs:untypedAtomic('2.5'))", "3"},
		{"round(xs:untypedAtomic('2.5')) instance of xs:double", "true"},
	}, nil)
	numRunErr(t, []numErrCase{
		{"round('2.5')", "XPTY0004"},
		{"abs('x')", "XPTY0004"},
		{"floor(true())", "XPTY0004"},
		{"ceiling(xs:untypedAtomic('abc'))", "FORG0001"},
	}, nil)
}

func TestNumRoundingUntypedNode(t *testing.T) {
	if got := xpStrCtx(t, "round(/r)", "<r>2.5</r>"); got != "3" {
		t.Errorf("round(/r) = %q, want 3", got)
	}
	if got := xpStrCtx(t, "abs(/r) instance of xs:double", "<r>-4</r>"); got != "true" {
		t.Errorf("abs(untyped) not a double: %q", got)
	}
}

func TestNumRoundHalfToEvenSpec(t *testing.T) {
	numRun(t, []numCase{
		// §4.4.5 examples.
		{"round-half-to-even(0.5)", "0"},
		{"round-half-to-even(1.5)", "2"},
		{"round-half-to-even(2.5)", "2"},
		{"round-half-to-even(3.567812e+3, 2)", "3567.81"},
		{"round-half-to-even(4.7564e-3, 2)", "0"},
		{"round-half-to-even(35612.25, -2)", "35600"},
		// Ties to even on both signs, integers at negative precision.
		{"round-half-to-even(-2.5)", "-2"},
		{"round-half-to-even(-3.5)", "-4"},
		{"round-half-to-even(-3.6)", "-4"},
		{"round-half-to-even(1350, -2)", "1400"},
		{"round-half-to-even(-1250, -2)", "-1200"},
		{"round-half-to-even(1251, -2)", "1300"},
		{"round-half-to-even(1250, -2) instance of xs:integer", "true"},
		// xs:float keeps its type; 1.25 is exact in binary, so it is a tie.
		{"round-half-to-even(xs:float('1.25'), 1)", "1.2"},
		{"round-half-to-even(xs:float('1.25'), 1) instance of xs:float", "true"},
		{"round-half-to-even(xs:float('NaN'))", "NaN"},
		{"round-half-to-even(xs:float('INF'))", "INF"},
		{"round-half-to-even(xs:double('INF'))", "INF"},
		{"round-half-to-even(xs:double('NaN'))", "NaN"},
		// The double nearest 2.675 is below it: no tie (cast to exact decimal).
		{"round-half-to-even(2.675e0, 2)", "2.67"},
		{"round-half-to-even(-0.5e0)", "-0"},
		// Huge precisions.
		{"round-half-to-even(1.5, 5000)", "1.5"},
		{"round-half-to-even(12345, -5000)", "0"},
		{"round-half-to-even(1.5e0, 400)", "1.5"},
		{"round-half-to-even(1.5e0, -400)", "0"},
		{"empty(round-half-to-even(()))", "true"},
		{"round-half-to-even(xs:untypedAtomic('2.5'))", "2"},
	}, nil)
	numRunErr(t, []numErrCase{
		{"round-half-to-even('1')", "XPTY0004"},
		{"round-half-to-even(1.5, 'two')", "XPTY0004"},
	}, nil)
}

func TestNumAtan2Spec(t *testing.T) {
	// §4.8.14 examples.
	numRun(t, []numCase{
		{"math:atan2(+0.0e0, 0.0e0)", "0"},
		{"math:atan2(-0.0e0, 0.0e0)", "-0"},
		{"math:atan2(+0.0e0, -0.0e0)", "3.141592653589793"},
		{"math:atan2(-0.0e0, -0.0e0)", "-3.141592653589793"},
		{"math:atan2(-1, 0.0e0)", "-1.5707963267948966"},
		{"math:atan2(+1, 0.0e0)", "1.5707963267948966"},
		{"math:atan2(-0.0e0, -1)", "-3.141592653589793"},
		{"math:atan2(+0.0e0, -1)", "3.141592653589793"},
		{"math:atan2(-0.0e0, +1)", "-0"},
		{"math:atan2(+0.0e0, +1)", "0"},
		{"math:atan2(xs:double('NaN'), 1)", "NaN"},
		{"empty(math:atan2((), 1))", "true"},
	}, nil)
}

func TestNumFormatIntegerSpec(t *testing.T) {
	numRun(t, []numCase{
		// §4.6.1 examples.
		{"format-integer(123, '0000')", "0123"},
		{"format-integer(123, 'w')", "one hundred and twenty-three"},
		{"format-integer(21, '1;o', 'en')", "21st"},
		{"format-integer(7, 'a')", "g"},
		{"format-integer(57, 'I')", "LVII"},
		{"format-integer(1234, '#;##0;')", "1;234"},
		// Empty value → zero-length string; negatives get a leading minus.
		{"format-integer((), '1')", ""},
		{"format-integer(-57, 'I')", "-LVII"},
		{"format-integer(-5, '000')", "-005"},
		// Notes of §4.6.1: regular grouping extrapolates leftward.
		{"format-integer(1000000, '0''000')", "1'000'000"},
		{"format-integer(15, '0''000')", "0'015"},
		{"format-integer(1000000, '#''##0')", "1'000'000"},
		{"format-integer(15, '#''##0')", "15"},
		{"format-integer(300, '01')", "300"},
		{"format-integer(3, '99')", "03"},
		// Not regular (positions 3 and 5): only the explicit positions.
		{"format-integer(1234567, '#,##,##0')", "12,34,567"},
		// Not regular (multiple 6 of G=2 missing): positional.
		{"format-integer(1234567, '###,##,00')", "123,45,67"},
		// Not regular (different separator characters).
		{"format-integer(1234567, '0.000,000')", "1.234,567"},
		// Positions 2 and 6 are multiples of 2, but 4 is missing: positional.
		{"format-integer(12345678, '00,0000,00')", "12,3456,78"},
		// Regular with G=2 repeating.
		{"format-integer(1234567, '00,00,00')", "1,23,45,67"},
		{"format-integer(12345678901234567890, '#,##0')", "12,345,678,901,234,567,890"},
		// Other digit families (§4.6.1: Arabic-Indic one; Thai).
		{"format-integer(3, '\u0661')", "٣"},
		{"format-integer(123, '๐๐๐๐')", "๐๑๒๓"},
		// Ordinals with digits (English: 1st 2nd 3rd 4th …).
		{"format-integer(2, '1;o')", "2nd"},
		{"format-integer(3, '1;o')", "3rd"},
		{"format-integer(4, '1;o')", "4th"},
		{"format-integer(12, '1;o')", "12th"},
		{"format-integer(113, '1;o')", "113th"},
		{"format-integer(22, '1;o(-e)')", "22nd"},
		{"format-integer(5, '1;c')", "5"},
		{"format-integer(5, '1;c(x)a')", "5"},
		{"format-integer(5, '1;t')", "5"},
		// A/a and I/i sequences.
		{"format-integer(26, 'A')", "Z"},
		{"format-integer(27, 'A')", "AA"},
		{"format-integer(28, 'a')", "ab"},
		{"format-integer(1999, 'i')", "mcmxcix"},
		{"format-integer(4, 'I')", "IV"},
		// 0 has no Roman/alphabetic form: fallback to token 1.
		{"format-integer(0, 'I')", "0"},
		{"format-integer(0, 'a')", "0"},
		// Words (English).
		{"format-integer(7, 'W', 'en')", "SEVEN"},
		{"format-integer(7, 'Ww', 'en')", "Seven"},
		{"format-integer(2, 'Ww;o', 'en')", "Second"},
		{"format-integer(0, 'w', 'en')", "zero"},
		{"format-integer(-3, 'w', 'en')", "-three"},
		{"format-integer(1, 'w;o', 'en')", "first"},
		{"format-integer(21, 'w;o', 'en')", "twenty-first"},
		{"format-integer(20, 'w;o', 'en')", "twentieth"},
		{"format-integer(100, 'w;o', 'en')", "one hundredth"},
		{"format-integer(1000000, 'w', 'en')", "one million"},
		{"format-integer(2000000000, 'w', 'en')", "two billion"},
		{"format-integer(1001, 'w', 'en')", "one thousand and one"},
		{"format-integer(1001, 'w;o', 'en')", "one thousand and first"},
		{"format-integer(1200, 'w', 'en')", "one thousand two hundred"},
		{"format-integer(1200, 'w;o', 'en')", "one thousand two hundredth"},
		// Greek alphabetic sequence and Kanji numerals.
		{"format-integer(1, 'α')", "α"},
		{"format-integer(24, 'Α')", "Ω"},
		{"format-integer(25, 'α')", "αα"},
		{"format-integer(151, '一')", "百五十一"},
		{"format-integer(2025, '一')", "二千二十五"},
		{"format-integer(10, '一')", "十"},
		// Circled digits.
		{"format-integer(3, '①')", "③"},
		{"format-integer(3, '⑴')", "⑶"},
		{"format-integer(3, '⒈')", "⒊"},
		// Unsupported tokens fall back to the token '1'.
		{"format-integer(1500000, '#a')", "1500000"},
		{"format-integer(5, '#')", "5"},
		{"format-integer(5, 'x;o')", "5th"},
	}, nil)
	numRunErr(t, []numErrCase{
		{"format-integer(1, '')", "FODF1310"},
		{"format-integer(1, ';o')", "FODF1310"},
		{"format-integer(1, '0#')", "FODF1310"},
		{"format-integer(1, ',00')", "FODF1310"},
		{"format-integer(1, '00,')", "FODF1310"},
		{"format-integer(1, '0,,0')", "FODF1310"},
		{"format-integer(1, '0١')", "FODF1310"},
		{"format-integer(1, '0a0')", "FODF1310"},
		{"format-integer(1, '1;o(')", "FODF1310"},
		{"format-integer(1, '1;oz')", "FODF1310"},
		{"format-integer(1, '1;at')", "FODF1310"},
		{"format-integer(1, '1;o(-er)z')", "FODF1310"},
	}, nil)
}

func TestNumFormatIntegerDefaultLanguage(t *testing.T) {
	// The two-argument form depends on the default language (§4.6.1 properties).
	numRun(t, []numCase{
		{"format-integer(3, 'w')", "three"},
	}, func(c *Context) { c.DefaultLanguage = "en" })
}

func TestNumFormatNumberSpec(t *testing.T) {
	numRun(t, []numCase{
		// §4.7.2 examples (default decimal format).
		{"format-number(12345.6, '#,###.00')", "12,345.60"},
		{"format-number(12345678.9, '9,999.99')", "12,345,678.90"},
		{"format-number(123.9, '9999')", "0124"},
		{"format-number(0.14, '01%')", "14%"},
		{"format-number(-6, '000')", "-006"},
		// The four exponent examples, with the default exponent separator 'e'.
		{"format-number(1234.5678, '00.000e0')", "12.346e2"},
		{"format-number(0.234, '0.0e0')", "2.3e-1"},
		{"format-number(0.234, '#.00e0')", "0.23e0"},
		{"format-number(0.234, '.00e0')", ".23e0"},
		// §4.7.4 notes: '#.e9', '.9e9', '#.9e9', '#'.
		{"format-number(0.123, '#.e9')", "0.1e0"},
		{"format-number(0.1, '.9e9')", ".1e0"},
		{"format-number(0.1, '#.9e9')", "0.1e0"},
		{"format-number(0.23, '#')", "0"},
		// Minimum exponent size pads the exponent.
		{"format-number(0.234, '0.0e00')", "2.3e-01"},
		{"format-number(5, '0e00')", "5e00"},
		// Per-mille and percent (§4.7.5 adjusted number).
		{"format-number(0.5, '#‰')", "500‰"},
		{"format-number(0.4857, '###.###%')", "48.57%"},
		{"format-number(0.1e0, '0%')", "10%"},
		// Overflow on scaling gives infinity, no error.
		{"format-number(1.7e308, '0%')", "Infinity%"},
		// Sub-pictures: negative uses the second, a sole one gets minus + prefix.
		{"format-number(-6, '0;(0)')", "(6)"},
		{"format-number(0, '0;(0)')", "0"},
		{"format-number(-0e0, '0;(0)')", "(0)"},
		{"format-number(-5, '$0')", "-$5"},
		// NaN / infinity / empty.
		{"format-number(xs:double('NaN'), '0;(0)')", "NaN"},
		{"format-number((), '0')", "NaN"},
		{"format-number(xs:double('INF'), '0')", "Infinity"},
		{"format-number(xs:double('-INF'), '0')", "-Infinity"},
		{"format-number(xs:double('-INF'), '0;(0)')", "(Infinity)"},
		// Shortest decimal representation of doubles / floats.
		{"format-number(1e25, '#')", "10000000000000000000000000"},
		{"format-number(xs:float('0.1'), '0.000')", "0.100"},
		{"format-number(12345678901234567890, '#,###')", "12,345,678,901,234,567,890"},
		// Grouping: regular, irregular, non-regular, fractional.
		{"format-number(1234567, '#,###')", "1,234,567"},
		{"format-number(1234567, '#,##,##0')", "12,34,567"},
		{"format-number(1234567, '####,##')", "12345,67"},
		{"format-number(1234567, ',##0')", "1,234,567"},
		{"format-number(12345.6789, '#.#,##,#')", "12345.6,78,9"},
		// Decimal separator removed when no fractional digit follows; zero.
		{"format-number(5, '0.#')", "5"},
		{"format-number(0, '#.#')", ".0"},
		{"format-number(0.2, '#.#')", ".2"},
		// Rounding is round-half-to-even on the exact decimal.
		{"format-number(0.125, '0.00')", "0.12"},
		{"format-number(0.135, '0.00')", "0.14"},
		// Fewer digits than one group; leading zeros inside the fraction.
		{"format-number(12, '#,###')", "12"},
		{"format-number(1.05, '0.00')", "1.05"},
		// An empty or absent $decimal-format-name means the unnamed format.
		{"format-number(1.5, '0.0', '')", "1.5"},
		{"format-number(1.5, '0.0', ())", "1.5"},
		// Passive prefix and suffix.
		{"format-number(42, 'n=0 units')", "n=42 units"},
	}, nil)
	if got := xpStrCtx(t, "format-number(/r, '0.0')", "<r>2.25</r>"); got != "2.2" {
		t.Errorf("format-number(untyped 2.25, '0.0') = %q, want 2.2", got)
	}
	numRunErr(t, []numErrCase{
		// §4.7.3 FODF1310 rules, one per rule.
		{"format-number(1, '0;0;0')", "FODF1310"},
		{"format-number(1, '#.#.#')", "FODF1310"},
		{"format-number(1, 'abc')", "FODF1310"},
		{"format-number(1, '#%‰')", "FODF1310"},
		{"format-number(1, '#%%')", "FODF1310"},
		{"format-number(1, '#‰‰')", "FODF1310"},
		{"format-number(1, '0$0')", "FODF1310"},
		{"format-number(1, '#,.0')", "FODF1310"},
		{"format-number(1, '#.,0')", "FODF1310"},
		{"format-number(1, '#,')", "FODF1310"},
		{"format-number(1, '#,,#')", "FODF1310"},
		{"format-number(1, '0#')", "FODF1310"},
		{"format-number(1, '#.#0')", "FODF1310"},
		{"format-number(1, '0e0e0')", "FODF1310"},
		{"format-number(1, '0e0%')", "FODF1310"},
		{"format-number(1, '0e#')", "FODF1310"},
		{"format-number(1, '0e0#')", "FODF1310"},
		{"format-number(1, '.e0')", "FODF1310"},
		// 'E' is not the default exponent separator: a passive char inside.
		{"format-number(1, '0E0')", "FODF1310"},
		// $picture is xs:string.
		{"format-number(1, 2)", "XPTY0004"},
		// No decimal format of that name in the static context.
		{"format-number(1, '0', 'nope')", "FODF1280"},
	}, nil)
}

func TestNumFormatNumberNamedFormats(t *testing.T) {
	ch := DefaultDecimalFormat()
	ch.GroupingSep, ch.DecimalSep = 'ʹ', '·'
	fortran := DefaultDecimalFormat()
	fortran.ExponentSep = 'E'
	ar := DefaultDecimalFormat()
	ar.ZeroDigit = '٠'
	ar.DecimalSep = '٫'
	symbols := DefaultDecimalFormat()
	symbols.NaN, symbols.Infinity, symbols.MinusSign = "nan", "∞", '~'
	symbols.PatternSep, symbols.Digit, symbols.Percent = '|', 'x', 'p'
	legacy := DefaultDecimalFormat()
	legacy.ExponentSep = 0 // unset field: treated as 'e'
	setup := func(c *Context) {
		c.DecimalFormats = map[string]DecimalFormat{
			"":                        DefaultDecimalFormat(),
			"ch":                      ch,
			"fortran":                 fortran,
			"ar":                      ar,
			"sym":                     symbols,
			"legacy":                  legacy,
			"{http://example.com/d}f": fortran,
			"{http://www.w3.org/2005/xpath-functions}fn": ch,
		}
		c.NS = numNS{"d": "http://example.com/d"}
	}
	numRun(t, []numCase{
		// §4.7.2 examples with named formats.
		{"format-number(1234.5678, '#ʹ##0·00', 'ch')", "1ʹ234·57"},
		{"format-number(1234.5678, '00.000E0', 'fortran')", "12.346E2"},
		{"format-number(0.234, '0.0E0', 'fortran')", "2.3E-1"},
		{"format-number(0.234, '#.00E0', 'fortran')", "0.23E0"},
		{"format-number(0.234, '.00E0', 'fortran')", ".23E0"},
		// The name is an EQName: whitespace stripped, prefixes from the
		// static context, Q{uri}local.
		{"format-number(1234.5678, '#ʹ##0·00', ' ch ')", "1ʹ234·57"},
		{"format-number(0.234, '0.0E0', 'd:f')", "2.3E-1"},
		{"format-number(0.234, '0.0E0', 'Q{http://example.com/d}f')", "2.3E-1"},
		{"format-number(1234.5678, '#ʹ##0·00', 'Q{}ch')", "1ʹ234·57"},
		{"format-number(1234.5678, '#ʹ##0·00', 'fn:fn')", "1ʹ234·57"},
		// Digits of the zero-digit family (Arabic-Indic), also in exponents.
		{"format-number(1234.5, '٠٫٠', 'ar')", "١٢٣٤٫٥"},
		{"format-number(0.234, '٠٫٠e٠', 'ar')", "٢٫٣e-١"},
		// Other symbols: NaN, infinity, minus-sign, pattern-separator, digit,
		// percent.
		{"format-number(number('x'), '0', 'sym')", "nan"},
		{"format-number(xs:double('-INF'), '0', 'sym')", "~∞"},
		{"format-number(-5, 'x0', 'sym')", "~5"},
		{"format-number(-5, '0|[0]', 'sym')", "[5]"},
		{"format-number(0.25, '0p', 'sym')", "25p"},
		// An unset exponent separator still means 'e'.
		{"format-number(0.234, '0.0e0', 'legacy')", "2.3e-1"},
		// With no name the unnamed format is used.
		{"format-number(1.5, '0.0')", "1.5"},
	}, setup)
	numRunErr(t, []numErrCase{
		{"format-number(1, '0', 'unknown')", "FODF1280"},
		{"format-number(1, '0;0;0', 'ch')", "FODF1310"},
		{"format-number(1, '0|0|0', 'sym')", "FODF1310"},
	}, setup)
}

func TestNumFormatNumberPic(t *testing.T) {
	// formatNumberPic is the double entry point of the formatting phase.
	df := DefaultDecimalFormat()
	for _, c := range []struct {
		v    float64
		pic  string
		want string
	}{
		{12345.6, "#,###.00", "12,345.60"},
		{-6, "000", "-006"},
		{0.234, "0.0e0", "2.3e-1"},
	} {
		if got := formatNumberPic(c.v, c.pic, df); got != c.want {
			t.Errorf("formatNumberPic(%v, %q) = %q, want %q", c.v, c.pic, got, c.want)
		}
	}
}
