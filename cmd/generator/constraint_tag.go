package main

import (
	"fmt"
	"strings"
)

// tagConstraint is a single "name" or "name:value" condition extracted from
// a `validate` struct tag.
type tagConstraint struct {
	Name  string
	Value string
}

// constraintsRequiringValue lists every constraint name that must be
// followed by a ":value" segment. Constraints not listed here (currently
// only "required") take no value.
var constraintsRequiringValue = map[string]bool{
	"min":     true,
	"mineq":   true,
	"max":     true,
	"maxeq":   true,
	"eqfield": true,
	"regexp":  true,
}

// parseValidateTag is the single shared parser for `validate` struct tags,
// used by every consumer that needs to read constraints off a tag.
//
// Individual constraints are separated by whitespace using strings.Fields,
// so repeated spaces, tabs, etc. between constraints are tolerated rather
// than only a single literal " ". Each constraint's name and value are
// split only at the first colon via strings.SplitN(cond, ":", 2), so a
// value - such as a regular expression - may itself safely contain
// additional colons.
//
// structName and fieldName are used purely to attach struct/field context
// to the returned error; they don't affect parsing.
func parseValidateTag(structName, fieldName, tag string) ([]tagConstraint, error) {
	conditions := strings.Fields(tag)
	cs := make([]tagConstraint, 0, len(conditions))

	for _, cond := range conditions {
		parts := strings.SplitN(cond, ":", 2)
		name := parts[0]
		value := ""
		if len(parts) == 2 {
			value = parts[1]
		}

		if constraintsRequiringValue[name] && value == "" {
			return nil, fmt.Errorf("struct %s, field %s: constraint %q requires a value (e.g. %s:<value>)", structName, fieldName, name, name)
		}

		cs = append(cs, tagConstraint{Name: name, Value: value})
	}

	return cs, nil
}
