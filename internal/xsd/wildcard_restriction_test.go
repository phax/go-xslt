package xsd

import "testing"

// Wildcard namespace algebra in restrictions (1.0 §3.10.6 Wildcard Subset,
// 1.1 §3.10.6.2 with notNamespace/notQName), xs:all mappings, substitution
// groups and the 1.1-only language-inclusion routes (1.1 §3.4.6.4 Content
// type restricts).
func TestWildcardAndAllRestriction(t *testing.T) {
	const (
		elA = `<xs:element name="a" type="xs:string"/>`
		elB = `<xs:element name="b" type="xs:string"/>`
		elC = `<xs:element name="c" type="xs:string"/>`
		sub = `<xs:element name="h" type="xs:string"/><xs:element name="m1" substitutionGroup="h" type="xs:string"/><xs:element name="m2" substitutionGroup="h" type="xs:string"/>`
	)
	wrap := func(g, body string) string { return "<xs:" + g + ">" + body + "</xs:" + g + ">" }
	anyW := func(attrs string) string { return `<xs:any processContents="lax" ` + attrs + `/>` }
	cases := []struct {
		name          string
		ver           int
		base, derived string
		extra         string
		valid         bool
	}{
		// 1.1 negated namespace constraints.
		{"not-ns-list-subset", ver11, wrap("sequence", anyW(`notNamespace="urn:x"`)), wrap("sequence", anyW(`namespace="urn:y"`)), "", true},
		{"not-ns-list-hits-excluded", ver11, wrap("sequence", anyW(`notNamespace="urn:x"`)), wrap("sequence", anyW(`namespace="urn:x"`)), "", false},
		{"not-ns-larger-exclusion", ver11, wrap("sequence", anyW(`notNamespace="urn:x"`)), wrap("sequence", anyW(`notNamespace="urn:x urn:y"`)), "", true},
		{"not-ns-other-exclusion", ver11, wrap("sequence", anyW(`notNamespace="urn:x"`)), wrap("sequence", anyW(`notNamespace="urn:y"`)), "", false},
		{"not-ns-vs-other", ver11, wrap("sequence", anyW(`notNamespace="urn:x"`)), wrap("sequence", anyW(`namespace="##other"`)), "", false},
		{"not-local-vs-other", ver11, wrap("sequence", anyW(`notNamespace="##local"`)), wrap("sequence", anyW(`namespace="##other"`)), "", true},
		{"any-base-not-derived", ver11, wrap("sequence", anyW(``)), wrap("sequence", anyW(`notNamespace="urn:x"`)), "", true},
		{"other-base-not-local-derived", ver11, wrap("sequence", anyW(`namespace="##other"`)), wrap("sequence", anyW(`notNamespace="##local"`)), "", true},
		{"other-base-not-derived-too-wide", ver11, wrap("sequence", anyW(`namespace="##other"`)), wrap("sequence", anyW(`notNamespace="urn:x"`)), "", false},
		{"list-base-not-derived", ver11, wrap("sequence", anyW(`namespace="urn:a"`)), wrap("sequence", anyW(`notNamespace="urn:x"`)), "", false},
		{"not-ns-base-any-derived", ver11, wrap("sequence", anyW(`notNamespace="urn:x"`)), wrap("sequence", anyW(``)), "", false},
		{"elt-vs-not-ns", ver11, wrap("sequence", anyW(`notNamespace="urn:x"`)), wrap("sequence", elA), "", true},
		// notQName: the derived wildcard may not admit an excluded name.
		{"not-qname-dropped", ver11, wrap("sequence", anyW(`notQName="a"`)), wrap("sequence", anyW(``)), "", false},
		{"not-qname-kept", ver11, wrap("sequence", anyW(`notQName="a"`)), wrap("sequence", anyW(`notQName="a b"`)), "", true},
		{"elt-excluded-by-not-qname", ver11, wrap("sequence", anyW(`notQName="a"`)), wrap("sequence", elA), "", false},
		{"elt-not-excluded-by-not-qname", ver11, wrap("sequence", anyW(`notQName="b"`)), wrap("sequence", elA), "", true},
		// ##other / lists in both versions (no target namespace: ##other
		// is "not absent").
		{"other-vs-other", verBoth, wrap("sequence", anyW(`namespace="##other"`)), wrap("sequence", anyW(`namespace="##other"`)), "", true},
		{"other-vs-list", verBoth, wrap("sequence", anyW(`namespace="##other"`)), wrap("sequence", anyW(`namespace="urn:a"`)), "", true},
		{"other-vs-list-with-local", verBoth, wrap("sequence", anyW(`namespace="##other"`)), wrap("sequence", anyW(`namespace="urn:a ##local"`)), "", false},
		{"list-vs-other", verBoth, wrap("sequence", anyW(`namespace="urn:a"`)), wrap("sequence", anyW(`namespace="##other"`)), "", false},
		{"any-vs-other", verBoth, wrap("sequence", anyW(``)), wrap("sequence", anyW(`namespace="##other"`)), "", true},

		// Sequence restricting an all (RecurseUnordered): order is free.
		{"seq-restricts-all", verBoth, wrap("all", elA+`<xs:element name="b" minOccurs="0"/>`), wrap("sequence", `<xs:element name="b"/>`+elA), "", true},
		{"seq-restricts-all-unmapped", verBoth, wrap("all", elA+`<xs:element name="b" minOccurs="0"/>`), wrap("sequence", elA+elC), "", false},
		{"seq-restricts-all-missing-required", verBoth, wrap("all", elA+elB), wrap("sequence", elA), "", false},
		// Forbidden pairs in the 1.0 table (all:seq, all:choice, choice:all).
		{"all-restricts-seq", ver10, wrap("sequence", elA+elB), wrap("all", elA+elB), "", false},
		{"all-restricts-choice", verBoth, wrap("choice", elA+elB), wrap("all", elA+`<xs:element name="b" minOccurs="0"/>`), "", false},
		{"choice-restricts-all", ver10, wrap("all", `<xs:element name="a" minOccurs="0"/><xs:element name="b" minOccurs="0"/>`), wrap("choice", elA+elB), "", false},
		// 1.1: L(choice(a,b)) = {a},{b} ⊆ L(all(a?,b?)).
		{"choice-restricts-all-1.1", ver11, wrap("all", `<xs:element name="a" minOccurs="0"/><xs:element name="b" minOccurs="0"/>`), wrap("choice", elA+elB), "", true},
		{"choice-restricts-all-1.1-branch-misses", ver11, wrap("all", `<xs:element name="a" minOccurs="0"/><xs:element name="b" minOccurs="0"/>`), wrap("choice", elA+elC), "", false},
		// A 1..1 choice nested in a sequence over a run of optional base
		// particles: valid by language in 1.1, Forbidden (group:elt) in 1.0.
		{"choice-over-run-1.1", ver11, wrap("sequence", `<xs:element name="a" minOccurs="0"/><xs:element name="b" minOccurs="0"/>`), wrap("sequence", wrap("choice", elA+elB)), "", true},
		{"choice-over-run-1.0", ver10, wrap("sequence", `<xs:element name="a" minOccurs="0"/><xs:element name="b" minOccurs="0"/>`), wrap("sequence", wrap("choice", elA+elB)), "", false},
		// 1.1 xs:all with wildcards.
		{"all-wildcard-narrowed", ver11, wrap("all", elA+anyW(`namespace="##other"`)), wrap("all", elA+anyW(`namespace="urn:x"`)), "", true},
		{"all-wildcard-widened", ver11, wrap("all", elA+anyW(`namespace="urn:x"`)), wrap("all", elA+anyW(`namespace="##other"`)), "", false},
		// Two optional base wildcards that jointly cover ##any.
		{"all-wildcard-union", ver11, wrap("all", anyW(`namespace="##local" minOccurs="0"`)+anyW(`notNamespace="##local" minOccurs="0"`)), wrap("all", anyW(`minOccurs="0"`)), "", true},
		{"all-wildcard-union-gap", ver11, wrap("all", anyW(`namespace="##local" minOccurs="0"`)+anyW(`namespace="urn:x" minOccurs="0"`)), wrap("all", anyW(`minOccurs="0"`)), "", false},
		// Substitution members fan in onto their head inside an all.
		{"all-members-fan-in", ver11, wrap("all", `<xs:element ref="h" minOccurs="0" maxOccurs="4"/>`), wrap("all", `<xs:element ref="m1" minOccurs="0" maxOccurs="2"/><xs:element ref="m2" minOccurs="0" maxOccurs="2"/>`), sub, true},
		{"all-members-fan-in-too-many", ver11 + skipBug, wrap("all", `<xs:element ref="h" minOccurs="0" maxOccurs="3"/>`), wrap("all", `<xs:element ref="m1" minOccurs="0" maxOccurs="2"/><xs:element ref="m2" minOccurs="0" maxOccurs="2"/>`), sub, false},
		// A choice of members restricting the head (1.0 clause 2.2.1: the
		// head is treated as a choice of its substitution group).
		{"choice-of-members-restricts-head", verBoth, wrap("sequence", `<xs:element ref="h"/>`), wrap("sequence", wrap("choice", `<xs:element ref="m1"/><xs:element ref="m2"/>`)), sub, true},
		{"choice-member-or-wildcard-vs-head", verBoth, wrap("sequence", `<xs:element ref="h"/>`), wrap("sequence", wrap("choice", `<xs:element ref="m1"/>`+anyW(``))), sub, false},
		// A member whose substitution the head blocks cannot stand for it.
		{"blocked-member-restricts-head", verBoth, wrap("sequence", `<xs:element ref="h"/>`), wrap("sequence", `<xs:element ref="m1"/>`),
			`<xs:element name="h" type="xs:string" block="substitution"/><xs:element name="m1" substitutionGroup="h" type="xs:string"/>`, false},
		// A repeatable base wildcard absorbs several derived particles in a
		// sequence, compared on the summed element count.
		{"wildcard-absorbs-run", verBoth, wrap("sequence", anyW(`minOccurs="2" maxOccurs="3"`)), wrap("sequence", elA+elB), "", true},
		{"wildcard-absorbs-run-too-few", verBoth, wrap("sequence", anyW(`minOccurs="3" maxOccurs="3"`)), wrap("sequence", elA+elB), "", false},
		{"wildcard-absorbs-run-too-many", verBoth, wrap("sequence", anyW(`minOccurs="1" maxOccurs="2"`)), wrap("sequence", elA+elB+elC), "", false},
		// A choice restricting a wildcard: element count of the choice.
		{"choice-of-groups-vs-wildcard", verBoth, wrap("sequence", anyW(`maxOccurs="2"`)), wrap("sequence", wrap("choice", wrap("sequence", elA+elB)+elC)), "", true},
		{"choice-of-groups-vs-wildcard-unbounded", verBoth, wrap("sequence", anyW(`maxOccurs="2"`)), wrap("sequence", wrap("choice", `<xs:element name="a" maxOccurs="unbounded"/>`+elC)), "", false},
		// Particles removed with maxOccurs="0" contribute nothing.
		{"removed-particle", verBoth, wrap("sequence", elA+`<xs:element name="b" minOccurs="0"/>`), wrap("sequence", elA+`<xs:element name="b" minOccurs="0" maxOccurs="0"/>`), "", true},
		{"removed-choice-branch", verBoth, wrap("choice", elA+elB), wrap("choice", elA+`<xs:element name="b" minOccurs="0" maxOccurs="0"/>`), "", true},
	}
	for _, tc := range cases {
		for _, v := range versionsOf(tc.ver % skipBug) {
			t.Run(tc.name+"/"+v.String(), func(t *testing.T) {
				if tc.ver >= skipBug {
					t.Skip("BUG: engine verdict disagrees with the spec")
				}
				got := schemaVerdict(t, restrictionSchema(tc.base, tc.derived, tc.extra), v)
				if got != tc.valid {
					t.Fatalf("schema valid=%v, want %v", got, tc.valid)
				}
			})
		}
	}
}
