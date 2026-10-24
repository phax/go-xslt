package xpath

import "testing"

// TestWordsLangFormatInteger drives the German/French/Italian spellout through
// fn:format-integer's w/W/Ww tokens and the 'o' ordinal modifier (F&O 3.1
// §4.6.1) with $lang. Expected words are the standard (CLDR) spellings.
func TestWordsLangFormatInteger(t *testing.T) {
	dtfRun(t, []dtfCase{
		// German
		{expr: `format-integer(30, 'W', 'de')`, want: "DREISSIG"},
		{expr: `format-integer(21, 'Ww', 'de')`, want: "Einundzwanzig"},
		{expr: `format-integer(-21, 'w', 'de')`, want: "-einundzwanzig"},
		{expr: `format-integer(1100, 'w', 'de')`, want: "eintausendeinhundert"},
		{expr: `format-integer(3000000, 'w', 'de')`, want: "drei Millionen"},
		{expr: `format-integer(1000000000, 'w', 'de')`, want: "eine Milliarde"},
		{expr: `format-integer(2000000000, 'w', 'de')`, want: "zwei Milliarden"},
		{expr: `format-integer(1000000001, 'w', 'de')`, want: "eine Milliarde eins"},
		{expr: `format-integer(31, 'w;o', 'de-AT')`, want: "einunddreißigste"},
		{expr: `format-integer(2, 'w;o', 'de')`, want: "zweite"},
		// French
		{expr: `format-integer(30, 'w', 'fr')`, want: "trente"},
		{expr: `format-integer(32, 'w', 'fr')`, want: "trente-deux"},
		{expr: `format-integer(72, 'w', 'fr')`, want: "soixante-douze"},
		{expr: `format-integer(300, 'w', 'fr')`, want: "trois cents"},
		{expr: `format-integer(180, 'w', 'fr')`, want: "cent quatre-vingts"},
		{expr: `format-integer(2000, 'w', 'fr')`, want: "deux mille"},
		{expr: `format-integer(1000000, 'w', 'fr')`, want: "un million"},
		{expr: `format-integer(1000000000, 'w', 'fr')`, want: "un milliard"},
		{expr: `format-integer(2000000000, 'w', 'fr')`, want: "deux milliards"},
		{expr: `format-integer(21, 'W', 'fr-CA')`, want: "VINGT ET UN"},
		{expr: `format-integer(3, 'w;o', 'fr')`, want: "troisième"},
		{expr: `format-integer(11, 'w;o', 'fr')`, want: "onzième"},
		{expr: `format-integer(200, 'w;o', 'fr')`, want: "deux centième"},
		{expr: `format-integer(2000, 'w;o', 'fr')`, want: "deux millième"},
		{expr: `format-integer(1000000, 'w;o', 'fr')`, want: "un millionième"},
		// Italian
		{expr: `format-integer(21, 'W', 'it')`, want: "VENTUNO"},
		{expr: `format-integer(33, 'w', 'it')`, want: "trentatré"},
		{expr: `format-integer(2023, 'w', 'it')`, want: "duemilaventitré"},
		{expr: `format-integer(1000000000, 'w', 'it')`, want: "un miliardo"},
		{expr: `format-integer(3000000000, 'w', 'it')`, want: "tre miliardi"},
		{expr: `format-integer(1000, 'w;o', 'it')`, want: "millesimo"},
		{expr: `format-integer(2000, 'w;o', 'it')`, want: "duemillesimo"},
		{expr: `format-integer(33, 'w;o', 'it')`, want: "trentatreesimo"},
		{expr: `format-integer(3, 'w;o', 'it')`, want: "terzo"},
	})
}

// TestWordsLangSpellNumber checks SpellNumber's contract: native spellout for
// de/fr/it below 10^12, ok == false otherwise.
func TestWordsLangSpellNumber(t *testing.T) {
	for _, c := range []struct {
		n       int64
		token   string
		ordinal bool
		lang    string
		want    string
		ok      bool
	}{
		{21, "w", false, "de", "einundzwanzig", true},
		{-21, "w", false, "de_CH", "-einundzwanzig", true},
		{1, "w", true, "it", "primo", true},
		{1, "Ww", false, "fr", "Un", true},
		{21, "w", false, "sv", "", false},
		{21, "w", false, "", "", false},
		{1_000_000_000_000, "w", false, "de", "", false},
		{-1_000_000_000_000, "w", false, "fr", "", false},
	} {
		got, ok := SpellNumber(c.n, c.token, c.ordinal, c.lang, "")
		if got != c.want || ok != c.ok {
			t.Errorf("SpellNumber(%d, %q, %v, %q) = (%q, %v), want (%q, %v)",
				c.n, c.token, c.ordinal, c.lang, got, ok, c.want, c.ok)
		}
	}
}
