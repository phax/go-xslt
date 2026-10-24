package xsd

// Hand-written recognizers for the three lexical regexps an instance's
// values pass through most often (every integer, decimal and ID/IDREF value).
// Each accepts exactly the language of the regexp it stands for —
// lexscan_test.go checks them against it as the oracle — and costs a byte
// loop instead of a regexp machine run.

// isIntegerLex reports whether v matches reInteger, ^[+-]?[0-9]+$.
func isIntegerLex(v string) bool {
	i := 0
	if i < len(v) && (v[i] == '+' || v[i] == '-') {
		i++
	}
	if i == len(v) {
		return false
	}
	for ; i < len(v); i++ {
		if v[i] < '0' || v[i] > '9' {
			return false
		}
	}
	return true
}

// isDecimalLex reports whether v matches reDecimal,
// ^[+-]?([0-9]+\.?[0-9]*|\.[0-9]+)$: digits with an optional point and
// optional fraction digits, or a point followed by at least one digit.
func isDecimalLex(v string) bool {
	i := 0
	if i < len(v) && (v[i] == '+' || v[i] == '-') {
		i++
	}
	intDigits := 0
	for i < len(v) && v[i] >= '0' && v[i] <= '9' {
		i++
		intDigits++
	}
	fracDigits := 0
	if i < len(v) && v[i] == '.' {
		i++
		for i < len(v) && v[i] >= '0' && v[i] <= '9' {
			i++
			fracDigits++
		}
	}
	return i == len(v) && (intDigits > 0 || fracDigits > 0)
}

// isNCNameLex reports whether v matches reNCName. An all-ASCII value is
// decided here — the ASCII part of NameStartChar minus ':' is [A-Za-z_], and
// NameChar adds [-.0-9] — anything else is left to the regexp.
func isNCNameLex(v string) bool {
	if len(v) == 0 {
		return false
	}
	for i := 0; i < len(v); i++ {
		c := v[i]
		switch {
		case c >= 0x80:
			return reNCName.MatchString(v)
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c == '_':
		case i > 0 && (c >= '0' && c <= '9' || c == '-' || c == '.'):
		default:
			// An ASCII byte no NCName may hold at this position; a later
			// non-ASCII byte cannot make the value match.
			return false
		}
	}
	return true
}
