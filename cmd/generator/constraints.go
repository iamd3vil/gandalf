package main

import (
	"fmt"
	"strings"
)

// numericFieldTypes lists every built-in Go numeric type name (plus the
// "byte" and "rune" aliases) whose zero value can be compared directly
// against the untyped constant 0. complex64/complex128 are deliberately
// included here: Go allows comparing a complex value with the untyped
// constant 0 (e.g. `s.Field == 0`), so they're treated the same as any
// other numeric type for "required".
var numericFieldTypes = map[string]bool{
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true, "uintptr": true,
	"float32": true, "float64": true,
	"complex64": true, "complex128": true,
	"byte": true, "rune": true,
}

// typeCategory classifies a rendered field type string (see
// renderFieldType in parser.go) into the shape that matters for picking a
// safe constraint comparison. It relies on the fact that renderFieldType
// always produces one of a small set of unambiguous prefixes for
// slices/arrays/maps/pointers, falling back to a "selector" for any
// remaining package-qualified identifier (e.g. "pkg.User") and "ident" for
// a plain identifier (built-in or locally defined type name).
func typeCategory(typ string) string {
	switch {
	case strings.HasPrefix(typ, "[]"):
		return "slice"
	case strings.HasPrefix(typ, "["):
		return "array"
	case strings.HasPrefix(typ, "map["):
		return "map"
	case strings.HasPrefix(typ, "*"):
		return "pointer"
	case strings.Contains(typ, "."):
		return "selector"
	default:
		return "ident"
	}
}

// getConstraints turns every `validate` tag on fields into the constraints
// used to render the generated Validate() method. Constraint parsing errors
// (a malformed tag, an unknown constraint, or a constraint that isn't
// supported for the field's type) are returned with struct and field
// context rather than silently ignored or panicking.
func getConstraints(structName string, fields []structField) ([]constraint, error) {
	cs := []constraint{}
	for _, f := range fields {
		tagConstraints, err := parseValidateTag(structName, f.Name, f.Tag)
		if err != nil {
			return nil, err
		}

		for _, tc := range tagConstraints {
			var cons constraint
			switch tc.Name {
			case "required":
				cons, err = getConstraintForRequired(structName, f.Name, f.Type)
			case "min":
				cons, err = getConstraintForMin(structName, f.Name, f.Type, tc.Value)
			case "mineq":
				cons, err = getConstraintForMinEq(structName, f.Name, f.Type, tc.Value)
			case "max":
				cons, err = getConstraintForMax(structName, f.Name, f.Type, tc.Value)
			case "maxeq":
				cons, err = getConstraintForMaxEq(structName, f.Name, f.Type, tc.Value)
			case "eqfield":
				cons = getConstraintForEqField(f.Name, tc.Value)
			case "contains":
				cons = getConstraintForContains(f.Name, tc.Value)
			case "email":
				cons = getConstraintForEmail(structName, f.Name)
			case "eq":
				cons = getConstraintForEq(f.Name, tc.Value)
			case "excludes":
				cons = getConstraintForExcludes(f.Name, tc.Value)
			case "gtfield":
				cons = getConstraintForGtField(f.Name, tc.Value)
			case "gtefield":
				cons = getConstraintForGteField(f.Name, tc.Value)
			case "ip":
				cons = getConstraintForIP(structName, f.Name)
			case "len":
				cons, err = getConstraintForLen(f.Name, f.Type, tc.Value)
			case "ltfield":
				cons = getConstraintForLtField(f.Name, tc.Value)
			case "ne":
				cons = getConstraintForNe(f.Name, tc.Value)
			case "nefield":
				cons = getConstraintForNeField(f.Name, tc.Value)
			case "oneof":
				cons = getConstraintForOneOf(f.Name, tc.Value)
			case "required_if":
				cons, err = getConstraintForRequiredIf(structName, f.Name, f.Type, tc.Value)
			case "required_with":
				cons = getConstraintForRequiredWith(f.Name, tc.Value)
			case "required_without":
				cons = getConstraintForRequiredWithout(f.Name, tc.Value)
			case "unique":
				cons, err = getConstraintForUnique(structName, f.Name, f.Type)
			case "url":
				cons = getConstraintForURL(structName, f.Name)
			case "uuid":
				cons = getConstraintForUUID(structName, f.Name)
			case "validate":
				cons, err = getConstraintForValidate(structName, f.Name, f.Type)
			case "regexp":
				if f.Type != "string" {
					err = fmt.Errorf("struct %s, field %s: regexp is not supported for type %s", structName, f.Name, f.Type)
					break
				}
				cons = getConstraintForRegex(f.Name, structName, tc.Value)
			default:
				err = fmt.Errorf("struct %s, field %s: unknown constraint %q", structName, f.Name, tc.Name)
			}
			if err != nil {
				return nil, err
			}
			cs = append(cs, cons)
		}
	}
	return cs, nil
}

