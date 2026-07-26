package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// mustParseSource parses a Go source snippet (given as a %s-style format
// string; use a literal backtick placeholder to embed struct tags) and
// returns the resulting *ast.File.
func mustParseSource(t *testing.T, src string) *ast.File {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("failed to parse test source: %v\n%s", err, src)
	}
	return f
}

// tag builds a `validate:"..."` struct tag literal from a Go source
// fragment, without needing an actual backtick inside the test's own
// source string.
func tag(validate string) string {
	return "`validate:\"" + validate + "\"`"
}

// --- parseValidateTag (shared constraint tag parser) -----------------------

func TestParseValidateTag(t *testing.T) {
	tests := []struct {
		name    string
		tagStr  string
		want    []tagConstraint
		wantErr bool
	}{
		{
			name:   "single constraint no value",
			tagStr: "required",
			want:   []tagConstraint{{Name: "required"}},
		},
		{
			name:   "multiple constraints separated by a single space",
			tagStr: "required min:3 max:5",
			want: []tagConstraint{
				{Name: "required"},
				{Name: "min", Value: "3"},
				{Name: "max", Value: "5"},
			},
		},
		{
			name:   "constraints separated by mixed/repeated whitespace",
			tagStr: "required  min:3\tmax:5",
			want: []tagConstraint{
				{Name: "required"},
				{Name: "min", Value: "3"},
				{Name: "max", Value: "5"},
			},
		},
		{
			name:   "value containing additional colons is preserved whole",
			tagStr: `regexp:'^[0-9]+:[0-9]+$'`,
			want:   []tagConstraint{{Name: "regexp", Value: `'^[0-9]+:[0-9]+$'`}},
		},
		{
			name:    "min missing a value",
			tagStr:  "min",
			wantErr: true,
		},
		{
			name:    "max missing a value (trailing colon)",
			tagStr:  "max:",
			wantErr: true,
		},
		{
			name:    "eqfield missing a value",
			tagStr:  "eqfield",
			wantErr: true,
		},
		{
			name:    "regexp missing a value",
			tagStr:  "regexp",
			wantErr: true,
		},
		{
			name:   "required takes no value and none is needed",
			tagStr: "required",
			want:   []tagConstraint{{Name: "required"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseValidateTag("S", "F", tc.tagStr)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got none")
				}
				if !strings.Contains(err.Error(), "S") || !strings.Contains(err.Error(), "F") {
					t.Fatalf("expected error to include struct/field context, got: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %d constraints, want %d: %+v", len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("constraint %d = %+v, want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// --- getConstraints: unknown constraints & missing values -------------------

func TestGetConstraintsUnknownConstraint(t *testing.T) {
	tests := []struct {
		name string
		tag  string
	}{
		{name: "misspelled required", tag: "requird"},
		{name: "misspelled min", tag: "minn:3"},
		{name: "completely made up", tag: "frobnicate:5"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fields := []structField{{Name: "Name", Type: "string", Tag: tc.tag}}
			_, err := getConstraints("Widget", fields)
			if err == nil {
				t.Fatalf("expected an error for unknown constraint %q, got nil", tc.tag)
			}
			for _, want := range []string{"Widget", "Name"} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("expected error to mention %q, got: %v", want, err)
				}
			}
		})
	}
}

func TestGetConstraintsMissingValue(t *testing.T) {
	fields := []structField{{Name: "Age", Type: "int", Tag: "min"}}
	_, err := getConstraints("Widget", fields)
	if err == nil {
		t.Fatalf("expected an error for a min constraint missing its value, got nil")
	}
	if !strings.Contains(err.Error(), "Widget") || !strings.Contains(err.Error(), "Age") {
		t.Fatalf("expected error to include struct/field context, got: %v", err)
	}
}

// --- required: every built-in numeric type, plus string/bool --------------

func TestGetConstraintsRequiredBuiltinTypes(t *testing.T) {
	tests := []struct {
		typ       string
		wantValue string
	}{
		{typ: "string", wantValue: `""`},
		{typ: "bool", wantValue: "false"},
		{typ: "int", wantValue: "0"},
		{typ: "int8", wantValue: "0"},
		{typ: "int16", wantValue: "0"},
		{typ: "int32", wantValue: "0"},
		{typ: "int64", wantValue: "0"},
		{typ: "uint", wantValue: "0"},
		{typ: "uint8", wantValue: "0"},
		{typ: "uint16", wantValue: "0"},
		{typ: "uint32", wantValue: "0"},
		{typ: "uint64", wantValue: "0"},
		{typ: "uintptr", wantValue: "0"},
		{typ: "float32", wantValue: "0"},
		{typ: "float64", wantValue: "0"},
		{typ: "complex64", wantValue: "0"},
		{typ: "complex128", wantValue: "0"},
		{typ: "byte", wantValue: "0"},
		{typ: "rune", wantValue: "0"},
	}

	for _, tc := range tests {
		t.Run(tc.typ, func(t *testing.T) {
			fields := []structField{{Name: "Field", Type: tc.typ, Tag: "required"}}
			cs, err := getConstraints("Widget", fields)
			if err != nil {
				t.Fatalf("unexpected error for type %s: %v", tc.typ, err)
			}
			if len(cs) != 1 {
				t.Fatalf("expected exactly one constraint, got %d", len(cs))
			}
			if cs[0].Op != "==" {
				t.Fatalf("expected op ==, got %s", cs[0].Op)
			}
			if cs[0].Value != tc.wantValue {
				t.Fatalf("type %s: value = %q, want %q", tc.typ, cs[0].Value, tc.wantValue)
			}
		})
	}
}

// --- maps: required (nil) and min/max (len) --------------------------------

func TestGetConstraintsMap(t *testing.T) {
	fields := []structField{
		{Name: "Tags", Type: "map[string]int", Tag: "required min:1 max:5"},
	}
	cs, err := getConstraints("Widget", fields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cs) != 3 {
		t.Fatalf("expected 3 constraints, got %d: %+v", len(cs), cs)
	}

	required := cs[0]
	if required.FieldName != "s.Tags" || required.Value != "nil" {
		t.Fatalf("required constraint for map = %+v, want FieldName s.Tags, Value nil", required)
	}

	min := cs[1]
	if min.FieldName != "len(s.Tags)" || min.Op != "<" {
		t.Fatalf("min constraint for map = %+v, want len(s.Tags) <", min)
	}

	max := cs[2]
	if max.FieldName != "len(s.Tags)" || max.Op != ">" {
		t.Fatalf("max constraint for map = %+v, want len(s.Tags) >", max)
	}
}

// --- arrays: min/max via len(), required explicitly rejected ---------------

func TestGetConstraintsArray(t *testing.T) {
	fields := []structField{
		{Name: "Scores", Type: "[3]int", Tag: "min:1 max:3"},
	}
	cs, err := getConstraints("Widget", fields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cs) != 2 || cs[0].FieldName != "len(s.Scores)" || cs[1].FieldName != "len(s.Scores)" {
		t.Fatalf("expected len()-based min/max for array, got %+v", cs)
	}
}

func TestGetConstraintsArrayRequiredRejected(t *testing.T) {
	fields := []structField{
		{Name: "Scores", Type: "[3]int", Tag: "required"},
	}
	_, err := getConstraints("Widget", fields)
	if err == nil {
		t.Fatalf("expected required on an array field to be rejected, got nil error")
	}
	if !strings.Contains(err.Error(), "Widget") || !strings.Contains(err.Error(), "Scores") {
		t.Fatalf("expected error to include struct/field context, got: %v", err)
	}
}

// --- pointers: required is a nil check, min/max is rejected ----------------

func TestGetConstraintsPointerRequired(t *testing.T) {
	fields := []structField{
		{Name: "Owner", Type: "*User", Tag: "required"},
	}
	cs, err := getConstraints("Widget", fields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cs) != 1 || cs[0].Value != "nil" {
		t.Fatalf("expected a nil check for a pointer field, got %+v", cs)
	}
}

func TestGetConstraintsPointerMinMaxRejected(t *testing.T) {
	fields := []structField{
		{Name: "Owner", Type: "*int", Tag: "min:1"},
	}
	_, err := getConstraints("Widget", fields)
	if err == nil {
		t.Fatalf("expected min on a pointer field to be rejected, got nil error")
	}
}

// --- selectors (qualified types): required is rejected with guidance -------

func TestGetConstraintsSelectorRequiredRejected(t *testing.T) {
	fields := []structField{
		{Name: "Owner", Type: "pkg.User", Tag: "required"},
	}
	_, err := getConstraints("Widget", fields)
	if err == nil {
		t.Fatalf("expected required on a qualified type field to be rejected, got nil error")
	}
	if !strings.Contains(err.Error(), "pointer") {
		t.Fatalf("expected error to suggest wrapping in a pointer, got: %v", err)
	}
}

// --- max: corrected wording ("greater", not "greather") --------------------

func TestGetConstraintsMaxWording(t *testing.T) {
	fields := []structField{{Name: "Age", Type: "int", Tag: "max:10"}}
	cs, err := getConstraints("Widget", fields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(cs[0].Error, "greather") {
		t.Fatalf("expected corrected spelling, got: %q", cs[0].Error)
	}
	if !strings.Contains(cs[0].Error, "greater") {
		t.Fatalf("expected %q to contain \"greater\"", cs[0].Error)
	}
}

// --- parser.go: AST type rendering without unchecked assertions ------------

func TestParseNodeFieldTypes(t *testing.T) {
	tests := []struct {
		name     string
		fieldSrc string
		wantType string
	}{
		{name: "slice of pointer", fieldSrc: "Items []*User " + tag("required"), wantType: "[]*User"},
		{name: "slice of qualified type", fieldSrc: "Items []pkg.User " + tag("required"), wantType: "[]pkg.User"},
		{name: "map", fieldSrc: "M map[string]int " + tag("required"), wantType: "map[string]int"},
		{name: "array", fieldSrc: "A [3]int " + tag("required"), wantType: "[3]int"},
		{name: "pointer", fieldSrc: "P *int " + tag("required"), wantType: "*int"},
		{name: "selector", fieldSrc: "Q pkg.User " + tag("required"), wantType: "pkg.User"},
		{name: "pointer to qualified type", fieldSrc: "R *pkg.User " + tag("required"), wantType: "*pkg.User"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := "package p\n\ntype User struct{}\n\ntype S struct {\n\t" + tc.fieldSrc + "\n}\n"
			f := mustParseSource(t, src)

			structs, err := parseNode(f)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			fields, ok := structs["S"]
			if !ok || len(fields) != 1 {
				t.Fatalf("expected exactly one tagged field on S, got %+v", structs)
			}
			if fields[0].Type != tc.wantType {
				t.Fatalf("field type = %q, want %q", fields[0].Type, tc.wantType)
			}
		})
	}
}

func TestParseNodeEmbeddedTaggedFieldRejected(t *testing.T) {
	src := "package p\n\ntype Embedded struct{}\n\ntype S struct {\n\tEmbedded " + tag("required") + "\n}\n"
	f := mustParseSource(t, src)

	_, err := parseNode(f)
	if err == nil {
		t.Fatalf("expected a validate tag on an embedded field to be rejected, got nil error")
	}
}

func TestParseNodeEmbeddedUntaggedFieldIgnored(t *testing.T) {
	src := "package p\n\ntype Embedded struct{}\n\ntype S struct {\n\tEmbedded\n\tName string " + tag("required") + "\n}\n"
	f := mustParseSource(t, src)

	structs, err := parseNode(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	fields, ok := structs["S"]
	if !ok || len(fields) != 1 || fields[0].Name != "Name" {
		t.Fatalf("expected only the tagged Name field, got %+v", structs)
	}
}

func TestParseNodeUnsupportedFieldTypeRejected(t *testing.T) {
	src := "package p\n\ntype S struct {\n\tF func() " + tag("required") + "\n}\n"
	f := mustParseSource(t, src)

	_, err := parseNode(f)
	if err == nil {
		t.Fatalf("expected an unsupported field type (func) to produce an error, got nil")
	}
	if !strings.Contains(err.Error(), "S") || !strings.Contains(err.Error(), "F") {
		t.Fatalf("expected error to include struct/field context, got: %v", err)
	}
}
