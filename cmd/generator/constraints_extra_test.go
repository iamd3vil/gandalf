package main

import (
	"go/parser"
	"regexp"
	"strings"
	"testing"
)

// These tests cover the constraint builders in constraints_extra.go in
// isolation: they assert on the constraint struct each function returns
// (mainly its Raw expression and Error message), not on the rendered output
// of the full generation pipeline.

// --- eq / ne ---------------------------------------------------------------

func TestNewConstraintForEq(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		value     string
		wantRaw   string
		wantErr   string
	}{
		{
			name:      "string value is quoted",
			fieldName: "Kind",
			value:     "admin",
			wantRaw:   `s.Kind != "admin"`,
			wantErr:   "kind should be admin",
		},
		{
			name:      "numeric value is a bare literal",
			fieldName: "Age",
			value:     "18",
			wantRaw:   "s.Age != 18",
			wantErr:   "age should be 18",
		},
		{
			name:      "negative numeric value is a bare literal",
			fieldName: "Delta",
			value:     "-3",
			wantRaw:   "s.Delta != -3",
			wantErr:   "delta should be -3",
		},
		{
			name:      "string value with a quote is escaped",
			fieldName: "Kind",
			value:     `a"b`,
			wantRaw:   `s.Kind != "a\"b"`,
			wantErr:   `kind should be a"b`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := getConstraintForEq(tc.fieldName, tc.value)
			assertRawConstraint(t, got, tc.wantRaw, tc.wantErr)
		})
	}
}

func TestNewConstraintForNe(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		value     string
		wantRaw   string
		wantErr   string
	}{
		{
			name:      "string value is quoted",
			fieldName: "Kind",
			value:     "admin",
			wantRaw:   `s.Kind == "admin"`,
			wantErr:   "kind should not be admin",
		},
		{
			name:      "numeric value is a bare literal",
			fieldName: "Age",
			value:     "0",
			wantRaw:   "s.Age == 0",
			wantErr:   "age should not be 0",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := getConstraintForNe(tc.fieldName, tc.value)
			assertRawConstraint(t, got, tc.wantRaw, tc.wantErr)
		})
	}
}

// --- oneof ----------------------------------------------------------------

func TestNewConstraintForOneOf(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		value     string
		wantRaw   string
		wantErr   string
	}{
		{
			name:      "multiple values are ANDed together",
			fieldName: "Kind",
			value:     "admin,user,editor",
			wantRaw:   `s.Kind != "admin" && s.Kind != "user" && s.Kind != "editor"`,
			wantErr:   "kind should be one of admin, user, editor",
		},
		{
			name:      "single value",
			fieldName: "Kind",
			value:     "admin",
			wantRaw:   `s.Kind != "admin"`,
			wantErr:   "kind should be one of admin",
		},
		{
			name:      "surrounding single quotes are stripped",
			fieldName: "Kind",
			value:     "'admin,user'",
			wantRaw:   `s.Kind != "admin" && s.Kind != "user"`,
			wantErr:   "kind should be one of admin, user",
		},
		{
			name:      "extra whitespace around values is trimmed",
			fieldName: "Kind",
			value:     "'admin  ,  user'",
			wantRaw:   `s.Kind != "admin" && s.Kind != "user"`,
			wantErr:   "kind should be one of admin, user",
		},
		{
			name:      "empty list never fires",
			fieldName: "Kind",
			value:     "",
			wantRaw:   "false",
			wantErr:   "kind should be one of ",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := getConstraintForOneOf(tc.fieldName, tc.value)
			assertRawConstraint(t, got, tc.wantRaw, tc.wantErr)
		})
	}
}

// --- len ------------------------------------------------------------------

func TestNewConstraintForLenSupportedTypes(t *testing.T) {
	tests := []struct {
		name      string
		fieldType string
		wantRaw   string
	}{
		{name: "string", fieldType: "string", wantRaw: "len(s.Name) != 3"},
		{name: "slice", fieldType: "[]string", wantRaw: "len(s.Name) != 3"},
		{name: "array", fieldType: "[3]int", wantRaw: "len(s.Name) != 3"},
		{name: "map", fieldType: "map[string]int", wantRaw: "len(s.Name) != 3"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := getConstraintForLen("Name", tc.fieldType, "3")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertRawConstraint(t, got, tc.wantRaw, "name length should be 3")
		})
	}
}

