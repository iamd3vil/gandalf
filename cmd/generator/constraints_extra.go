package main

import (
	"fmt"
	"strconv"
	"strings"
)

// This file holds the constraints that don't fit the simple
// <FieldName> <Op> <Value> shape used by constraints.go. They all populate
// constraint.Raw with a complete Go boolean expression describing the
// *failure* condition, which the template renders as
// `if <Raw> { return errors.New("<Error>") }`.
//
// Every Raw expression here is a single self-contained expression (using an
// immediately invoked func literal where a loop is needed) so that it can be
// dropped into that `if` unchanged.

// isNumericLiteral reports whether a constraint value from a struct tag
// should be rendered as a bare Go literal (a number) rather than a quoted
// string. A leading digit or sign is treated as numeric; anything else is a
// string. This deliberately doesn't fully validate the number: a malformed
// literal such as "1.2.3" surfaces as a compile error in the generated code
// rather than being silently quoted into a wrong-but-valid comparison.
func isNumericLiteral(value string) bool {
	if value == "" {
		return false
	}
	switch c := value[0]; {
	case c == '-' || c == '+':
		return true
	case c >= '0' && c <= '9':
		return true
	default:
		return false
	}
}

// literalForValue renders a tag value as a Go literal: numeric-looking
// values verbatim, everything else as a quoted string. strconv.Quote is used
// so that backslashes, quotes and other special characters survive
// round-tripping into the generated source.
func literalForValue(value string) string {
	if isNumericLiteral(value) {
		return value
	}
	return strconv.Quote(value)
}

// unquoteTagValue strips a single pair of surrounding single quotes from a
// tag value. parseValidateTag splits constraints on whitespace, so a
// multi-word value (e.g. a oneof list) has to be written as
// `oneof:'admin user editor'`; this is where those quotes come off. Values
// without surrounding quotes are returned unchanged.
func unquoteTagValue(value string) string {
	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		return value[1 : len(value)-1]
	}
	return value
}

// zeroCheckExpr returns a Go expression that is true when the field holds
// its zero value, for the types where such a check is safe. The bool result
// reports whether the type is supported.
func zeroCheckExpr(fieldName, fieldType string) (string, bool) {
	switch {
	case fieldType == "string":
		return fmt.Sprintf("s.%s == \"\"", fieldName), true
	case fieldType == "bool":
		return fmt.Sprintf("s.%s == false", fieldName), true
	case numericFieldTypes[fieldType]:
		return fmt.Sprintf("s.%s == 0", fieldName), true
	}

	switch typeCategory(fieldType) {
	case "slice", "map", "pointer":
		return fmt.Sprintf("s.%s == nil", fieldName), true
	default:
		return "", false
	}
}

