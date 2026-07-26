package main

import (
	"fmt"
	"regexp"
	"strconv"
)

// Regex holds a single field's regular expression constraint. Pattern is
// already a Go-safe quoted string literal (produced with strconv.Quote) so
// that templates can render it verbatim without re-quoting.
type Regex struct {
	StructName string
	FieldName  string
	Regex      string
}

// getRegexes collects and validates every `regexp` constraint declared on
// fields of the given struct, in field declaration order. Each pattern is
// compiled with regexp.Compile to catch invalid regular expressions during
// generation instead of at generated-package init time, and quoted with
// strconv.Quote so that backslashes, quotes, and other special characters
// survive round-tripping through the generated Go source.
func getRegexes(structName string, fields []structField) ([]Regex, error) {
	r := []Regex{}

	for _, f := range fields {
		constraints, err := parseValidateTag(structName, f.Name, f.Tag)
		if err != nil {
			return nil, err
		}
		for _, constraint := range constraints {
			if constraint.Name != "regexp" {
				continue
			}

			pattern := constraint.Value
			if len(pattern) >= 2 && pattern[0] == '\'' && pattern[len(pattern)-1] == '\'' {
				pattern = pattern[1 : len(pattern)-1]
			}
			if _, err := regexp.Compile(pattern); err != nil {
				return nil, fmt.Errorf("struct %s field %s: invalid regexp %q: %w", structName, f.Name, pattern, err)
			}

			reg := Regex{
				StructName: structName,
				FieldName:  f.Name,
				Regex:      strconv.Quote(pattern),
			}
			r = append(r, reg)
		}
	}
	return r, nil
}