func getConstraintForRequired(structName, name, typ string) (constraint, error) {
	c := constraint{
		FieldName: fmt.Sprintf("s.%s", name),
		Op:        "==",
		Error:     fmt.Sprintf("%s can't be blank", strings.ToLower(name)),
	}

	switch {
	case typ == "string":
		c.Value = "\"\""
		return c, nil
	case typ == "bool":
		c.Value = "false"
		return c, nil
	case numericFieldTypes[typ]:
		c.Value = "0"
		return c, nil
	}

	switch typeCategory(typ) {
	case "slice", "map", "pointer":
		c.Value = "nil"
		return c, nil
	case "array":
		return constraint{}, fmt.Errorf("struct %s, field %s: required is not supported for array type %s (arrays are never nil); use min/max on its length instead", structName, name, typ)
	case "selector":
		return constraint{}, fmt.Errorf("struct %s, field %s: required is not supported for qualified type %s; wrap it in a pointer (e.g. *%s) to check for nil", structName, name, typ, typ)
	default:
		return constraint{}, fmt.Errorf("struct %s, field %s: required is not supported for type %s", structName, name, typ)
	}
}

func getConstraintForMin(structName, name, typ, value string) (constraint, error) {
	return minMaxConstraint(structName, name, typ, value, "<", "min",
		fmt.Sprintf("%s can't be less than %s", strings.ToLower(name), value))
}

func getConstraintForMax(structName, name, typ, value string) (constraint, error) {
	return minMaxConstraint(structName, name, typ, value, ">", "max",
		fmt.Sprintf("%s can't be greater than %s", strings.ToLower(name), value))
}

func getConstraintForMinEq(structName, name, typ, value string) (constraint, error) {
	c, err := minMaxConstraint(structName, name, typ, value, "<=", "mineq",
		fmt.Sprintf("%s can't be less than %s", strings.ToLower(name), value))
	return c, err
}

func getConstraintForMaxEq(structName, name, typ, value string) (constraint, error) {
	c, err := minMaxConstraint(structName, name, typ, value, ">=", "maxeq",
		fmt.Sprintf("%s can't be greater than %s", strings.ToLower(name), value))
	return c, err
}

// minMaxConstraint builds the shared shape behind min/mineq/max/maxeq: a
// direct comparison for strings (by length) and numeric types, a len()
// comparison for slices/arrays/maps, and a descriptive error for any type
// it isn't safe to compare this way (pointers, selectors, bools, and any
// other unrecognised type).
func minMaxConstraint(structName, name, typ, value, op, constraintName, errMsg string) (constraint, error) {
	c := constraint{
		FieldName: fmt.Sprintf("s.%s", name),
		Op:        op,
		Value:     value,
		Error:     errMsg,
	}

	if typ == "string" {
		c.FieldName = fmt.Sprintf("len(s.%s)", name)
		return c, nil
	}
	if numericFieldTypes[typ] {
		return c, nil
	}

	switch typeCategory(typ) {
	case "slice", "array", "map":
		c.FieldName = fmt.Sprintf("len(s.%s)", name)
		return c, nil
	case "pointer":
		return constraint{}, fmt.Errorf("struct %s, field %s: %s is not supported for pointer type %s", structName, name, constraintName, typ)
	case "selector":
		return constraint{}, fmt.Errorf("struct %s, field %s: %s is not supported for qualified type %s", structName, name, constraintName, typ)
	default:
		return constraint{}, fmt.Errorf("struct %s, field %s: %s is not supported for type %s", structName, name, constraintName, typ)
	}
}

func getConstraintForEqField(name, value string) constraint {
	c := constraint{
		FieldName: fmt.Sprintf("s.%s", name),
		Op:        "!=",
		Value:     fmt.Sprintf("s.%s", value),
		Error:     fmt.Sprintf("%s should be equal to %s", name, value),
	}

	return c
}

func getConstraintForRegex(fieldName, structName string, regexp string) constraint {
	c := constraint{
		FieldName: fmt.Sprintf("!%s%sRegex.MatchString(s.%s)", structName, fieldName, fieldName),
		Error:     fmt.Sprintf("%s doesn't match given regex", fieldName),
	}
	return c
}
