package xpath

import "testing"

// spStrm classifies expr in sc and names the (posture, sweep) pair.
func spStrm(t *testing.T, expr string, sc StreamContext) string {
	t.Helper()
	p, err := Parse(expr)
	if err != nil {
		t.Fatalf("parse %q: %v", expr, err)
	}
	return strmName(p.Streamability(sc))
}

type spStrmCase struct{ expr, want string }

func spRunStrm(t *testing.T, sc StreamContext, cases []spStrmCase) {
	t.Helper()
	for _, c := range cases {
		if got := spStrm(t, c.expr, sc); got != c.want {
			t.Errorf("%s: got %s, want %s", c.expr, got, c.want)
		}
	}
}

const (
	spGM = "grounded/motionless"
	spGC = "grounded/consuming"
	spSM = "striding/motionless"
	spSC = "striding/consuming"
	spCM = "crawling/motionless"
	spCC = "crawling/consuming"
	spLM = "climbing/motionless"
	spLC = "climbing/consuming"
	spRF = "roaming/free-ranging"
)

// TestSpGeneralRules drives §19.8.1's general streamability rules directly,
// one clause at a time.
func TestSpGeneralRules(t *testing.T) {
	op := func(p Posture, s Sweep, u StreamUsage) StreamOperand { return StreamOperand{P: p, S: s, U: u} }
	cases := []struct {
		name string
		ops  []StreamOperand
		want string
	}{
		{"no operands", nil, spGM},
		{"free-ranging operand", []StreamOperand{op(PostureGrounded, SweepFreeRanging, UsageInspection)}, spRF},
		{"roaming operand", []StreamOperand{op(PostureRoaming, SweepMotionless, UsageInspection)}, spRF},
		{"grounded keeps its sweep", []StreamOperand{op(PostureGrounded, SweepConsuming, UsageNavigation)}, spGC},
		{"climbing absorption", []StreamOperand{op(PostureClimbing, SweepMotionless, UsageAbsorption)}, spRF},
		{"climbing absorption of childless", []StreamOperand{{P: PostureClimbing, S: SweepMotionless, U: UsageAbsorption, Childless: true}}, spGM},
		{"striding absorption", []StreamOperand{op(PostureStriding, SweepMotionless, UsageAbsorption)}, spGC},
		{"crawling absorption", []StreamOperand{op(PostureCrawling, SweepConsuming, UsageAbsorption)}, spGC},
		{"striding inspection consuming", []StreamOperand{op(PostureStriding, SweepConsuming, UsageInspection)}, spGC},
		{"striding inspection motionless", []StreamOperand{op(PostureStriding, SweepMotionless, UsageInspection)}, spGM},
		{"navigation of a streamed node", []StreamOperand{op(PostureStriding, SweepMotionless, UsageNavigation)}, spRF},
		{"transmission keeps posture", []StreamOperand{op(PostureCrawling, SweepConsuming, UsageTransmission)}, spCC},
		{"transmission motionless", []StreamOperand{op(PostureClimbing, SweepMotionless, UsageTransmission)}, spLM},
		{"higher-order consuming", []StreamOperand{{P: PostureStriding, S: SweepConsuming, U: UsageTransmission, HigherOrder: true}}, spRF},
		{"higher-order motionless", []StreamOperand{{P: PostureStriding, S: SweepMotionless, U: UsageInspection, HigherOrder: true}}, spGM},
		{"two consuming", []StreamOperand{
			op(PostureStriding, SweepConsuming, UsageAbsorption),
			op(PostureStriding, SweepConsuming, UsageAbsorption)}, spRF},
		{"two motionless transmissions, one posture", []StreamOperand{
			op(PostureStriding, SweepMotionless, UsageTransmission),
			op(PostureStriding, SweepMotionless, UsageTransmission)}, spSM},
		{"two motionless transmissions, two postures", []StreamOperand{
			op(PostureStriding, SweepMotionless, UsageTransmission),
			op(PostureClimbing, SweepMotionless, UsageTransmission)}, spRF},
		{"choice group", []StreamOperand{
			{P: PostureStriding, S: SweepConsuming, U: UsageTransmission, Choice: 1},
			{P: PostureStriding, S: SweepMotionless, U: UsageTransmission, Choice: 1}}, spSC},
		{"choice group with a grounded member", []StreamOperand{
			{P: PostureCrawling, S: SweepConsuming, U: UsageTransmission, Choice: 1},
			{P: PostureGrounded, S: SweepConsuming, U: UsageAbsorption, Choice: 1}}, spCC},
		{"choice group, different streamed postures", []StreamOperand{
			{P: PostureStriding, S: SweepConsuming, U: UsageTransmission, Choice: 1},
			{P: PostureClimbing, S: SweepMotionless, U: UsageTransmission, Choice: 1}}, spRF},
		{"two choice groups", []StreamOperand{
			{P: PostureStriding, S: SweepConsuming, U: UsageTransmission, Choice: 1},
			{P: PostureStriding, S: SweepConsuming, U: UsageTransmission, Choice: 2}}, spRF},
		{"one consuming among motionless", []StreamOperand{
			op(PostureGrounded, SweepMotionless, UsageAbsorption),
			op(PostureStriding, SweepConsuming, UsageTransmission),
			op(PostureStriding, SweepMotionless, UsageInspection)}, spSC},
	}
	for _, c := range cases {
		got := strmName(StreamGeneralRules(c.ops))
		if got != c.want {
			t.Errorf("%s: got %s, want %s", c.name, got, c.want)
		}
		var def bool
		got2 := strmName(StreamGeneralRulesIn(c.ops, &StreamContext{Definite: &def}))
		if got2 != got {
			t.Errorf("%s: StreamGeneralRulesIn %s differs from StreamGeneralRules %s", c.name, got2, got)
		}
	}
	if !Streamable(PostureCrawling, SweepConsuming) || Streamable(PostureRoaming, SweepMotionless) ||
		Streamable(PostureGrounded, SweepFreeRanging) {
		t.Error("Streamable: §19.7 roaming or free-ranging means not streamable")
	}
}

