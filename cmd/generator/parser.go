package main

import (
	"fmt"
	"go/ast"
	"reflect"
	"strings"
)

type structField struct {
	Name string
	// Type is a rendered, human readable representation of the field's Go
	// type, e.g. "string", "[]*User", "map[string]int", "*pkg.User",
	// "[3]int". Constraint generation inspects this string (its prefix, in
	// particular) to tell slices, arrays, maps, pointers and qualified
	// (selector) types apart.
	Type string
	Tag  string
}

func parseNode(node ast.Node) (map[string][]structField, error) {
	structs := make(map[string][]structField)

	var err error
	ast.Inspect(node, func(n ast.Node) bool {
		if err != nil {
			return false
		}

		t, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		s, ok := t.Type.(*ast.StructType)
		if !ok {
			return true
		}

		var stFields []structField
		stFields, err = parseStructSpec(t.Name.String(), s)
		if err != nil {
			return false
		}
		if len(stFields) != 0 {
			structs[t.Name.String()] = stFields
		}
		return true
	})
	if err != nil {
		return nil, err
	}

	return structs, nil
}

// parseStructSpec parses the struct and returns the fields carrying a
// `validate` tag.
func parseStructSpec(structName string, s *ast.StructType) ([]structField, error) {
	stFields := []structField{}
	for _, f := range s.Fields.List {
		if f.Tag == nil {
			continue
		}
		// Get `validate` tag. If it doesn't have that field, ignore that field
		tag := reflect.StructTag(strings.Replace(f.Tag.Value, "`", "", -1)).Get("validate")
		if tag == "" {
			continue
		}

		name, err := fieldName(f)
		if err != nil {
			return nil, fmt.Errorf("struct %s: %w", structName, err)
		}

		typeStr, err := renderFieldType(f.Type)
		if err != nil {
			return nil, fmt.Errorf("struct %s, field %s: %w", structName, name, err)
		}

		stFields = append(stFields, structField{
			Name: name,
			Tag:  tag,
			Type: typeStr,
		})
	}
	return stFields, nil
}

// fieldName returns a field's name, safely handling embedded (anonymous)
// fields instead of blindly indexing f.Names[0], which panics for them.
// gandalf has no way to address an embedded field by name in generated
// code, so a `validate` tag on an embedded field is rejected with a
// descriptive error rather than silently ignored or mishandled.
func fieldName(f *ast.Field) (string, error) {
	if len(f.Names) == 0 {
		return "", fmt.Errorf("validate tags on embedded (anonymous) fields are not supported")
	}
	return f.Names[0].Name, nil
}

// renderFieldType walks the AST expression describing a field's type and
// renders it to a string, without ever performing an unchecked type
// assertion. Slices, arrays, maps, pointers and selector (package-qualified)
// types are all handled explicitly and recursively, so element/key/value
// types can themselves be pointers, selectors, or further nested
// slices/arrays. Expressions gandalf doesn't understand (inline struct,
// interface, func, channel types, etc.) produce a descriptive error instead
// of panicking.
func renderFieldType(expr ast.Expr) (string, error) {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name, nil

	case *ast.StarExpr:
		elt, err := renderFieldType(t.X)
		if err != nil {
			return "", err
		}
		return "*" + elt, nil

	case *ast.SelectorExpr:
		pkgIdent, ok := t.X.(*ast.Ident)
		if !ok {
			return "", fmt.Errorf("unsupported qualified type expression of type %T", t.X)
		}
		return fmt.Sprintf("%s.%s", pkgIdent.Name, t.Sel.Name), nil

	case *ast.ArrayType:
		elt, err := renderFieldType(t.Elt)
		if err != nil {
			return "", err
		}
		if t.Len == nil {
			return "[]" + elt, nil
		}
		return "[" + arrayLen(t.Len) + "]" + elt, nil

	case *ast.MapType:
		key, err := renderFieldType(t.Key)
		if err != nil {
			return "", err
		}
		val, err := renderFieldType(t.Value)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("map[%s]%s", key, val), nil

	default:
		return "", fmt.Errorf("unsupported field type: %T", expr)
	}
}

// arrayLen best-effort renders an array type's length expression. The
// result is only ever used for error messages and the human readable Type
// string (never for generated code), so falling back to "N" for lengths
// that aren't a simple literal (e.g. a named constant) is safe.
func arrayLen(expr ast.Expr) string {
	switch l := expr.(type) {
	case *ast.BasicLit:
		return l.Value
	case *ast.Ident:
		return l.Name
	default:
		return "N"
	}
}
