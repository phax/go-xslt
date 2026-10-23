package xpath

import "strings"

// Integer spellout (format-integer's w/W/Ww tokens, with or without the "o"
// ordinal modifier) for German, French and Italian. English stays in
// fn_numeric.go. Every function here takes a non-negative n below
// wordsLangMax; callers fall back to English (or digits) beyond that.
//
// The spellings follow CLDR's spellout-cardinal / spellout-ordinal rules for
// the three languages: German writes numbers below a million as one word and
// larger groups with spaces ("eine Million zweihunderttausend"); French
// hyphenates 17-99 and uses "et" before 1 in 21..61 and 71 ("vingt et un",
// "soixante et onze"); Italian joins everything up to the millions into one
// word with the usual elisions ("ventuno", "centottanta", "ventitré").

const wordsLangMax = 1_000_000_000_000

// wordLang is one language's spellout.
type wordLang struct {
	cardinal func(n int64) string
	// ordinal renders n as an ordinal; variant is the text between the
	// parentheses of the picture's "o(...)" modifier ("" when absent): an
	// ending such as "-er" / "-a", or a CLDR rule name such as
	// "%spellout-ordinal-feminine".
	ordinal func(n int64, variant string) string
}

// wordLangFor returns the spellout for a language tag ("de", "fr-CA", ...), or
// nil when that language is not implemented.
func wordLangFor(tag string) *wordLang {
	base, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(tag)), "-")
	base, _, _ = strings.Cut(base, "_")
	switch base {
	case "de":
		return &wordLang{deCardinal, deOrdinal}
	case "fr":
		return &wordLang{frCardinal, frOrdinal}
	case "it":
		return &wordLang{itCardinal, itOrdinal}
	}
	return nil
}

// numberWordsLang is numberWords for a requested language: the three languages
// above are spelled natively, anything else (and numbers beyond their range)
// takes the English path.
func numberWordsLang(n int64, token string, ordinal bool, lang, variant string) string {
	wl := wordLangFor(lang)
	abs := n
	if abs < 0 {
		abs = -abs
	}
	if wl == nil || abs >= wordsLangMax {
		return numberWords(n, token, ordinal)
	}
	var w string
	if ordinal {
		w = wl.ordinal(abs, variant)
	} else {
		w = wl.cardinal(abs)
	}
	switch token {
	case "W":
		w = upperCaseFull(w) // full case mapping: "dreißig" → "DREISSIG"
	case "Ww":
		w = titleCaseWords(w)
	}
	if n < 0 {
		return "-" + w
	}
	return w
}

// SpellNumber spells n out in words in a natively implemented language other
// than English (German, French, Italian), for xsl:number's word tokens. token
// is "w", "W" or "Ww"; variant is the ordinal variant ("-er",
// "%spellout-ordinal-feminine", or "" for the default). ok is false for any
// other language and for numbers beyond the spellout's range.
func SpellNumber(n int64, token string, ordinal bool, lang, variant string) (string, bool) {
	abs := n
	if abs < 0 {
		abs = -abs
	}
	if wordLangFor(lang) == nil || abs >= wordsLangMax {
		return "", false
	}
	return numberWordsLang(n, token, ordinal, lang, variant), true
}

// ordinalEnding extracts the ending a variant asks for: "-er" -> "er". A CLDR
// rule name or anything else yields ok == false.
func ordinalEnding(variant string) (string, bool) {
	if strings.HasPrefix(variant, "-") && len(variant) > 1 {
		return variant[1:], true
	}
	return "", false
}

// --- German ------------------------------------------------------------------

var deOnes = []string{"null", "eins", "zwei", "drei", "vier", "fünf", "sechs",
	"sieben", "acht", "neun", "zehn", "elf", "zwölf", "dreizehn", "vierzehn",
	"fünfzehn", "sechzehn", "siebzehn", "achtzehn", "neunzehn"}
var deTens = []string{"", "", "zwanzig", "dreißig", "vierzig", "fünfzig",
	"sechzig", "siebzig", "achtzig", "neunzig"}

// deOrdSmall are the ordinal stems (without the adjective ending) of 0..19.
var deOrdSmall = []string{"nullt", "erst", "zweit", "dritt", "viert", "fünft",
	"sechst", "siebt", "acht", "neunt", "zehnt", "elft", "zwölft", "dreizehnt",
	"vierzehnt", "fünfzehnt", "sechzehnt", "siebzehnt", "achtzehnt", "neunzehnt"}

// deUnit spells 1..9 as the prefix of a compound ("ein", not "eins").
func deUnit(n int64) string {
	if n == 1 {
		return "ein"
	}
	return deOnes[n]
}