// TestSpTypeDeterminedUsage: §19.4 type-determined usage.
func TestSpTypeDeterminedUsage(t *testing.T) {
	if TypeDeterminedUsage(true, false) != UsageInspection {
		t.Error("function(*) → inspection")
	}
	if TypeDeterminedUsage(false, true) != UsageAbsorption {
		t.Error("atomic → absorption")
	}
	if TypeDeterminedUsage(false, false) != UsageNavigation {
		t.Error("other → navigation")
	}
}

// TestSpOperatorProformas walks §19.8.2's worked examples and §19.8.8's
// proforma table (A = absorption, I = inspection, T = transmission, N =
// navigation), striding element context.
func TestSpOperatorProformas(t *testing.T) {
	spRunStrm(t, StreamContext{ContextPosture: PostureStriding}, []spStrmCase{
		// §19.8.2.
		{`2 + 2`, spGM},
		{`price * 2`, spGC},
		{`price - discount`, spRF},
		{`price * @discount`, spGC},
		{`count(a/b/c)`, spGC},
		{`a//c`, spCC},
		// A + A, -A, A to A, A || A, A = A, A eq A, cast/castable.
		{`-price`, spGC},
		{`+@a`, spGM},
		{`1 to 3`, spGM},
		{`1 to count(*)`, spGC},
		{`'a' || b`, spGC},
		{`@a || @b`, spGM},
		{`a || b`, spRF},
		{`@a = 'x'`, spGM},
		{`a eq 'x'`, spGC},
		{`a idiv @b`, spGC},
		{`a cast as xs:integer`, spGC},
		{`@a castable as xs:integer`, spGM},
		// I and I, I or I.
		{`@a and @b`, spGM},
		{`exists(a) or @b`, spGC},
		{`a or b`, spRF},
		// I is I (Note 1 of §19.8.8).
		{`. is ..`, spGM},
		{`. << child::x`, spGC},
		// T, T.
		{`(@a, @b)`, spSM},
		{`(1, a)`, spSC},
		{`(a, @b)`, spRF},
		{`()`, spGM},
		{`'s'`, spGM},
		// §19.8.8.4.
		{`a | b`, spCC},
		{`@a | @b`, spCM},
		{`. | following-sibling::*`, spRF},
		{`.. | *`, spRF},
		{`(1, 2)[1] | a`, spSC},
		// §19.8.8.5: inspection, except a document-node(element()) test.
		{`a instance of element()*`, spGC},
		{`. instance of element(E)*`, spGM},
		{`. instance of document-node(element(a))`, spGC},
		// §19.8.8.6.
		{`. treat as document-node(element(a))`, spRF},
		{`a treat as element()*`, spSC},
		// §19.8.8.3: a choice operand group of transmissions.
		{`if (@a) then b else c`, spSC},
		{`if (@x) then @name else 'n'`, spSM},
		{`if (a) then b else c`, spRF},
		{`if (@x) then b else ..`, spRF},
		// §19.8.8.1.
		{`for $i in 1 to 3 return salary`, spRF},
		{`for $i in 1 to 3 return name(ancestor::x[$i])`, spGM},
		{`for $e in copy-of(emp) return $e/salary`, spGC},
		{`for $x in child::section return $x/para`, spRF},
		// let $var := N return T.
		{`let $x := 1 return @a`, spSM},
		{`let $x := @a return 1`, spRF},
		{`let $x := count(a) return $x + 1`, spGC},
		// §19.8.8.2.
		{`some $i in 1 to 3 satisfies author[$i] eq "Kay"`, spRF},
		{`some $i in 1 to 3 satisfies @grade = $i`, spGM},
		{`every $e in emp/salary/number(.) satisfies $e gt 10000`, spGC},
		// §19.8.8.7: posture of the right operand, sweep the wider.
		{`a ! @b`, spSC},
		{`@a ! string(.)`, spGM},
		{`a ! (. + 1)`, spGC},
		{`(1, 2) ! .`, spGM},
		// §19.8.8.16.
		{`function($x) { $x + 1 }`, spGM},
		// §19.8.8.15.
		{`name#0`, spRF},
		{`upper-case#1`, spGM},
		{`position#0`, spRF},
		// SquareArrayConstructor [N, N].
		{`[1, 'x']`, spGM},
		{`[@a]`, spRF},
		{`array{@a}`, spRF},
		{`array{}`, spGM},
		// §19.8.8.17 / §19.8.4.24: key absorption, value navigation.
		{`map{'k': 1}`, spGM},
		{`map{'k': @a}`, spRF},
		{`map{}`, spGM},
		// §19.8.8.18.
		{`map{'a': 1}?a`, spGM},
		{`map{'a': 1}?*`, spGM},
		{`?k`, spGM},
		{`map{'a': 1}?(@a)`, spGM},
		{`map{'a': 1}?(a)`, spGC},
		// §19.8.8.11: no signature → navigation; a map → absorption.
		{`$f(@a)`, spRF},
		{`$f(1)`, spGM},
		{`map{'a': 1}(@a)`, spGM},
		{`map{'a': 1}(a)`, spGC},
		{`[1](1)`, spGM},
		// §19.8.8.14 ArrowExpr: X => F(Y) is F(X, Y).
		{`@a => string()`, spGM},
		{`a => string()`, spGC},
		{`a => count() => string()`, spGC},
		{`@a => $f()`, spRF},
		{`1 => $f()`, spGM},
	})
}

