package main

import (
	"fmt"
	"strings"
)

func getConstraints(structName string, fields []structField) []constraint {
	cs := []constraint{}
	for _, f := range fields {
		conditions := strings.Split(f.Tag, " ")
		for _, cond := range conditions {
			c := strings.Split(cond, ":")
			switch c[0] {
			case "required":
				cons := getConstraintForRequired(f.Name, f.Type)
				cs = append(cs, cons)
			case "min":
				cons := getConstraintForMin(f.Name, f.Type, c[1])
				cs = append(cs, cons)
			case "mineq":
				cons := getConstraintForMinEq(f.Name, f.Type, c[1])
				cs = append(cs, cons)
			case "maxeq":
				cons := getConstraintForMaxEq(f.Name, f.Type, c[1])
				cs = append(cs, cons)
			case "max":
				cons := getConstraintForMax(f.Name, f.Type, c[1])
				cs = append(cs, cons)
			case "eqfield":
				cons := getConstraintForEqField(f.Name, c[1])
				cs = append(cs, cons)
			case "regexp":
				cons := getConstraintForRegex(f.Name, structName, c[1])
				cs = append(cs, cons)
			}
		}
	}
	return cs
}

func getConstraintForRequired(name, typ string) constraint {
	c := constraint{
		FieldName: fmt.Sprintf("s.%s", name),
		Op:        "==",
		Error:     fmt.Sprintf("%s can't be blank", strings.ToLower(name)),
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

func getConstraintForMin(name, typ, value string) constraint {
	c := constraint{
		FieldName: fmt.Sprintf("s.%s", name),
		Op:        "<",
		Value:     value,
		Error:     fmt.Sprintf("%s can't be less than %s", strings.ToLower(name), value),
	}
	switch typ {
	case "string":
		c.FieldName = fmt.Sprintf("len(s.%s)", name)
	}
	// Handle arrays
	if strings.HasPrefix(typ, "[]") || strings.HasPrefix(typ, "map") {
		c.FieldName = fmt.Sprintf("len(s.%s)", name)
	}
	return c
}

func getConstraintForMax(name, typ, value string) constraint {
	c := constraint{
		FieldName: fmt.Sprintf("s.%s", name),
		Op:        ">",
		Value:     value,
		Error:     fmt.Sprintf("%s can't be greather than %s", strings.ToLower(name), value),
	}
	switch typ {
	case "string":
		c.FieldName = fmt.Sprintf("len(s.%s)", name)
	}
	// Handle arrays
	if strings.HasPrefix(typ, "[]") || strings.HasPrefix(typ, "map") {
		c.FieldName = fmt.Sprintf("len(s.%s)", name)
	}
	return c
}

func getConstraintForMinEq(name, typ, value string) constraint {
	cons := getConstraintForMin(name, typ, value)
	cons.Op = "<="
	return cons
}

func getConstraintForMaxEq(name, typ, value string) constraint {
	cons := getConstraintForMax(name, typ, value)
	cons.Op = ">="
	return cons
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
