package main

import (
	"io"
	"sort"
	"strings"

	"github.com/knadh/stuffbin"
)

type structContext struct {
	StructName  string
	Constraints []constraint
}

type constraint struct {
	FieldName string
	Op        string
	Value     string
	Error     string
	// Raw is an optional Go expression for complex constraints that don't
	// fit the simple FieldName-Op-Value pattern (e.g. oneof, contains,
	// unique, validate, required_if). When non-empty, the template renders
	// it directly instead of assembling <FieldName> <Op> <Value>.
	Raw string
}

func generateCodeForStructs(fs stuffbin.FileSystem, pkg string, structs map[string][]structField, dest io.Writer) error {
	tmplContext := make(map[string]interface{})
	tmplContext["Pkg"] = pkg
	tmplContext["BuildDate"] = buildDate
	tmplContext["BuildVersion"] = buildVersion

	// Sort struct names so that generated regex declarations and Validate
	// methods always appear in the same, deterministic order regardless of
	// Go's randomized map iteration order.
	names := make([]string, 0, len(structs))
	for name := range structs {
		names = append(names, name)
	}
	sort.Strings(names)

	// Check and aggregate all Regexes
	regexes := []Regex{}
	for _, name := range names {
		rs, err := getRegexes(name, structs[name])
		if err != nil {
			return err
		}
		regexes = append(regexes, rs...)
	}

	tmplContext["Regexes"] = regexes

	sts := make([]structContext, 0, len(names))
	for _, name := range names {
		constraints, err := getConstraints(name, structs[name])
		if err != nil {
			return err
		}
		sts = append(sts, structContext{
			StructName:  name,
			Constraints: constraints,
		})
	}

	tmplContext["Structs"] = sts

	// The generated file only imports "strings" when at least one rendered
	// constraint expression actually uses it (contains/excludes), otherwise
	// the generated code wouldn't compile due to an unused import.
	// The same applies to "regexp": the named-format constraints (email,
	// url, uuid, ip) call regexp.MustCompile inline, so the import is
	// needed even when no `regexp:` tag produced a package-level Regex.
	hasStrings, hasRegexp := false, false
	for _, sc := range sts {
		for _, c := range sc.Constraints {
			if strings.Contains(c.Raw, "strings.") {
				hasStrings = true
			}
			if strings.Contains(c.Raw, "regexp.") {
				hasRegexp = true
			}
		}
	}
	tmplContext["HasStrings"] = hasStrings
	tmplContext["HasRegexp"] = hasRegexp

	return saveResource("struct", []string{"/templates/struct.tmpl"}, dest, tmplContext, fs)
}
