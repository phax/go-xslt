package xslt

import (
	"runtime"
	"strings"
	"sync"
	"testing"
)

const importSchemaXSL = `<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xsl:import-schema><xs:schema><xs:simpleType name="t"><xs:restriction base="xs:int"/></xs:simpleType></xs:schema></xsl:import-schema>
  <xsl:template name="xsl:initial-template"><r aware="{system-property('xsl:is-schema-aware')}" ok="{3 castable as t}"/></xsl:template>
</xsl:stylesheet>`

// Features belong to the compiled stylesheet: a schema-aware and a basic
// stylesheet compiled and run at the same time each keep their own.
func TestFeaturesArePerStylesheet(t *testing.T) {
	if DefaultFeatures().SchemaAware {
		t.Skip("process-wide schema awareness is on")
	}
	var wg sync.WaitGroup
	errs := make(chan string, 64)
	for i := 0; i < 16; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			ss, err := CompileWith(importSchemaXSL, "", CompileOptions{Features: &Features{SchemaAware: true, StreamingClaim: true}})
			if err != nil {
				errs <- "schema-aware compile: " + err.Error()
				return
			}
			rr, err := ss.TransformEntry("", Entry{Template: "xsl:initial-template"}, "")
			if err != nil || !strings.Contains(rr.Output, `aware="yes" ok="true"`) {
				errs <- "schema-aware run: " + rr.Output
			}
		}()
		go func() {
			defer wg.Done()
			if _, err := CompileWith(importSchemaXSL, "", CompileOptions{}); err == nil || !strings.Contains(err.Error(), "optional feature") {
				errs <- "basic compile must reject xsl:import-schema"
			}
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
}

func registryLen(m *sync.Map) int {
	n := 0
	m.Range(func(_, _ any) bool { n++; return true })
	return n
}

// The registries must not keep compiled stylesheets alive: a host that
// compiles one stylesheet after another would otherwise grow without bound.
func TestRegistriesReleaseStylesheets(t *testing.T) {
	before := registryLen(&featureRegistry)
	for i := 0; i < 200; i++ {
		if _, err := CompileWith(importSchemaXSL, "", CompileOptions{Features: &Features{SchemaAware: true}}); err != nil {
			t.Fatal(err)
		}
	}
	if registryLen(&schemaRegistry) == 0 {
		t.Fatal("control: compiling registered no schema at all")
	}
	// Cleanups run after a collection, on their own goroutine; give them a
	// bounded number of collections, not a wall-clock wait.
	for i := 0; i < 50 && registryLen(&featureRegistry) > before+10; i++ {
		runtime.GC()
		runtime.Gosched()
	}
	if n := registryLen(&featureRegistry); n > before+10 {
		t.Errorf("feature registry still holds %d entries after 200 discarded compiles", n-before)
	}
	if n := registryLen(&schemaRegistry); n > 10 {
		t.Errorf("schema registry still holds %d entries after 200 discarded compiles", n)
	}
}
