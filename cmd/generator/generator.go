package main

import (
	"io"

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
}

func generateCodeForStructs(fs stuffbin.FileSystem, pkg string, structs map[string][]structField, dest io.Writer) error {
	tmplContext := make(map[string]interface{})
	tmplContext["Pkg"] = pkg
	tmplContext["BuildDate"] = buildDate
	tmplContext["BuildVersion"] = buildVersion

	// Check and aggregate all Regexes
	regexes := []Regex{}
	for name, fields := range structs {
		rs := getRegexes(name, fields)
		regexes = append(regexes, rs...)
	}

	tmplContext["Regexes"] = regexes

	sts := make([]structContext, 0)
	for name, fields := range structs {
		sctx := structContext{
			StructName:  name,
			Constraints: getConstraints(name, fields),
		}
		sts = append(sts, sctx)
	}

	tmplContext["Structs"] = sts

	return saveResource("struct", []string{"/templates/struct.tmpl"}, dest, tmplContext, fs)
}
