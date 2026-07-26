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
//
// Named format constraints (email, url, uuid, ip) are also collected here
// so that their pre-compiled regexp vars live alongside the user-defined
// `regexp:` vars, avoiding per-call recompilation.
func getRegexes(structName string, fields []structField) ([]Regex, error) {
	r := []Regex{}

	for _, f := range fields {
		constraints, err := parseValidateTag(structName, f.Name, f.Tag)
		if err != nil {
			return nil, err
		}
		for _, constraint := range constraints {
			switch constraint.Name {
			case "regexp":
				pattern := constraint.Value
				if len(pattern) >= 2 && pattern[0] == '\'' && pattern[len(pattern)-1] == '\'' {
					pattern = pattern[1 : len(pattern)-1]
				}
				if _, err := regexp.Compile(pattern); err != nil {
					return nil, fmt.Errorf("struct %s field %s: invalid regexp %q: %w", structName, f.Name, pattern, err)
				}
				r = append(r, Regex{
					StructName: structName,
					FieldName:  f.Name,
					Regex:      strconv.Quote(pattern),
				})

			case "email":
				r = append(r, Regex{
					StructName: structName,
					FieldName:  f.Name + "Email",
					Regex:      strconv.Quote(emailPattern),
				})
			case "url":
				r = append(r, Regex{
					StructName: structName,
					FieldName:  f.Name + "URL",
					Regex:      strconv.Quote(urlPattern),
				})
			case "uuid":
				r = append(r, Regex{
					StructName: structName,
					FieldName:  f.Name + "UUID",
					Regex:      strconv.Quote(uuidPattern),
				})
			case "ip":
				r = append(r, Regex{
					StructName: structName,
					FieldName:  f.Name + "IP",
					Regex:      strconv.Quote(ipPattern),
				})
			}
		}
	}
	return r, nil
}

// Patterns backing the named format constraints.
const (
	emailPattern = `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	urlPattern   = `^(https?|ftp)://[^\s/$.?#].[^\s]*$`
	uuidPattern  = `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`
	ipPattern    = `^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`
)
