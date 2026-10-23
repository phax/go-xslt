package engine

import (
	"context"
	"io"
	"strings"
	"time"
)

// TransformFromReader is Transform with the source document read from src
// instead of taken from req.Source (which is ignored). A caller already
// holding the source as an io.Reader — a file, a network body — would
// otherwise have to materialize it into a string themselves before calling
// Transform, which costs a copy on top of whatever io.ReadAll itself
// allocates: ReadAll's own growing buffer, then another full copy for the
// []byte->string conversion. Reading src into a strings.Builder and taking
// its String() needs only the one buffer — Builder.String() is a zero-copy
// view of it — so this avoids that second copy entirely.
//
// Like Transform it never panics: an internal panic, or a read error from
// src, becomes an error diagnostic.
func TransformFromReader(src io.Reader, req Request) (res Result) {
	start := time.Now()
	defer func() {
		if r := recover(); r != nil {
			res.Diagnostics = append(res.Diagnostics, Diagnostic{
				Severity: SeverityError,
				Code:     "INTERNAL",
				Phase:    "run",
				Message:  "internal engine error: " + toString(r),
			})
		}
		res.DurationMs = time.Since(start).Milliseconds()
	}()

	var sb strings.Builder
	if _, err := io.Copy(&sb, src); err != nil {
		return Result{Diagnostics: []Diagnostic{{
			Severity: SeverityError,
			Code:     "IO",
			Phase:    "read",
			Message:  "reading source: " + err.Error(),
		}}}
	}
	req.Source = sb.String()
	return run(context.Background(), req)
}
