package main

import (
	"io"
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
}

func generateCodeForStructs(fs stuffbin.FileSystem, pkg string, structs map[string][]structField, dest io.Writer) error {
	tmplContext := make(map[string]interface{})
	tmplContext["Pkg"] = pkg

	sts := make([]structContext, 0)
	for name, fields := range structs {
		sctx := structContext{
			StructName:  name,
			Constraints: getConstraints(fields),
		}
		sts = append(sts, sctx)
	}

	tmplContext["Structs"] = sts

	return saveResource("struct", []string{"/templates/struct.tmpl"}, dest, tmplContext, fs)
}

func getConstraints(fields []structField) []constraint {
	cs := []constraint{}
	for _, f := range fields {
		conditions := strings.Split(f.Tag, " ")
		for _, cond := range conditions {
			c := strings.Split(cond, ":")
			switch c[0] {
			case "required":
				cons := getConstraintForRequired(f.Name, f.Type)
				cs = append(cs, cons)
			}
		}
	}
	return cs
}

func getConstraintForRequired(name, typ string) constraint {
	c := constraint{
		FieldName: name,
		Op:        "==",
	}

	switch typ {
	case "string":
		c.Value = "\"\""
	case "int", "int32", "int64", "int16", "float64", "float32":
		c.Value = "0"
	default:
		c.Value = "nil"
	}
	return c
}