// splitCommaList splits a comma-separated list of values, trimming
// whitespace around each element and discarding empty segments (so a
// trailing comma is harmless). Compatible with the parser: because
// parseValidateTag splits constraints on whitespace, multi-value
// constraints MUST use commas rather than spaces (e.g.
// `oneof:admin,user,editor`).
func splitCommaList(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// --- value equality: eq, ne, oneof -----------------------------------------

// getConstraintForEq builds the constraint for `eq:<value>`: the field must
// equal value, so the failure condition is inequality.
func getConstraintForEq(fieldName, value string) constraint {
	return constraint{
		Raw:   fmt.Sprintf("s.%s != %s", fieldName, literalForValue(value)),
		Error: fmt.Sprintf("%s should be %s", strings.ToLower(fieldName), value),
	}
}

// getConstraintForNe builds the constraint for `ne:<value>`: the field must
// not equal value, so the failure condition is equality.
func getConstraintForNe(fieldName, value string) constraint {
	return constraint{
		Raw:   fmt.Sprintf("s.%s == %s", fieldName, literalForValue(value)),
		Error: fmt.Sprintf("%s should not be %s", strings.ToLower(fieldName), value),
	}
}

// getConstraintForOneOf builds the constraint for `oneof:'a,b,c'`: the field
// must equal one of the comma-separated values, so the failure condition is
// that it matches none of them. Values may be single-quote-wrapped (e.g.
// `oneof:'San Francisco, CA',Austin`) to include commas in the value itself.
//
// Only string fields are supported: every value is rendered with
// strconv.Quote.
func getConstraintForOneOf(fieldName, value string) constraint {
	value = unquoteTagValue(value)
	values := splitCommaList(value)

	// No allowed values means there is nothing to check; emit a condition
	// that never fires rather than an empty Raw (which would make the
	// template fall back to the FieldName/Op/Value branch and render
	// invalid Go).
	if len(values) == 0 {
		return constraint{
			Raw:   "false",
			Error: fmt.Sprintf("%s should be one of %s", strings.ToLower(fieldName), value),
		}
	}

	checks := make([]string, 0, len(values))
	for _, v := range values {
		checks = append(checks, fmt.Sprintf("s.%s != %s", fieldName, strconv.Quote(v)))
	}

	return constraint{
		Raw:   strings.Join(checks, " && "),
		Error: fmt.Sprintf("%s should be one of %s", strings.ToLower(fieldName), strings.Join(values, ", ")),
	}
}

// --- exact length: len -----------------------------------------------------

// getConstraintForLen builds the constraint for `len:<n>`: the field's length
// must be exactly n. Supported for the types that have a length (strings,
// slices, arrays, maps).
func getConstraintForLen(fieldName, fieldType, value string) (constraint, error) {
	if fieldType != "string" {
		switch typeCategory(fieldType) {
		case "slice", "array", "map":
		default:
			return constraint{}, fmt.Errorf("len is not supported for type %s", fieldType)
		}
	}

	return constraint{
		Raw:   fmt.Sprintf("len(s.%s) != %s", fieldName, value),
		Error: fmt.Sprintf("%s length should be %s", strings.ToLower(fieldName), value),
	}, nil
}

// --- slice uniqueness: unique ----------------------------------------------

// getConstraintForUnique builds the constraint for `unique`: a slice field
// must not contain duplicate values. The Raw expression is an immediately
// invoked func literal so that the duplicate-tracking map and loop fit in a
// single expression.
//
// Limitation: the tracking map is keyed by string, so `unique` currently only
// works on []string (and named string types). Applying it to a slice of any
// other element type produces a generated file that doesn't compile; a future
// version should key the map by the real element type.
func getConstraintForUnique(structName, fieldName, fieldType string) (constraint, error) {
	if typeCategory(fieldType) != "slice" {
		return constraint{}, fmt.Errorf("struct %s, field %s: unique is not supported for type %s (only slices)", structName, fieldName, fieldType)
	}

	raw := fmt.Sprintf("func() bool { seen := make(map[string]bool, len(s.%s)); for _, v := range s.%s { if seen[v] { return true }; seen[v] = true }; return false }()",
		fieldName, fieldName)

	return constraint{
		Raw:   raw,
		Error: fmt.Sprintf("%s contains duplicate values", strings.ToLower(fieldName)),
	}, nil
}

// --- substring checks: contains, excludes ----------------------------------

// getConstraintForContains builds the constraint for `contains:<value>`: a
// string field must contain value as a substring.
//
// The generated code calls strings.Contains, so the generated file needs the
// "strings" import.
func getConstraintForContains(fieldName, value string) constraint {
	return constraint{
		Raw:   fmt.Sprintf("!strings.Contains(s.%s, %s)", fieldName, strconv.Quote(unquoteTagValue(value))),
		Error: fmt.Sprintf("%s should contain %s", strings.ToLower(fieldName), value),
	}
}

// getConstraintForExcludes builds the constraint for `excludes:<value>`: a
// string field must not contain value as a substring.
func getConstraintForExcludes(fieldName, value string) constraint {
	return constraint{
		Raw:   fmt.Sprintf("strings.Contains(s.%s, %s)", fieldName, strconv.Quote(unquoteTagValue(value))),
		Error: fmt.Sprintf("%s should not contain %s", strings.ToLower(fieldName), value),
	}
}

// --- named formats: email, url, uuid, ip -----------------------------------
//
// These follow the same pre-compiled-var pattern as the `regexp:` constraint:
// getRegexes() (in regexes.go) collects a Regex entry per named format per
// field, and the template emits a package-level `var XxxYyyFormatRegex =
// regexp.MustCompile(...)`. Each constraint's FieldName references that var,
// so the regex is compiled exactly once at package init, not on every
// Validate() call.

func getConstraintForEmail(structName, fieldName string) constraint {
	return constraint{
		FieldName: fmt.Sprintf("!%s%sEmailRegex.MatchString(s.%s)", structName, fieldName, fieldName),
		Error:     fmt.Sprintf("%s is not a valid email", strings.ToLower(fieldName)),
	}
}

func getConstraintForURL(structName, fieldName string) constraint {
	return constraint{
		FieldName: fmt.Sprintf("!%s%sURLRegex.MatchString(s.%s)", structName, fieldName, fieldName),
		Error:     fmt.Sprintf("%s is not a valid url", strings.ToLower(fieldName)),
	}
}

func getConstraintForUUID(structName, fieldName string) constraint {
	return constraint{
		FieldName: fmt.Sprintf("!%s%sUUIDRegex.MatchString(s.%s)", structName, fieldName, fieldName),
		Error:     fmt.Sprintf("%s is not a valid uuid", strings.ToLower(fieldName)),
	}
}

func getConstraintForIP(structName, fieldName string) constraint {
	return constraint{
		FieldName: fmt.Sprintf("!%s%sIPRegex.MatchString(s.%s)", structName, fieldName, fieldName),
		Error:     fmt.Sprintf("%s is not a valid ip address", strings.ToLower(fieldName)),
	}
}

// --- cross-field comparisons ----------------------------------------------

// crossFieldConstraint builds the shared shape behind the *field
// constraints: comparing two fields of the same struct. failureOp is the
// operator that makes the comparison *fail*.
func crossFieldConstraint(fieldName, otherField, failureOp, description string) constraint {
	return constraint{
		Raw: fmt.Sprintf("s.%s %s s.%s", fieldName, failureOp, otherField),
		Error: fmt.Sprintf("%s should %s %s", strings.ToLower(fieldName), description,
			strings.ToLower(otherField)),
	}
}

// getConstraintForNeField builds the constraint for `nefield:<other>`: the
// two fields must differ, so equality is the failure.
func getConstraintForNeField(fieldName, otherField string) constraint {
	return crossFieldConstraint(fieldName, otherField, "==", "not be equal to")
}

// getConstraintForGtField builds the constraint for `gtfield:<other>`: the
// field must be strictly greater, so `<=` is the failure.
func getConstraintForGtField(fieldName, otherField string) constraint {
	return crossFieldConstraint(fieldName, otherField, "<=", "be greater than")
}

// getConstraintForLtField builds the constraint for `ltfield:<other>`: the
// field must be strictly less, so `>=` is the failure.
func getConstraintForLtField(fieldName, otherField string) constraint {
	return crossFieldConstraint(fieldName, otherField, ">=", "be less than")
}

// getConstraintForGteField builds the constraint for `gtefield:<other>`: the
// field must be greater than or equal, so `<` is the failure.
func getConstraintForGteField(fieldName, otherField string) constraint {
	return crossFieldConstraint(fieldName, otherField, "<", "be greater than or equal to")
}

// --- conditional required: required_with, required_without, required_if ----

// getConstraintForRequiredWith builds the constraint for
// `required_with:'A B'`: this field is required when all the named fields are
// set (non-zero).
//
// Limitation: the emitted zero checks assume string fields (both for this
// field and for the named ones), since only field names - not their types -
// are available here.
func getConstraintForRequiredWith(fieldName, value string) constraint {
	return conditionalRequired(fieldName, value, "!=")
}

// getConstraintForRequiredWithout builds the constraint for
// `required_without:'A B'`: this field is required when all the named fields
// are unset (zero). Same string-field limitation as required_with.
func getConstraintForRequiredWithout(fieldName, value string) constraint {
	return conditionalRequired(fieldName, value, "==")
}

// conditionalRequired builds the shared shape behind
// required_with/required_without. othersOp is the operator applied to each
// named field against "" - `!=` for "is set" (required_with) and `==` for
// "is unset" (required_without). Fields are comma-separated.
func conditionalRequired(fieldName, value, othersOp string) constraint {
	others := splitCommaList(unquoteTagValue(value))

	c := constraint{
		Error: fmt.Sprintf("%s can't be blank", strings.ToLower(fieldName)),
	}

	// With no named fields there is no condition that could trigger the
	// requirement, so emit a condition that never fires.
	if len(others) == 0 {
		c.Raw = "false"
		return c
	}

	checks := make([]string, 0, len(others))
	for _, o := range others {
		checks = append(checks, fmt.Sprintf("s.%s %s \"\"", o, othersOp))
	}

	c.Raw = fmt.Sprintf("(%s) && s.%s == \"\"", strings.Join(checks, " && "), fieldName)
	return c
}

// getConstraintForRequiredIf builds the constraint for
// `required_if:Field=value`: this field is required when Field equals value.
//
// The condition field's type isn't known here, so its expected value is
// rendered with the same numeric/string heuristic as eq/ne. This field's own
// zero check does use its real type.
func getConstraintForRequiredIf(structName, fieldName, fieldType, value string) (constraint, error) {
	value = unquoteTagValue(value)

	condField, condValue, found := strings.Cut(value, "=")
	if !found {
		return constraint{}, fmt.Errorf("struct %s, field %s: required_if expects a value of the form Field=value, got %q", structName, fieldName, value)
	}
	if condField == "" {
		return constraint{}, fmt.Errorf("struct %s, field %s: required_if is missing the condition field name in %q", structName, fieldName, value)
	}

	zeroCheck, ok := zeroCheckExpr(fieldName, fieldType)
	if !ok {
		return constraint{}, fmt.Errorf("struct %s, field %s: required_if is not supported for type %s", structName, fieldName, fieldType)
	}

	return constraint{
		Raw:   fmt.Sprintf("s.%s == %s && %s", condField, literalForValue(condValue), zeroCheck),
		Error: fmt.Sprintf("%s can't be blank", strings.ToLower(fieldName)),
	}, nil
}

// --- nested validation: validate ------------------------------------------

// getConstraintForValidate builds the constraint for `validate`: the field's
// own Validate() method must pass. Value and pointer fields are called
// directly (pointers are nil-checked first, so a nil field is skipped);
// slices and arrays are ranged over.
//
// Limitation: the current template renders constraints as
// `if <Raw> { return errors.New("<Error>") }`, so the nested error's own
// message is discarded and replaced by a generic one. Propagating the sub
// error requires template support for a dedicated nested-validation block.
func getConstraintForValidate(structName, fieldName, fieldType string) (constraint, error) {
	c := constraint{
		Error: fmt.Sprintf("%s validation failed", strings.ToLower(fieldName)),
	}

	switch typeCategory(fieldType) {
	case "ident", "selector":
		c.Raw = fmt.Sprintf("s.%s.Validate() != nil", fieldName)
	case "pointer":
		c.Raw = fmt.Sprintf("s.%s != nil && s.%s.Validate() != nil", fieldName, fieldName)
	case "slice", "array":
		c.Raw = fmt.Sprintf("func() bool { for _, v := range s.%s { if err := v.Validate(); err != nil { return true } }; return false }()", fieldName)
	default:
		return constraint{}, fmt.Errorf("struct %s, field %s: validate is not supported for type %s", structName, fieldName, fieldType)
	}

	return c, nil
}