// TestSpAxisSteps checks §19.8.8.9's table and its special clauses, and
// §19.8.8.8's path rules.
func TestSpAxisSteps(t *testing.T) {
	striding := StreamContext{ContextPosture: PostureStriding}
	spRunStrm(t, striding, []spStrmCase{
		{`self::x`, spSM},
		{`@a`, spSM},
		{`namespace::*`, spSM},
		{`..`, spLM},
		{`ancestor::x`, spLM},
		{`ancestor-or-self::x`, spLM},
		{`child::x`, spSC},
		{`descendant::x`, spCC},
		{`descendant-or-self::node()`, spCC},
		{`descendant::text()`, spSC},
		{`descendant::comment()`, spSC},
		{`descendant::processing-instruction()`, spSC},
		{`following-sibling::x`, spRF},
		{`preceding-sibling::x`, spRF},
		{`following::x`, spRF},
		// The numeric, focus-free predicate clause.
		{`descendant::section[1]`, spSC},
		{`descendant::section[$i + 1]`, spSC},
		{`descendant::section[count($x)]`, spSC},
		{`descendant::section[-1]`, spSC},
		{`descendant::section[1 to 2]`, spSC},
		{`descendant::section[position()]`, spCC},
		{`descendant::section[b]`, spRF},
		{`descendant::section[@a]`, spCC},
		// Predicates must be motionless.
		{`*[@a = 1]`, spSC},
		{`*[b]`, spRF},
		{`*[1]`, spSC},
		{`@a[. = 'x']`, spSM},
		{`text()[. = 'x']`, spSC},
		// §19.8.8.8: climbing, then downwards.
		{`ancestor::x/self::y`, spLM},
		{`ancestor::x/@y`, spSM},
		{`ancestor::x/namespace::*`, spSM},
		{`../b`, spRF},
		{`ancestor::x/descendant::y`, spRF},
		// Crawling context for the next step.
		{`descendant::x/@a`, spSC},
		{`descendant::x/self::y`, spCC},
		{`descendant::x/self::text()`, spSC},
		{`descendant::x/..`, spLC},
		{`descendant::x/following::y`, spRF},
		// §19.8.8.8 worked notes.
		{`a/@code`, spSC},
		{`./@code`, spSM},
		{`./a`, spSC},
		{`a/following::b`, spRF},
		{`./.`, spSM},
		{`text()/..`, spLC},
		// Scanning expressions.
		{`self::node()//section/head`, spCC},
		{`descendant::section/head`, spCC},
		{`section//head/text()`, spSC},
		{`section//head[@a]`, spCC},
		{`section//head[b]`, spRF},
		{`section//head[position() = 1]`, spRF},
		{`section//head/following::p`, spRF},
		{`(a|b)//c`, spCC},
		// A RootedPath can never be a scanning expression.
		{`$v//section/head`, spGM},
		// Postfix steps.
		{`a/string()`, spGC},
		{`a/(@b)`, spSC},
		{`a/(b | c)`, spCC},
		{`a/name()`, spGC},
		{`copy-of(.)//a/following-sibling::*`, spGC},
		// §19.8.8.10 filter expressions.
		{`(descendant::x)[1]`, spSC},
		{`(descendant::x)[$i + 1]`, spSC},
		{`(descendant::x)[@a]`, spCC},
		{`(descendant::x)[b]`, spRF},
		{`(a)[position() = 1]`, spSC},
		{`$s[child::*]`, spGM},
		{`(1, 2, 3)[. > 1]`, spGM},
		// Grounded start.
		{`copy-of(.)/a/b`, spGC},
		{`(1, 2)/string()`, spGM},
	})
	// Context postures other than striding.
	spRunStrm(t, StreamContext{ContextPosture: PostureGrounded}, []spStrmCase{
		{`a/b`, spGM},
		{`following::x`, spGM},
		{`/a`, spGM},
		{`last()`, spGM},
		{`root()`, spGM},
		{`name#0`, spGM},
	})
	spRunStrm(t, StreamContext{ContextPosture: PostureRoaming}, []spStrmCase{
		{`a`, spRF},
		{`last()`, spRF},
	})
	spRunStrm(t, StreamContext{ContextPosture: PostureClimbing}, []spStrmCase{
		{`*`, spRF},
		{`self::x`, spLM},
		{`@a`, spSM},
		{`..`, spLM},
		{`last()`, spGM},
		{`/a`, spRF},
	})
	spRunStrm(t, StreamContext{ContextPosture: PostureCrawling}, []spStrmCase{
		{`*`, spRF},
		{`descendant::x`, spRF},
		{`self::x`, spCM},
		{`self::text()`, spSM},
		{`@a`, spSM},
		{`ancestor::x`, spLM},
		{`last()`, spRF},
		{`(.)[1]`, spSM},
	})
	// §19.8.9.18: a leading "/" over a document node in striding posture.
	doc := StreamContext{ContextPosture: PostureStriding, ContextIsDocument: true}
	spRunStrm(t, doc, []spStrmCase{
		{`/`, spSM},
		{`/a`, spSC},
		{`//section/head`, spCC},
		{`(//x)[3]`, spSC},
		{`(//x)[1 to 5]`, spSC},
		{`root()`, spSM},
		{`(/a|/b)//c`, spCC},
	})
}

