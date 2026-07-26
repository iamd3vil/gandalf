package main

import (
	"bytes"
	"go/format"
	"path/filepath"
	"strings"
	"testing"

	"github.com/knadh/stuffbin"
)

// testFileSystem builds a stuffbin.FileSystem backed by the repository's
// on-disk templates directory, mirroring the STATIC mapping used by the
// Makefile's `build` target (./templates:/templates), so tests exercise the
// real template without requiring a packed binary.
func testFileSystem(t *testing.T) stuffbin.FileSystem {
	t.Helper()

	tmplDir, err := filepath.Abs(filepath.Join("..", "..", "templates"))
	if err != nil {
		t.Fatalf("couldn't resolve templates dir: %v", err)
	}

	fs, err := stuffbin.NewLocalFS("/", tmplDir+":/templates")
	if err != nil {
		t.Fatalf("couldn't build local stuffbin filesystem: %v", err)
	}
	return fs
}

// generate is a small helper that runs generateCodeForStructs and returns
// the raw, unformatted generated source.
func generate(t *testing.T, structs map[string][]structField) (string, error) {
	t.Helper()

	fs := testFileSystem(t)
	buf := &bytes.Buffer{}
	err := generateCodeForStructs(fs, "testpkg", structs, buf)
	return buf.String(), err
}

func TestGenerateDeterministicOutput(t *testing.T) {
	structs := map[string][]structField{
		"Zebra": {
			{Name: "Name", Type: "string", Tag: "required"},
		},
		"Apple": {
			{Name: "Name", Type: "string", Tag: "required"},
		},
		"Mango": {
			{Name: "Code", Type: "string", Tag: "regexp:'^[A-Z]+$'"},
		},
	}

	var first string
	for i := 0; i < 20; i++ {
		out, err := generate(t, structs)
		if err != nil {
			t.Fatalf("generate() unexpected error on iteration %d: %v", i, err)
		}
		if i == 0 {
			first = out
			continue
		}
		if out != first {
			t.Fatalf("generated output is not deterministic across repeated runs (iteration %d differs)", i)
		}
	}

	// Struct methods must appear in sorted (Apple, Mango, Zebra) order,
	// regardless of the map's random iteration order.
	appleIdx := strings.Index(first, "func (s *Apple) Validate()")
	mangoIdx := strings.Index(first, "func (s *Mango) Validate()")
	zebraIdx := strings.Index(first, "func (s *Zebra) Validate()")
	if appleIdx == -1 || mangoIdx == -1 || zebraIdx == -1 {
		t.Fatalf("expected all three Validate methods in output, got:\n%s", first)
	}
	if !(appleIdx < mangoIdx && mangoIdx < zebraIdx) {
		t.Fatalf("expected structs in sorted order Apple < Mango < Zebra, got indices %d, %d, %d", appleIdx, mangoIdx, zebraIdx)
	}

	if _, err := format.Source([]byte(first)); err != nil {
		t.Fatalf("generated code doesn't parse/format: %v\n%s", err, first)
	}
}

func TestGenerateWithoutRegex(t *testing.T) {
	structs := map[string][]structField{
		"Account": {
			{Name: "Username", Type: "string", Tag: "required min:3"},
		},
	}

	out, err := generate(t, structs)
	if err != nil {
		t.Fatalf("generate() unexpected error: %v", err)
	}
	if strings.Contains(out, "\"regexp\"") {
		t.Fatalf("expected no regexp import when no regexp constraints are present, got:\n%s", out)
	}
	if strings.Contains(out, "Regex") {
		t.Fatalf("expected no regex declarations, got:\n%s", out)
	}

	if _, err := format.Source([]byte(out)); err != nil {
		t.Fatalf("generated code doesn't parse/format: %v\n%s", err, out)
	}
}

func TestGenerateWithRegexEscapedPatterns(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
	}{
		{name: "colon", pattern: `^[a-z]+:[0-9]+$`},
		{name: "backslash_digit_class", pattern: `^\d{3}-\d{4}$`},
		{name: "quote_in_char_class", pattern: `^["']+$`},
		{name: "backslash_and_quote", pattern: `^\\"quoted\\"$`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			structs := map[string][]structField{
				"Widget": {
					{Name: "Code", Type: "string", Tag: "regexp:'" + tc.pattern + "'"},
				},
			}

			out, err := generate(t, structs)
			if err != nil {
				t.Fatalf("generate() unexpected error for pattern %q: %v", tc.pattern, err)
			}

			fmted, err := format.Source([]byte(out))
			if err != nil {
				t.Fatalf("generated code doesn't parse/format for pattern %q: %v\n%s", tc.pattern, err, out)
			}
			if !strings.Contains(string(fmted), "regexp.MustCompile(") {
				t.Fatalf("expected a regexp.MustCompile call for pattern %q, got:\n%s", tc.pattern, fmted)
			}
		})
	}
}

func TestGenerateWithInvalidRegexRejected(t *testing.T) {
	structs := map[string][]structField{
		"Widget": {
			{Name: "Code", Type: "string", Tag: "regexp:'^(unterminated'"},
		},
	}

	_, err := generate(t, structs)
	if err == nil {
		t.Fatalf("expected an error for an invalid regexp, got nil")
	}
	if !strings.Contains(err.Error(), "Widget") || !strings.Contains(err.Error(), "Code") {
		t.Fatalf("expected error to include struct and field context, got: %v", err)
	}
}
