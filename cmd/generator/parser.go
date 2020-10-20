package main

import (
	"fmt"
	"go/ast"
	"reflect"
	"strings"
)

type structField struct {
	Name string `validate:"required min:3 max:5"`
	Type string
	Tag  string
}

func parseNode(node ast.Node) map[string][]structField {
	structs := make(map[string][]structField)

	ast.Inspect(node, func(n ast.Node) bool {
		switch t := n.(type) {
		case *ast.TypeSpec:
			switch t.Type.(type) {
			case *ast.StructType:
				s := t.Type.(*ast.StructType)
				fmt.Printf("parsing struct: %v\n", t.Name.String())
				stFields := parseStructSpec(t.Name.String(), s)
				if len(stFields) != 0 {
					structs[t.Name.String()] = stFields
				}
			}
		}
		return true

	})

	return structs
}

// parseStructSpec parses the struct and returns the fields
func parseStructSpec(structName string, s *ast.StructType) []structField {
	stFields := []structField{}
	for _, f := range s.Fields.List {
		var fieldType string
		switch f.Type.(type) {
		case *ast.Ident:
			fieldType = f.Type.(*ast.Ident).Name
		case *ast.ArrayType:
			arrayType := f.Type.(*ast.ArrayType)
			fieldType = fmt.Sprintf("[]%s", arrayType.Elt.(*ast.Ident).String())
		}
		if f.Tag != nil {
			tag := reflect.StructTag(strings.Replace(f.Tag.Value, "`", "", -1)).Get("validate")
			if tag != "" {
				stFields = append(stFields, structField{
					Name: f.Names[0].Name,
					Tag:  tag,
					Type: fieldType,
				})
			}
		}
	}
	return stFields
}
