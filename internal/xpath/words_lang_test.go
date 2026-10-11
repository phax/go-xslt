package xpath

import "testing"

// TestWordsLangCardinalOrdinal pins the German/French/Italian spellout against
// well-known correct forms (elisions, accents, plural -s, "et" before 1).
func TestWordsLangCardinalOrdinal(t *testing.T) {
	type tc struct {
		lang    string
		n       int64
		ordinal bool
		variant string
		want    string
	}
	cases := []tc{
		// German
		{"de", 0, false, "", "null"},
		{"de", 1, false, "", "eins"},
		{"de", 21, false, "", "einundzwanzig"},
		{"de", 100, false, "", "einhundert"},
		{"de", 101, false, "", "einhunderteins"},
		{"de", 1234, false, "", "eintausendzweihundertvierunddreißig"},
		{"de", 1_000_000, false, "", "eine Million"},
		{"de", 2_000_000, false, "", "zwei Millionen"},
		{"de", 1_234_567, false, "", "eine Million zweihundertvierunddreißigtausendfünfhundertsiebenundsechzig"},
		{"de", 1, true, "", "erste"},
		{"de", 1, true, "-er", "erster"},
		{"de", 3, true, "", "dritte"},
		{"de", 7, true, "", "siebte"},
		{"de", 8, true, "", "achte"},
		{"de", 19, true, "", "neunzehnte"},
		{"de", 20, true, "%spellout-ordinal", "zwanzigste"},
		{"de", 21, true, "", "einundzwanzigste"},
		{"de", 101, true, "", "einhunderterste"},
		{"de", 1000, true, "", "eintausendste"},
		// French
		{"fr", 0, false, "", "zéro"},
		{"fr", 21, false, "", "vingt et un"},
		{"fr", 71, false, "", "soixante et onze"},
		{"fr", 80, false, "", "quatre-vingts"},
		{"fr", 81, false, "", "quatre-vingt-un"},
		{"fr", 97, false, "", "quatre-vingt-dix-sept"},
		{"fr", 200, false, "", "deux cents"},
		{"fr", 201, false, "", "deux cent un"},
		{"fr", 1000, false, "", "mille"},
		{"fr", 80000, false, "", "quatre-vingt mille"},
		{"fr", 200000, false, "", "deux cent mille"},
		{"fr", 2_000_000, false, "", "deux millions"},
		{"fr", 1, true, "", "premier"},
		{"fr", 1, true, "-e", "première"},
		{"fr", 2, true, "", "deuxième"},
		{"fr", 4, true, "", "quatrième"},
		{"fr", 5, true, "", "cinquième"},
		{"fr", 9, true, "", "neuvième"},
		{"fr", 21, true, "", "vingt et unième"},
		{"fr", 80, true, "", "quatre-vingtième"},
		{"fr", 1000, true, "", "millième"},
		// Italian
		{"it", 0, false, "", "zero"},
		{"it", 21, false, "", "ventuno"},
		{"it", 23, false, "", "ventitré"},
		{"it", 28, false, "", "ventotto"},
		{"it", 180, false, "", "centottanta"},
		{"it", 200, false, "", "duecento"},
		{"it", 1000, false, "", "mille"},
		{"it", 2000, false, "", "duemila"},
		{"it", 23000, false, "", "ventitremila"},
		{"it", 2300, false, "", "duemilatrecento"},
		{"it", 1_000_000, false, "", "un milione"},
		{"it", 2_000_000, false, "", "due milioni"},
		{"it", 5, true, "-o", "quinto"},
		{"it", 5, true, "-a", "quinta"},
		{"it", 5, true, "%spellout-ordinal-feminine", "quinta"},
		{"it", 10, true, "", "decimo"},
		{"it", 11, true, "", "undicesimo"},
		{"it", 20, true, "", "ventesimo"},
		{"it", 21, true, "", "ventunesimo"},
		{"it", 23, true, "", "ventitreesimo"},
		{"it", 26, true, "", "ventiseiesimo"},
		{"it", 100, true, "", "centesimo"},
		{"it", 1000, true, "", "millesimo"},
		{"it", 2000, true, "", "duemillesimo"},
		{"it", 1_000_000, true, "", "un milionesimo"},
	}
	for _, c := range cases {
		wl := wordLangFor(c.lang)
		var got string
		if c.ordinal {
			got = wl.ordinal(c.n, c.variant)
		} else {
			got = wl.cardinal(c.n)
		}
		if got != c.want {
			t.Errorf("%s %d ordinal=%v variant=%q: got %q, want %q", c.lang, c.n, c.ordinal, c.variant, got, c.want)
		}
	}
}