func TestNewConstraintForLenUnsupportedTypes(t *testing.T) {
	for _, fieldType := range []string{"int", "bool", "*string", "pkg.User"} {
		t.Run(fieldType, func(t *testing.T) {
			_, err := getConstraintForLen("Name", fieldType, "3")
			if err == nil {
				t.Fatalf("expected an error for type %s, got nil", fieldType)
			}
			if !strings.Contains(err.Error(), "len is not supported for type "+fieldType) {
				t.Errorf("unexpected error message: %v", err)
			}
		})
	}
}

// --- unique ---------------------------------------------------------------

func TestNewConstraintForUniqueStringSlice(t *testing.T) {
	got, err := getConstraintForUnique("User", "Tags", "[]string")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "func() bool { seen := make(map[string]bool, len(s.Tags)); for _, v := range s.Tags { if seen[v] { return true }; seen[v] = true }; return false }()"
	assertRawConstraint(t, got, want, "tags contains duplicate values")
}

func TestNewConstraintForUniqueNonSliceRejected(t *testing.T) {
	for _, fieldType := range []string{"string", "int", "map[string]int", "[3]int", "*[]string"} {
		t.Run(fieldType, func(t *testing.T) {
			_, err := getConstraintForUnique("User", "Tags", fieldType)
			if err == nil {
				t.Fatalf("expected an error for type %s, got nil", fieldType)
			}
			if !strings.Contains(err.Error(), "struct User, field Tags") {
				t.Errorf("error missing struct/field context: %v", err)
			}
		})
	}
}

// --- contains / excludes --------------------------------------------------

func TestNewConstraintForContains(t *testing.T) {
	got := getConstraintForContains("Name", "abc")
	assertRawConstraint(t, got, `!strings.Contains(s.Name, "abc")`, "name should contain abc")
}

func TestNewConstraintForExcludes(t *testing.T) {
	got := getConstraintForExcludes("Name", "abc")
	assertRawConstraint(t, got, `strings.Contains(s.Name, "abc")`, "name should not contain abc")
}

func TestNewConstraintForContainsQuotesSpecialCharacters(t *testing.T) {
	got := getConstraintForContains("Name", `a"b`)
	assertRawConstraint(t, got, `!strings.Contains(s.Name, "a\"b")`, `name should contain a"b`)
}

// --- named formats --------------------------------------------------------

func TestNewConstraintForNamedFormats(t *testing.T) {
	tests := []struct {
		name        string
		got         constraint
		wantField   string
		wantErr     string
	}{
		{
			name:      "email",
			got:       getConstraintForEmail("Account", "Email"),
			wantField: "!AccountEmailEmailRegex.MatchString(s.Email)",
			wantErr:   "email is not a valid email",
		},
		{
			name:      "url",
			got:       getConstraintForURL("Page", "Site"),
			wantField: "!PageSiteURLRegex.MatchString(s.Site)",
			wantErr:   "site is not a valid url",
		},
		{
			name:      "uuid",
			got:       getConstraintForUUID("Record", "ID"),
			wantField: "!RecordIDUUIDRegex.MatchString(s.ID)",
			wantErr:   "id is not a valid uuid",
		},
		{
			name:      "ip",
			got:       getConstraintForIP("Server", "Addr"),
			wantField: "!ServerAddrIPRegex.MatchString(s.Addr)",
			wantErr:   "addr is not a valid ip address",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got.FieldName != tc.wantField {
				t.Errorf("FieldName = %q, want %q", tc.got.FieldName, tc.wantField)
			}
			if tc.got.Error != tc.wantErr {
				t.Errorf("Error = %q, want %q", tc.got.Error, tc.wantErr)
			}
			if tc.got.Raw != "" {
				t.Errorf("Raw should be empty (uses pre-compiled var), got %q", tc.got.Raw)
			}
		})
	}
}