// TestSpBuiltinFunctions covers §19.8.9's per-function rules.
func TestSpBuiltinFunctions(t *testing.T) {
	striding := StreamContext{ContextPosture: PostureStriding}
	spRunStrm(t, striding, []spStrmCase{
		// Inspection and absorption defaults of the zero-arity forms.
		{`name()`, spGM},
		{`local-name(..)`, spGM},
		{`string()`, spGC},
		{`data()`, spGC},
		{`string(@a)`, spGM},
		{`normalize-space()`, spGC},
		{`string-length()`, spGC},
		{`generate-id()`, spGM},
		{`has-children()`, spGM},
		{`concat(@a, @b, @c)`, spGM},
		{`concat(a, b)`, spRF},
		{`xs:integer(a)`, spGC},
		{`xs:integer(@a)`, spGM},
		{`math:sqrt(a)`, spGC},
		{`true()`, spGM},
		{`count(a)`, spGC},
		{`boolean(@a)`, spGM},
		{`id('x')`, spRF},
		{`key('k', 'v')`, spRF},
		// §19.8.9.14/.16.
		{`last()`, spRF},
		{`position()`, spGM},
		{`current-grouping-key()`, spGM},
		// §19.8.1's maximum-cardinality-one clause.
		{`head(descendant::x)`, spSC},
		{`exactly-one(descendant::x)`, spSC},
		{`zero-or-one(descendant::x)`, spSC},
		{`tail(descendant::x)`, spCC},
		{`head(@a)`, spSM},
		{`head(a)`, spSC},
		{`subsequence(a, 1, 2)`, spSC},
		{`insert-before(a, 1, 'x')`, spSC},
		{`trace(a, 'x')`, spSC},
		{`unordered(descendant::x)`, spCC},
		// §19.8.9.15/.13/.17.
		{`outermost(descendant::para)`, spSC},
		{`outermost(@a)`, spSM},
		{`outermost(descendant::x/following::y)`, spRF},
		{`innermost(descendant::para)`, spRF},
		{`innermost(copy-of(.)//x)`, spGC},
		{`reverse(*)`, spRF},
		{`reverse((1, 2))`, spGM},
		// Higher-order functions: the first argument is navigation.
		{`filter(*, function($x) { true() })`, spRF},
		{`for-each((1, 2), function($x) { $x })`, spGM},
		{`fold-left((1, 2), 0, function($a, $b) { $a + $b })`, spGM},
		// Maps and arrays.
		{`map:get(map{}, @a)`, spGM},
		{`map:get(map{}, a)`, spGC},
		{`map:size(map{})`, spGM},
		{`array:size([1])`, spGM},
		// §19.8.9.18.
		{`root()`, spLM},
		{`root(.)`, spLM},
		{`root(a)`, spLC},
		{`root(copy-of(.))`, spGC},
		// §19.8.9.1/.2.
		{`accumulator-before('a')`, spGM},
		{`accumulator-after('a')`, spGC},
		// §19.8.9.4: no containing xsl:for-each-group.
		{`current-group()`, spRF},
		// fn:current() defaults to the striding focus of the instruction.
		{`current()`, spSM},
		// Unknown and extension functions fail closed.
		{`my:f(a)`, spRF},
		{`unknown-function(1)`, spRF},
		{`xs:integer(1, 2)`, spRF},
		// §19.8.8.14 partial applications.
		{`function-lookup(?, 0)`, spRF},
		{`substring(?, 1)`, spGM},
		{`substring(?, string(a))`, spGC},
	})
	grounded := StreamContext{ContextPosture: PostureGrounded}
	spRunStrm(t, grounded, []spStrmCase{
		{`function-lookup(?, 0)`, spGM},
		{`accumulator-after('a')`, spGM},
	})
	// Accumulator-after per §19.8.9.1's later clauses.
	cases := []struct {
		sc   StreamContext
		want string
	}{
		{StreamContext{ContextPosture: PostureStriding, ContextChildless: true}, spGM},
		{StreamContext{ContextPosture: PostureStriding, AccPhase: AccPhaseStart}, "grounded/free-ranging"},
		{StreamContext{ContextPosture: PostureStriding, AccPhase: AccPhaseEnd}, spGM},
		{StreamContext{ContextPosture: PostureStriding, AccAfterConsumed: true}, spGM},
		{StreamContext{ContextPosture: PostureStriding, FocusReset: true}, "grounded/free-ranging"},
	}
	for i, c := range cases {
		if got := spStrm(t, `accumulator-after('a')`, c.sc); got != c.want {
			t.Errorf("accumulator-after case %d: got %s, want %s", i, got, c.want)
		}
	}
	// Inside a predicate the focus-setting container is not the instruction's.
	if got := spStrm(t, `(a)[accumulator-after('x') = 1]`, StreamContext{ContextPosture: PostureStriding}); got == spGM {
		t.Errorf("accumulator-after in a predicate: got %s, want not motionless", got)
	}
	// The accumulator name must be motionless (both functions).
	for _, e := range []string{`accumulator-before(string(a))`, `accumulator-after(string(a))`} {
		p, _ := Parse(e)
		if Streamable(p.Streamability(StreamContext{ContextPosture: PostureStriding})) {
			t.Errorf("%s: want non-streamable", e)
		}
	}
	// current-group(): the population's posture (§19.8.9.4).
	spRunStrm(t, StreamContext{ContextPosture: PostureStriding, HasGroup: true, GroupPosture: PostureGrounded}, []spStrmCase{
		{`current-group()`, spGM},
		{`count(current-group())`, spGM},
	})
	spRunStrm(t, StreamContext{ContextPosture: PostureStriding, HasGroup: true, GroupPosture: PostureStriding}, []spStrmCase{
		{`current-group()`, spSC},
	})
	spRunStrm(t, StreamContext{ContextPosture: PostureStriding, HasCurrentPosture: true, CurrentPosture: PostureGrounded}, []spStrmCase{
		{`current()`, spGM},
		{`(20 to 21)[. gt current()]`, spGM},
	})
}

