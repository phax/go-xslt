package xpath

import "testing"

// sqAttendees is the $at document of the F&O 3.1 §14.2.3 fn:deep-equal
// examples, plus a few siblings for the node-kind cases.
const sqAttendees = `<doc><attendees><name last='Parker' first='Peter'/><name last='Barker' first='Bob'/><name last='Parker' first='Peter'/></attendees>` +
	`<e a="1" b="2"><c>t</c></e><e b="2" a="1"><c>t</c></e><e a="1"><c>t</c></e><e a="1" b="3"><c>t</c></e>` +
	`<f xmlns="urn:x"/><f/><g>text</g><g><h/></g><k A="x"/><k A="X"/>` +
	`<!--c1--><!--c1--><!--c2--><?p one?><?p one?><?q one?><?p two?></doc>`

// F&O 3.1 §14.2.3 fn:deep-equal.
func TestSeqSpecDeepEqual(t *testing.T) {
	sqRun(t, sqAttendees, []sqCase{
		// The REC examples.
		{`let $at := //attendees return deep-equal($at, $at/*)`, "false"},
		{`let $at := //attendees return deep-equal($at/name[1], $at/name[2])`, "false"},
		{`let $at := //attendees return deep-equal($at/name[1], $at/name[3])`, "true"},
		{`let $at := //attendees return deep-equal($at/name[1], 'Peter Parker')`, "false"},
		{`deep-equal(map{1:'a', 2:'b'}, map{2:'b', 1:'a'})`, "true"},
		{`deep-equal([1, 2, 3], [1, 2, 3])`, "true"},
		{`deep-equal((1, 2, 3), [1, 2, 3])`, "false"},
		// Notes: items of different kinds compare false, not an error.
		{`deep-equal(1, current-dateTime())`, "false"},
		{`deep-equal(map{}, [])`, "false"},
		{`deep-equal([], map{})`, "false"},
		{`deep-equal(1, map{})`, "false"},
		{`deep-equal(1, [1])`, "false"},
		{`deep-equal(//e[1], 1)`, "false"},

		// Atomic values: eq semantics, except that NaN is deep-equal to NaN.
		{`deep-equal(xs:double('NaN'), xs:float('NaN'))`, "true"},
		{`deep-equal(1, 1.0)`, "true"},
		{`deep-equal(xs:untypedAtomic('a'), 'a')`, "true"},
		{`deep-equal(xs:double('NaN'), 1)`, "false"},

		// Maps: same keys, values deep-equal; size matters.
		{`deep-equal(map{1:'a'}, map{1:'a', 2:'b'})`, "false"},
		{`deep-equal(map{1:'a'}, map{2:'a'})`, "false"},
		{`deep-equal(map{1:('a','b')}, map{1:('a','b')})`, "true"},
		{`deep-equal(map{1:('a','b')}, map{1:('b','a')})`, "false"},
		// Arrays: size and members in order.
		{`deep-equal([1, 2], [1, 2, 3])`, "false"},
		{`deep-equal([1, 2], [2, 1])`, "false"},
		{`deep-equal([(1,2), 3], [(1,2), 3])`, "true"},

		// Element nodes: name, attributes (any order), children.
		{`deep-equal(//e[1], //e[2])`, "true"},
		{`deep-equal(//e[1], //e[3])`, "false"},
		{`deep-equal(//e[1], //e[4])`, "false"},
		{`deep-equal(//f[1], //f[2])`, "false"},
		{`deep-equal(//g[1], //g[2])`, "false"},
		{`deep-equal(//e[1], //g[1])`, "false"},
		{`deep-equal(//e[1]/c, //e[2]/c)`, "true"},
		// Document nodes.
		{`deep-equal(/, /)`, "true"},
		{`deep-equal(/, /doc)`, "false"},
		// Attribute nodes: name and typed value.
		{`deep-equal(//e[1]/@a, //e[2]/@a)`, "true"},
		{`deep-equal(//e[1]/@a, //e[1]/@b)`, "false"},
		{`deep-equal(//e[1]/@b, //e[4]/@b)`, "false"},
		// Text, comment and PI nodes as items of the sequences.
		{`deep-equal(//g[1]/text(), //c[1]/text())`, "false"},
		{`deep-equal(//e[1]/c/text(), //e[2]/c/text())`, "true"},
		{`deep-equal(/doc/comment()[1], /doc/comment()[2])`, "true"},
		{`deep-equal(/doc/comment()[1], /doc/comment()[3])`, "false"},
		{`deep-equal(/doc/processing-instruction()[1], /doc/processing-instruction()[2])`, "true"},
		{`deep-equal(/doc/processing-instruction()[1], /doc/processing-instruction()[3])`, "false"},
		{`deep-equal(/doc/processing-instruction()[1], /doc/processing-instruction()[4])`, "false"},
		{`deep-equal(/doc/comment()[1], //c[1]/text())`, "false"},
		// Nodes of different kinds.
		{`deep-equal(//e[1]/@a, //e[1]/c)`, "false"},

		// The collation governs string comparisons, attribute values included.
		{`deep-equal('a', 'A', 'http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive')`, "true"},
		{`deep-equal('a', 'A', 'http://www.w3.org/2005/xpath-functions/collation/codepoint')`, "false"},
		{`deep-equal(//k[1], //k[2])`, "false"},
		{`deep-equal(//k[1], //k[2], 'http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive')`, "true"},
		{`deep-equal(('a', 1), ('A', 1), 'http://www.w3.org/2013/collation/UCA?strength=secondary')`, "true"},
	})
	sqRunErr(t, sqAttendees, []sqErrCase{
		// FOTY0015: a function item (not a map or array) anywhere in the input.
		{`deep-equal(abs#1, abs#1)`, "FOTY0015"},
		{`deep-equal(1, abs#1)`, "FOTY0015"},
		{`deep-equal([abs#1], [1])`, "FOTY0015"},
		{`deep-equal(map{1:abs#1}, map{1:1})`, "FOTY0015"},
		{`deep-equal(1, 1, 'http://example.com/no-such-collation')`, "FOCH0002"},
	})
}