func TestNewNamedFormatPatternsCompile(t *testing.T) {
	for name, pattern := range map[string]string{
		"email": emailPattern,
		"url":   urlPattern,
		"uuid":  uuidPattern,
		"ip":    ipPattern,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := regexp.Compile(pattern); err != nil {
				t.Errorf("pattern %q doesn't compile: %v", pattern, err)
			}
		})
	}
}

// --- cross-field ----------------------------------------------------------

func TestNewConstraintForCrossFields(t *testing.T) {
	tests := []struct {
		name    string
		got     constraint
		wantRaw string
		wantErr string
	}{
		{
			name:    "nefield fails on equality",
			got:     getConstraintForNeField("Password", "Username"),
			wantRaw: "s.Password == s.Username",
			wantErr: "password should not be equal to username",
		},
		{
			name:    "gtfield fails on less-or-equal",
			got:     getConstraintForGtField("End", "Start"),
			wantRaw: "s.End <= s.Start",
			wantErr: "end should be greater than start",
		},
		{
			name:    "ltfield fails on greater-or-equal",
			got:     getConstraintForLtField("Start", "End"),
			wantRaw: "s.Start >= s.End",
			wantErr: "start should be less than end",
		},
		{
			name:    "gtefield fails on less-than",
			got:     getConstraintForGteField("End", "Start"),
			wantRaw: "s.End < s.Start",
			wantErr: "end should be greater than or equal to start",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertRawConstraint(t, tc.got, tc.wantRaw, tc.wantErr)
		})
	}
}

// --- conditional required -------------------------------------------------

func TestNewConstraintForRequiredWith(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantRaw string
	}{
		{
			name:    "single other field",
			value:   "Kind",
			wantRaw: `(s.Kind != "") && s.Name == ""`,
		},
		{
			name:    "multiple other fields are ANDed",
			value:   "'Kind,Role'",
			wantRaw: `(s.Kind != "" && s.Role != "") && s.Name == ""`,
		},
		{
			name:    "no other fields never fires",
			value:   "",
			wantRaw: "false",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := getConstraintForRequiredWith("Name", tc.value)
			assertRawConstraint(t, got, tc.wantRaw, "name can't be blank")
		})
	}
}

func TestNewConstraintForRequiredWithout(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantRaw string
	}{
		{
			name:    "single other field",
			value:   "Kind",
			wantRaw: `(s.Kind == "") && s.Name == ""`,
		},
		{
			name:    "multiple other fields are ANDed",
			value:   "'Kind,Role'",
			wantRaw: `(s.Kind == "" && s.Role == "") && s.Name == ""`,
		},
		{
			name:    "no other fields never fires",
			value:   "",
			wantRaw: "false",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := getConstraintForRequiredWithout("Name", tc.value)
			assertRawConstraint(t, got, tc.wantRaw, "name can't be blank")
		})
	}
}

func TestNewConstraintForRequiredIf(t *testing.T) {
	tests := []struct {
		name      string
		fieldType string
		value     string
		wantRaw   string
	}{
		{
			name:      "string field with string condition",
			fieldType: "string",
			value:     "Kind=admin",
			wantRaw:   `s.Kind == "admin" && s.Name == ""`,
		},
		{
			name:      "numeric condition value is a bare literal",
			fieldType: "string",
			value:     "Age=18",
			wantRaw:   `s.Age == 18 && s.Name == ""`,
		},
		{
			name:      "numeric field zero check",
			fieldType: "int",
			value:     "Kind=admin",
			wantRaw:   `s.Kind == "admin" && s.Name == 0`,
		},
		{
			name:      "bool field zero check",
			fieldType: "bool",
			value:     "Kind=admin",
			wantRaw:   `s.Kind == "admin" && s.Name == false`,
		},
		{
			name:      "slice field zero check",
			fieldType: "[]string",
			value:     "Kind=admin",
			wantRaw:   `s.Kind == "admin" && s.Name == nil`,
		},
		{
			name:      "pointer field zero check",
			fieldType: "*User",
			value:     "Kind=admin",
			wantRaw:   `s.Kind == "admin" && s.Name == nil`,
		},
		{
			name:      "only the first equals sign splits the value",
			fieldType: "string",
			value:     "Kind=a=b",
			wantRaw:   `s.Kind == "a=b" && s.Name == ""`,
		},
		{
			name:      "surrounding single quotes are stripped",
			fieldType: "string",
			value:     "'Kind=admin'",
			wantRaw:   `s.Kind == "admin" && s.Name == ""`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := getConstraintForRequiredIf("User", "Name", tc.fieldType, tc.value)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertRawConstraint(t, got, tc.wantRaw, "name can't be blank")
		})
	}
}