// spFuncs is a stylesheet-function resolver for §19.8.5 call-site rules.
type spFuncs map[string]StreamFuncSig

func (f spFuncs) StreamFunc(prefix, local string, arity int) (StreamFuncSig, bool) {
	s, ok := f[prefix+":"+local]
	return s, ok
}

func spSig(cat StreamFuncCategory, nodes bool, usage ...StreamUsage) StreamFuncSig {
	return StreamFuncSig{Category: cat, ParamUsage: usage, FirstAdjust: UsageTransmission,
		FirstParamNodes: nodes, Streamable: cat != StreamFnUnclassified}
}

// TestSpStylesheetFunctions checks §19.8.5's call-site rules per category.
func TestSpStylesheetFunctions(t *testing.T) {
	funcs := spFuncs{
		"f:zero":    spSig(StreamFnUnclassified, false),
		"f:min":     spSig(StreamFnUnclassified, false, UsageAbsorption, UsageAbsorption),
		"f:nodes":   spSig(StreamFnUnclassified, true, UsageNavigation),
		"f:absorb":  spSig(StreamFnAbsorbing, true, UsageNavigation, UsageNavigation),
		"f:inspect": spSig(StreamFnInspection, true, UsageNavigation),
		"f:filter":  spSig(StreamFnFilter, true, UsageNavigation),
		"f:asc":     spSig(StreamFnAscent, true, UsageNavigation, UsageAbsorption),
		"f:shallow": spSig(StreamFnShallowDescent, true, UsageNavigation, UsageAbsorption),
		"f:deep":    spSig(StreamFnDeepDescent, true, UsageNavigation, UsageAbsorption),
		"f:attrs":   spSig(StreamFnShallowDescent, false, UsageNavigation),
		"f:bad":     {Category: StreamFuncCategory(99), Streamable: true},
	}
	sc := StreamContext{ContextPosture: PostureStriding, Funcs: funcs}
	spRunStrm(t, sc, []spStrmCase{
		{`f:zero()`, spGM},
		// §19.8.5.1's f:min example: two consuming arguments are not
		// streamable, two attributes are.
		{`f:min(price, discount)`, spRF},
		{`f:min(@a, @b)`, spGM},
		{`f:min(price, 1)`, spGC},
		{`f:nodes(a)`, spRF},
		{`f:nodes(copy-of(a))`, spGC},
		// §19.8.5.2: absorption of the first argument; crawling is out.
		{`f:absorb(a, 1)`, spGC},
		{`f:absorb(descendant::a, 1)`, spRF},
		{`f:absorb(copy-of(a), b)`, spRF},
		{`f:absorb(., copy-of(b))`, spRF},
		{`f:absorb(., 1)`, spGC},
		// §19.8.5.3.
		{`f:inspect(.)`, spGM},
		{`f:inspect(a)`, spGC},
		// §19.8.5.4: the call has its argument's posture and sweep.
		{`f:filter(a)`, spSC},
		{`f:filter(@a)`, spSM},
		// §19.8.5.7.
		{`f:asc(.)`, spGM},
		{`f:asc(a)`, spRF},
		{`f:asc(., 1)`, spGM},
		// §19.8.5.5/.6.
		{`f:shallow(., 1)`, spSC},
		{`f:deep(., 1)`, spCC},
		{`f:shallow(a, 1)`, spSC},
		{`f:shallow(a, string(b))`, spRF},
		{`f:shallow(descendant::a, 1)`, spRF},
		{`f:shallow(., b)`, spSC},
		{`f:shallow(., @b)`, spSC},
		{`f:shallow(copy-of(.), 1)`, spGC},
		{`f:shallow(., ..)`, spRF},
		{`f:attrs(.)`, spSM},
		{`f:shallow(@a, 1)`, spSM},
		// §19.8.8.14 partial applications.
		{`f:absorb(., ?)`, spRF},
		{`f:absorb(?, 1)`, spGM},
		{`f:min(?, @a)`, spGM},
		{`f:min(?, a)`, spGC},
		// An unknown category fails closed.
		{`f:bad(1)`, spRF},
		// Not known to the resolver: an extension function.
		{`g:other(.)`, spRF},
		// Arrow form.
		{`. => f:filter()`, spSM},
	})
	p, _ := Parse(`count(f:absorb(.)) + a => f:shallow(1)`)
	if !p.StreamUsesDeclaredStreamable(funcs) {
		t.Error("StreamUsesDeclaredStreamable: f:absorb is declared-streamable")
	}
	for _, e := range []string{`a => f:shallow(1)`, `f:deep(.)`} {
		p, _ := Parse(e)
		if !p.StreamUsesDeclaredStreamable(funcs) {
			t.Errorf("%s: want StreamUsesDeclaredStreamable", e)
		}
	}
	for _, e := range []string{`f:min(1, 2)`, `a => count()`, `. => $g()`, `count(a)`} {
		p, _ := Parse(e)
		if p.StreamUsesDeclaredStreamable(funcs) {
			t.Errorf("%s: no declared-streamable function is called", e)
		}
	}
	if p.StreamUsesDeclaredStreamable(nil) {
		t.Error("nil resolver knows no functions")
	}
	var nilP *Parsed
	if nilP.StreamUsesDeclaredStreamable(funcs) || nilP.StreamUsesFunction("x") {
		t.Error("nil expression calls nothing")
	}
	if got := strmName(nilP.Streamability(sc)); got != spGM {
		t.Errorf("nil expression: got %s, want %s", got, spGM)
	}
}