// deBelow100 spells 1..99. attach: something follows in the same word, so a
// trailing 1 is "ein" rather than "eins".
func deBelow100(n int64, attach bool) string {
	if n < 20 {
		if n == 1 && attach {
			return "ein"
		}
		return deOnes[n]
	}
	t, o := n/10, n%10
	if o == 0 {
		return deTens[t]
	}
	return deUnit(o) + "und" + deTens[t]
}

func deBelow1000(n int64, attach bool) string {
	h, r := n/100, n%100
	s := ""
	if h > 0 {
		s = deUnit(h) + "hundert"
	}
	if r > 0 {
		s += deBelow100(r, attach)
	}
	return s
}

func deCardinal(n int64) string {
	if n == 0 {
		return "null"
	}
	var parts []string
	if b := n / 1_000_000_000; b > 0 {
		if b == 1 {
			parts = append(parts, "eine Milliarde")
		} else {
			parts = append(parts, deBelow1000(b, true)+" Milliarden")
		}
	}
	if m := n % 1_000_000_000 / 1_000_000; m > 0 {
		if m == 1 {
			parts = append(parts, "eine Million")
		} else {
			parts = append(parts, deBelow1000(m, true)+" Millionen")
		}
	}
	t, r := n%1_000_000/1000, n%1000
	tail := ""
	if t > 0 {
		if t == 1 {
			tail = "eintausend"
		} else {
			tail = deBelow1000(t, true) + "tausend"
		}
	}
	if r > 0 {
		tail += deBelow1000(r, false)
	}
	if tail != "" {
		parts = append(parts, tail)
	}
	return strings.Join(parts, " ")
}

// deOrdinalStem is the ordinal without its adjective ending.
func deOrdinalStem(n int64) string {
	for _, sc := range []struct {
		v    int64
		word string
	}{{1_000_000_000, "milliardst"}, {1_000_000, "millionst"}} {
		if n < sc.v {
			continue
		}
		if rem := n % sc.v; rem != 0 {
			return deCardinal(n-rem) + " " + deOrdinalStem(rem)
		}
		if q := n / sc.v; q != 1 {
			return deBelow1000(q, true) + sc.word
		}
		return sc.word
	}
	if n < 20 {
		return deOrdSmall[n]
	}
	if r := n % 100; r >= 1 && r < 20 {
		return deCardinal(n-r) + deOrdSmall[r]
	}
	return deCardinal(n) + "st"
}

func deOrdinal(n int64, variant string) string {
	ending := "e"
	if e, ok := ordinalEnding(variant); ok {
		ending = e
	}
	return deOrdinalStem(n) + ending
}

// --- French ------------------------------------------------------------------

var frOnes = []string{"zéro", "un", "deux", "trois", "quatre", "cinq", "six",
	"sept", "huit", "neuf", "dix", "onze", "douze", "treize", "quatorze",
	"quinze", "seize", "dix-sept", "dix-huit", "dix-neuf"}
var frTens = []string{"", "", "vingt", "trente", "quarante", "cinquante", "soixante"}

// frBelow100 spells 1..99. final: nothing that would keep the plural off
// follows, so "quatre-vingts" takes its s.
func frBelow100(n int64, final bool) string {
	switch {
	case n < 20:
		return frOnes[n]
	case n < 70:
		t, o := n/10, n%10
		switch o {
		case 0:
			return frTens[t]
		case 1:
			return frTens[t] + " et un"
		}
		return frTens[t] + "-" + frOnes[o]
	case n < 80:
		if n == 71 {
			return "soixante et onze"
		}
		return "soixante-" + frOnes[n-60]
	}
	if n == 80 {
		if final {
			return "quatre-vingts"
		}
		return "quatre-vingt"
	}
	return "quatre-vingt-" + frOnes[n-80]
}

func frBelow1000(n int64, final bool) string {
	h, r := n/100, n%100
	if h == 0 {
		return frBelow100(r, final)
	}
	head := "cent"
	if h > 1 {
		head = frOnes[h] + " cent"
		if r == 0 && final {
			head += "s"
		}
	}
	if r == 0 {
		return head
	}
	return head + " " + frBelow100(r, final)
}

func frCardinal(n int64) string {
	if n == 0 {
		return "zéro"
	}
	var parts []string
	if b := n / 1_000_000_000; b > 0 {
		w := frBelow1000(b, true) + " milliard"
		if b > 1 {
			w += "s"
		}
		parts = append(parts, w)
	}
	if m := n % 1_000_000_000 / 1_000_000; m > 0 {
		w := frBelow1000(m, true) + " million"
		if m > 1 {
			w += "s"
		}
		parts = append(parts, w)
	}
	if t := n % 1_000_000 / 1000; t > 0 {
		if t == 1 {
			parts = append(parts, "mille")
		} else {
			parts = append(parts, frBelow1000(t, false)+" mille")
		}
	}
	if r := n % 1000; r > 0 {
		parts = append(parts, frBelow1000(r, true))
	}
	return strings.Join(parts, " ")
}