func TestNewConstraintForRequiredIfRejectsBadValues(t *testing.T) {
	tests := []struct {
		name      string
		fieldType string
		value     string
	}{
		{name: "missing equals sign", fieldType: "string", value: "Kind"},
		{name: "empty value", fieldType: "string", value: ""},
		{name: "missing condition field", fieldType: "string", value: "=admin"},
		{name: "unsupported field type", fieldType: "pkg.User", value: "Kind=admin"},
		{name: "unsupported array field type", fieldType: "[3]int", value: "Kind=admin"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := getConstraintForRequiredIf("User", "Name", tc.fieldType, tc.value)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if !strings.Contains(err.Error(), "struct User, field Name") {
				t.Errorf("error missing struct/field context: %v", err)
			}
		})
	}
}

// --- nested validate -----------------------------------------------------

func TestNewConstraintForValidate(t *testing.T) {
	tests := []struct {
		name      string
		fieldType string
		wantRaw   string
	}{
		{
			name:      "value type",
			fieldType: "Address",
			wantRaw:   "s.Addr.Validate() != nil",
		},
		{
			name:      "selector type",
			fieldType: "pkg.Address",
			wantRaw:   "s.Addr.Validate() != nil",
		},
		{
			name:      "pointer type is nil-checked first",
			fieldType: "*Address",
			wantRaw:   "s.Addr != nil && s.Addr.Validate() != nil",
		},
		{
			name:      "slice type is ranged over",
			fieldType: "[]Address",
			wantRaw:   "func() bool { for _, v := range s.Addr { if err := v.Validate(); err != nil { return true } }; return false }()",
		},
		{
			name:      "array type is ranged over",
			fieldType: "[3]Address",
			wantRaw:   "func() bool { for _, v := range s.Addr { if err := v.Validate(); err != nil { return true } }; return false }()",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := getConstraintForValidate("User", "Addr", tc.fieldType)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertRawConstraint(t, got, tc.wantRaw, "addr validation failed")
		})
	}
}

func TestNewConstraintForValidateUnsupportedType(t *testing.T) {
	_, err := getConstraintForValidate("User", "Addr", "map[string]Address")
	if err == nil {
		t.Fatal("expected an error for a map field, got nil")
	}
	if !strings.Contains(err.Error(), "struct User, field Addr") {
		t.Errorf("error missing struct/field context: %v", err)
	}
}

// --- helpers -------------------------------------------------------------

// assertRawConstraint checks that a constraint uses the Raw rendering path
// with exactly the expected expression and error message.
func assertRawConstraint(t *testing.T, got constraint, wantRaw, wantErr string) {
	t.Helper()

	if got.Raw != wantRaw {
		t.Errorf("Raw = %q, want %q", got.Raw, wantRaw)
	}
	if got.Error != wantErr {
		t.Errorf("Error = %q, want %q", got.Error, wantErr)
	}
	assertOnlyRawSet(t, got)
}

// assertOnlyRawSet guards the template contract: when Raw is set, the
// FieldName/Op/Value fields must be empty, since the template ignores them.
func assertOnlyRawSet(t *testing.T, got constraint) {
	t.Helper()

	if got.Raw == "" {
		t.Fatalf("Raw is empty; constraint would render via the FieldName/Op/Value path: %+v", got)
	}
	if got.FieldName != "" || got.Op != "" || got.Value != "" {
		t.Errorf("FieldName/Op/Value should be empty when Raw is set, got %+v", got)
	}

	// Raw is dropped verbatim into `if <Raw> { ... }`, so it must at least
	// parse as a Go expression. This catches malformed generated code here
	// instead of as an opaque failure when the generated file is compiled.
	if _, err := parser.ParseExpr(got.Raw); err != nil {
		t.Errorf("Raw %q is not a valid Go expression: %v", got.Raw, err)
	}
}