// TestSpStreamingParameter covers §19.8.8.12 / §19.8.5's references to a
// streaming parameter and §19.8.8.16 inline functions closing over one.
func TestSpStreamingParameter(t *testing.T) {
	sc := StreamContext{
		ContextPosture:        PostureStriding,
		StreamingParams:       map[string]bool{":n": true},
		StreamingParamPosture: PostureStriding,
		StreamingParamSweep:   SweepMotionless,
	}
	spRunStrm(t, sc, []spStrmCase{
		{`$n`, spSM},
		{`$n/a`, spSC},
		{`$other`, spGM},
		{`function() { $n }`, spRF},
		{`function() { $other }`, spGM},
		{`for $i in 1 to 2 return $n`, spRF},
		// §19.8.8.10's counter-example.
		{`$s[$n = 2]`, spRF},
		// §19.8.8.8 Note: $node//section/head is not a scanning expression.
		{`$n/(descendant::section/head)`, spCC},
	})
	// Ascent: references are climbing.
	sc.StreamingParamPosture = PostureClimbing
	if got := spStrm(t, `$n`, sc); got != spLM {
		t.Errorf("ascent reference: got %s, want %s", got, spLM)
	}
	// A repeatable reference in a higher-order operand stays motionless.
	sc.StreamingParamPosture = PostureStriding
	sc.StreamingParamRepeatable = true
	if got := spStrm(t, `for $i in 1 to 2 return name($n)`, sc); got != spGM {
		t.Errorf("repeatable reference: got %s, want %s", got, spGM)
	}
	// The pattern rule (§19.8.10, third bullet).
	pat, err := ParsePattern(`p[@a = $n]`)
	if err != nil {
		t.Fatal(err)
	}
	if Streamable(pat.Streamability(sc)) {
		t.Error("p[@a = $n] references a streaming parameter: not motionless")
	}
	// A parameter registered without a posture fails closed.
	bad := StreamContext{ContextPosture: PostureStriding, StreamingParams: map[string]bool{":n": true}}
	if got := spStrm(t, `$n`, bad); got != spRF {
		t.Errorf("unregistered posture: got %s, want %s", got, spRF)
	}
	// A variable known to hold a map: §19.8.8.11's absorption note.
	mv := StreamContext{ContextPosture: PostureStriding, MapArrayVars: map[string]bool{":m": true}}
	spRunStrm(t, mv, []spStrmCase{
		{`$m(@type)`, spGM},
		{`$m(a)`, spGC},
		{`$f(@type)`, spRF},
	})
	// InHigherOrder marks the context as inside a repeated operand.
	ho := sc
	ho.StreamingParamRepeatable = false
	ho = ho.InHigherOrder()
	if got := spStrm(t, `$n`, ho); got != spRF {
		t.Errorf("$n in a higher-order operand: got %s, want %s", got, spRF)
	}
}