// The REC's §14.2.3 rules compare element children as $i1/(*|text()) (or
// $i1/* for element-only content): comments and processing instructions
// among the children do not take part ("The contents of comments and
// processing instructions are significant only if these nodes appear directly
// as items in the two sequences being compared").
func TestSeqSpecDeepEqualIgnoresCommentChildren(t *testing.T) {
	sqRun(t, `<r><a><b/><!--x--><b/></a><a><b/><b/></a><a><b/><?pi?><b/></a></r>`, []sqCase{
		{`deep-equal(//a[1], //a[2])`, "true"},
		{`deep-equal(//a[3], //a[2])`, "true"},
	})
}

// F&O 3.1 §14.1: insert-before, remove, unordered, and the cardinality
// functions of §14.3.
func TestSeqSpecPositionalAndCardinality(t *testing.T) {
	sqRun(t, `<r><p>2</p><q>x</q></r>`, []sqCase{
		// §14.1.4 examples, $abc := ("a", "b", "c").
		{`string-join(insert-before(("a", "b", "c"), 0, "z"), ",")`, "z,a,b,c"},
		{`string-join(insert-before(("a", "b", "c"), 1, "z"), ",")`, "z,a,b,c"},
		{`string-join(insert-before(("a", "b", "c"), 2, "z"), ",")`, "a,z,b,c"},
		{`string-join(insert-before(("a", "b", "c"), 3, "z"), ",")`, "a,b,z,c"},
		{`string-join(insert-before(("a", "b", "c"), 4, "z"), ",")`, "a,b,c,z"},
		{`string-join(insert-before((), 3, ("x", "y")), ",")`, "x,y"},
		{`string-join(insert-before(("a", "b"), 2, ()), ",")`, "a,b"},
		// A node or untypedAtomic position is cast to xs:integer.
		{`string-join(insert-before(("a", "b", "c"), //p, "z"), ",")`, "a,z,b,c"},
		{`string-join(insert-before(("a", "b", "c"), xs:untypedAtomic('3'), "z"), ",")`, "a,b,z,c"},
		// Any integer subtype is acceptable.
		{`string-join(insert-before(("a", "b"), xs:byte(2), "z"), ",")`, "a,z,b"},

		// §14.1.5 examples.
		{`string-join(remove(("a", "b", "c"), 0), ",")`, "a,b,c"},
		{`string-join(remove(("a", "b", "c"), 1), ",")`, "b,c"},
		{`string-join(remove(("a", "b", "c"), 6), ",")`, "a,b,c"},
		{`count(remove((), 3))`, "0"},
		{`string-join(remove(("a", "b", "c"), //p), ",")`, "a,c"},
		{`string-join(remove(("a", "b", "c"), xs:untypedAtomic('3')), ",")`, "a,b"},

		// §14.1.8 fn:unordered: some permutation of the input.
		{`count(unordered((1, 2, 3)))`, "3"},
		{`sum(unordered((1, 2, 3)))`, "6"},

		// §14.3.
		{`count(zero-or-one(()))`, "0"},
		{`zero-or-one(4)`, "4"},
		{`count(one-or-more((1, 2)))`, "2"},
		{`exactly-one(//p)`, "2"},
	})
	sqRunErr(t, `<r><p>2</p><q>x</q></r>`, []sqErrCase{
		{`zero-or-one((1, 2))`, "FORG0003"},
		{`one-or-more(())`, "FORG0004"},
		{`exactly-one(())`, "FORG0005"},
		{`exactly-one((1, 2))`, "FORG0005"},
		// $position as xs:integer: no decimal/double/string, exactly one.
		{`insert-before((1), (), 2)`, "XPTY0004"},
		{`insert-before((1), (1, 2), 2)`, "XPTY0004"},
		{`insert-before((1), 1.0, 2)`, "XPTY0004"},
		{`insert-before((1), 1e0, 2)`, "XPTY0004"},
		{`insert-before((1), '1', 2)`, "XPTY0004"},
		{`remove((1), 1.0)`, "XPTY0004"},
		{`remove((1), "1")`, "XPTY0004"},
		{`remove((1), ())`, "XPTY0004"},
		// A node whose value is not an integer: the cast fails.
		{`remove((1), //q)`, "FORG0001"},
		{`insert-before((1), //q, 2)`, "FORG0001"},
		{`insert-before((1), xs:untypedAtomic('x'), 2)`, "FORG0001"},
	})
}