func frOrdinal(n int64, variant string) string {
	if n == 1 {
		if e, ok := ordinalEnding(variant); (ok && (e == "e" || e == "re" || e == "ère" || e == "ere")) ||
			strings.HasSuffix(variant, "feminine") {
			return "première"
		}
		return "premier"
	}
	card := frCardinal(n)
	cut := strings.LastIndexAny(card, " -") + 1
	head, last := card[:cut], card[cut:]
	switch last {
	case "vingts", "cents", "millions", "milliards":
		last = strings.TrimSuffix(last, "s")
	}
	switch {
	case last == "mille":
		last = "mill"
	case last == "cinq":
		last = "cinqu"
	case last == "neuf":
		last = "neuv"
	case strings.HasSuffix(last, "e"):
		last = strings.TrimSuffix(last, "e")
	}
	return head + last + "ième"
}

// --- Italian -----------------------------------------------------------------

var itOnes = []string{"zero", "uno", "due", "tre", "quattro", "cinque", "sei",
	"sette", "otto", "nove", "dieci", "undici", "dodici", "tredici",
	"quattordici", "quindici", "sedici", "diciassette", "diciotto", "diciannove"}
var itTens = []string{"", "", "venti", "trenta", "quaranta", "cinquanta",
	"sessanta", "settanta", "ottanta", "novanta"}

// itBelow100 spells 1..99; final: a trailing "tre" in a compound takes its
// accent (ventitré) unless more of the number follows in the same word.
func itBelow100(n int64, final bool) string {
	if n < 20 {
		return itOnes[n]
	}
	t, o := n/10, n%10
	base := itTens[t]
	switch o {
	case 0:
		return base
	case 1, 8:
		base = base[:len(base)-1] // venti+uno -> ventuno, trenta+otto -> trentotto
	case 3:
		if final {
			return base + "tré"
		}
	}
	return base + itOnes[o]
}

func itBelow1000(n int64, final bool) string {
	h, r := n/100, n%100
	if h == 0 {
		return itBelow100(r, final)
	}
	head := "cento"
	if h > 1 {
		head = itOnes[h] + "cento"
	}
	if r == 0 {
		return head
	}
	rest := itBelow100(r, final)
	if strings.HasPrefix(rest, "o") { // cento+ottanta -> centottanta
		head = head[:len(head)-1]
	}
	return head + rest
}

func itCardinal(n int64) string {
	if n == 0 {
		return "zero"
	}
	var parts []string
	if b := n / 1_000_000_000; b > 0 {
		if b == 1 {
			parts = append(parts, "un miliardo")
		} else {
			parts = append(parts, itBelow1000(b, true)+" miliardi")
		}
	}
	if m := n % 1_000_000_000 / 1_000_000; m > 0 {
		if m == 1 {
			parts = append(parts, "un milione")
		} else {
			parts = append(parts, itBelow1000(m, true)+" milioni")
		}
	}
	t, r := n%1_000_000/1000, n%1000
	tail := ""
	if t > 0 {
		if t == 1 {
			tail = "mille"
		} else {
			tail = itBelow1000(t, false) + "mila"
		}
	}
	if r > 0 {
		tail += itBelow1000(r, true)
	}
	if tail != "" {
		parts = append(parts, tail)
	}
	return strings.Join(parts, " ")
}

// itOrdSmall are the masculine ordinal stems (without -o) of 1..10.
var itOrdSmall = []string{"", "prim", "second", "terz", "quart", "quint", "sest",
	"settim", "ottav", "non", "decim"}

func itOrdinal(n int64, variant string) string {
	ending := "o"
	if e, ok := ordinalEnding(variant); ok {
		ending = e
	} else if strings.HasSuffix(variant, "feminine") {
		ending = "a"
	}
	if n >= 1 && n <= 10 {
		return itOrdSmall[n] + ending
	}
	card := itCardinal(n)
	if n == 0 {
		return card
	}
	cut := strings.LastIndex(card, " ") + 1
	head, last := card[:cut], card[cut:]
	switch {
	case strings.HasSuffix(last, "mila"):
		last = strings.TrimSuffix(last, "mila") + "mill"
	case strings.HasSuffix(last, "mille"):
		last = strings.TrimSuffix(last, "mille") + "mill"
	case strings.HasSuffix(last, "milione"), strings.HasSuffix(last, "milioni"):
		last = last[:len(last)-1]
	case strings.HasSuffix(last, "miliardo"), strings.HasSuffix(last, "miliardi"):
		last = last[:len(last)-1]
	case strings.HasSuffix(last, "tré"):
		last = strings.TrimSuffix(last, "é") + "e"
	case strings.HasSuffix(last, "tre"), strings.HasSuffix(last, "sei"):
		// keep the vowel: ventitreesimo, ventiseiesimo
	default:
		last = last[:len(last)-1]
	}
	return head + last + "esim" + ending
}