// TestSpDefinite: the Definite flag is raised by rules the REC states
// unconditionally and left alone where the analysis merely declines.
func TestSpDefinite(t *testing.T) {
	cases := []struct {
		expr     string
		definite bool
	}{
		{`last()`, true},                               // §19.8.9.14
		{`price - discount`, true},                     // two consuming operands
		{`. treat as document-node(element(a))`, true}, // §19.8.8.6
		{`*[b]`, true},                                 // non-motionless predicate
		{`(a)[b]`, true},                               // §19.8.8.10
		{`current-group()`, true},                      // §19.8.9.4
		{`reverse(*)`, true},                           // navigation stated by §19.8.9.17
		{`for $i in 1 to 3 return salary`, true},       // higher-order consuming
		{`string(..)`, true},                           // climbing absorption
		{`$v//section/head`, false},                    // grounded: nothing to report
		{`(1, 2)[1]//section/head`, false},             //
		{`my:f(a)`, false},                             // unknown function
		{`a/following::b`, false},                      // table default
		{`let $x := @a return 1`, false},               // navigation fallback
		{`map{'k': @a}`, true},                         // §19.8.4.24 navigation
	}
	for _, c := range cases {
		p, err := Parse(c.expr)
		if err != nil {
			t.Fatalf("parse %q: %v", c.expr, err)
		}
		var def bool
		p.Streamability(StreamContext{ContextPosture: PostureStriding, Definite: &def})
		if def != c.definite {
			t.Errorf("%s: Definite = %v, want %v", c.expr, def, c.definite)
		}
	}
	// A RootedPath start reports its non-scanning verdict when roaming.
	sp := StreamContext{
		ContextPosture: PostureStriding, StreamingParams: map[string]bool{":n": true},
		StreamingParamPosture: PostureStriding, StreamingParamSweep: SweepMotionless,
	}
	var def bool
	sp.Definite = &def
	p, _ := Parse(`$n//section/head[1]`)
	if got := strmName(p.Streamability(sp)); got != spRF {
		t.Errorf("$n//section/head[1]: got %s, want %s", got, spRF)
	}
	if !def {
		t.Error("$n//section/head[1]: a RootedPath is never a scanning expression (definite)")
	}
	def = false
	p, _ = Parse(`(1 to 2, a)//section/head`)
	p.Streamability(sp)
	var nilCtx *StreamContext
	nilCtx.MarkDefinite() // must not panic
}

// TestSpPatternsMore extends §19.8.10's own examples.
func TestSpPatternsMore(t *testing.T) {
	sc := StreamContext{ContextPosture: PostureStriding}
	motionless := []string{
		`p[@status = $status-codes[1]]`, `p[@code = $status[last()]]`,
		`(p|q)[@a]`, `(p except q)[@a]`, `(p intersect q)`, `p/q[@a]`,
		`text()[. = current()]`, `@a[. = current()]`, `p//q`, `node()`,
		`p[@a][@b]`,
	}
	for _, src := range motionless {
		pat, err := ParsePattern(src)
		if err != nil {
			t.Fatalf("parse pattern %q: %v", src, err)
		}
		if p, s := pat.Streamability(sc); !Streamable(p, s) {
			t.Errorf("pattern %q: got %s, want motionless", src, strmName(p, s))
		}
	}
	notMotionless := []string{
		`$doc//p`, `(p|q)[1]`, `(p|q)[b]`, `p[function-lookup(xs:QName('fn:true'), 0)()]`,
		`p[position()]`, `p[. = current()]`, `key('k', 'v')`, `doc('x')//p`,
		`(p|$v)[@a]`, `p[@a][b]`, `p/q[last()]`,
	}
	for _, src := range notMotionless {
		pat, err := ParsePattern(src)
		if err != nil {
			t.Fatalf("parse pattern %q: %v", src, err)
		}
		if p, s := pat.Streamability(sc); Streamable(p, s) {
			t.Errorf("pattern %q: got %s, want free-ranging", src, strmName(p, s))
		}
	}
	var nilPat *Pattern
	if got := strmName(nilPat.Streamability(sc)); got != spGM {
		t.Errorf("nil pattern: %s", got)
	}
}

