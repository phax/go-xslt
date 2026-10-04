package engine

import (
	"errors"
	"strings"
	"testing"
)

// TestTransformFromReaderMirrorsTransform checks the contract: identical
// Result to Transform, with the source read from an io.Reader instead of
// req.Source (which this function ignores).
func TestTransformFromReaderMirrorsTransform(t *testing.T) {
	req := Request{
		Stylesheet: ttSheet(`<xsl:output method="xml"/>`,
			`<out><xsl:for-each select="root/rec"><r><xsl:value-of select="@id"/></r></xsl:for-each></out>`),
	}
	want := Transform(Request{Stylesheet: req.Stylesheet, Source: ttSrc})
	if want.HasErrors() {
		t.Fatalf("Transform: %v", want.Diagnostics)
	}
	got := TransformFromReader(strings.NewReader(ttSrc), req)
	if got.HasErrors() {
		t.Fatalf("TransformFromReader: %v", got.Diagnostics)
	}
	if got.Output != want.Output {
		t.Errorf("Output = %q, want %q", got.Output, want.Output)
	}
	if got.Method != want.Method {
		t.Errorf("Method %q, want %q", got.Method, want.Method)
	}
}

// TestTransformFromReaderIgnoresRequestSource pins that req.Source is not
// consulted at all: a caller who sets both must get the reader's content,
// never a silent fallback to the string field.
func TestTransformFromReaderIgnoresRequestSource(t *testing.T) {
	req := Request{
		Stylesheet: ttSheet(`<xsl:output method="text"/>`, `<xsl:value-of select="root/@from"/>`),
		Source:     `<root from="field"/>`,
	}
	got := TransformFromReader(strings.NewReader(`<root from="reader"/>`), req)
	if got.HasErrors() {
		t.Fatalf("TransformFromReader: %v", got.Diagnostics)
	}
	if got.Output != "reader" {
		t.Errorf("Output = %q, want %q (req.Source must be ignored)", got.Output, "reader")
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

// TestTransformFromReaderReportsReadError checks a failed read becomes a
// Diagnostic rather than a panic.
func TestTransformFromReaderReportsReadError(t *testing.T) {
	req := Request{Stylesheet: ttSheet(``, `<out/>`)}
	got := TransformFromReader(errReader{}, req)
	if !got.HasErrors() {
		t.Fatal("expected a read-error diagnostic, got none")
	}
}