// TestSpChildless checks the static "cannot have children" shape test that
// §19.8.1 uses to downgrade absorption to inspection.
func TestSpChildless(t *testing.T) {
	pats := []struct {
		src  string
		want bool
	}{
		{`@a`, true}, {`text() | comment()`, true}, {`a/@b`, true},
		{`processing-instruction()`, true}, {`namespace::*`, true},
		{`a`, false}, {`@a | b`, false}, {`.`, false}, {`/`, false},
		{`(a|@b)[1]`, false}, {`$v`, false}, {`(@a|@b)`, true},
	}
	for _, c := range pats {
		pat, err := ParsePattern(c.src)
		if err != nil {
			t.Fatalf("parse pattern %q: %v", c.src, err)
		}
		if got := pat.StreamChildless(); got != c.want {
			t.Errorf("pattern %q: StreamChildless = %v, want %v", c.src, got, c.want)
		}
	}
	var nilPat *Pattern
	if nilPat.StreamChildless() {
		t.Error("nil pattern: StreamChildless must be false")
	}
	exprs := []struct {
		src  string
		want bool
	}{
		{`@a`, true}, {`1`, true}, {`name()`, true}, {`1 + 2`, true},
		{`'a' || 'b'`, true}, {`1 to 2`, true}, {`@a = 1`, true},
		{`-1`, true}, {`@a cast as xs:string`, true}, {`. instance of node()`, true},
		{`map{}`, true}, {`[]`, true}, {`function() {1}`, true}, {`name#1`, true},
		{`$v`, true}, {`a!@b`, true}, {`a!b`, false}, {`@a except @b`, true},
		{`a except @b`, false}, {`@a | text()`, true}, {`@a | b`, false},
		{`if (a) then @b else 'x'`, true}, {`if (a) then b else 'x'`, false},
		{`(@a, 1)`, true}, {`(@a, b)`, false}, {`(@a)[1]`, true},
		{`a`, false}, {`.`, false}, {`copy-of(.)`, false}, {`head(@a)`, false},
		{`current()`, false}, {`a/string()`, false}, {`/`, false}, {`()`, false},
		{`$f(1)`, false},
	}
	for _, c := range exprs {
		p, err := Parse(c.src)
		if err != nil {
			t.Fatalf("parse %q: %v", c.src, err)
		}
		if got := p.StreamChildless(); got != c.want {
			t.Errorf("%q: StreamChildless = %v, want %v", c.src, got, c.want)
		}
	}
	p, _ := Parse(`.`)
	if !p.StreamChildlessIn(StreamContext{ContextChildless: true}) || p.StreamChildlessIn(StreamContext{}) {
		t.Error("StreamChildlessIn(.) follows ContextChildless")
	}
	p, _ = Parse(`current()`)
	if !p.StreamChildlessIn(StreamContext{CurrentChildless: true}) {
		t.Error("StreamChildlessIn(current()) follows CurrentChildless")
	}
	p, _ = Parse(`$n`)
	if p.StreamChildlessIn(StreamContext{StreamingParams: map[string]bool{":n": true}}) {
		t.Error("a streaming parameter may hold elements")
	}
	var nilP *Parsed
	if !nilP.StreamChildless() || !nilP.StreamChildlessIn(StreamContext{}) {
		t.Error("an absent operand is childless")
	}
}

// TestSpUsesFunction covers the call-walk used for §19.8.4's grouping-function
// boundary rule.
func TestSpUsesFunction(t *testing.T) {
	yes := []string{
		`current-group()`, `count(current-group())`, `current-group#0`,
		`a[current-group()]`, `(1)[current-group()]`, `1 + current-group()`,
		`-current-group()`, `a | current-group()`, `a except current-group()`,
		`current-group() = 1`, `'a' || current-group()`, `(1, current-group())`,
		`1 to current-group()`, `for $x in 1 return current-group()`,
		`let $x := 1 return current-group()`, `some $x in 1 satisfies current-group()`,
		`if (1) then current-group() else 2`, `map{1: current-group()}`,
		`[current-group()]`, `function() { current-group() }`, `map{}?(current-group())`,
		`$f(current-group())`, `current-group() instance of item()`,
		`current-group() treat as item()*`, `current-group() cast as xs:string`,
		`current-group() => count()`, `1 => $f(current-group())`, `a ! current-group()`,
		`a/current-group()`,
	}
	for _, e := range yes {
		p, err := Parse(e)
		if err != nil {
			t.Fatalf("parse %q: %v", e, err)
		}
		if !p.StreamUsesFunction("current-group") {
			t.Errorf("%s: current-group() not found", e)
		}
	}
	for _, e := range []string{`count(a)`, `current-grouping-key()`, `my:current-group()`} {
		p, _ := Parse(e)
		if p.StreamUsesFunction("current-group") {
			t.Errorf("%s: current-group() falsely found", e)
		}
	}
}
