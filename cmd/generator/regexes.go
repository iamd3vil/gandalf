package main

import "strings"

type Regex struct {
	StructName string
	FieldName  string
	Regex      string
}

func getRegexes(structName string, fields []structField) []Regex {
	r := []Regex{}

	for _, f := range fields {
		conditions := strings.Split(f.Tag, " ")
		for _, cond := range conditions {
			c := strings.Split(cond, ":")
			switch c[0] {
			case "regexp":
				reg := Regex{
					StructName: structName,
					FieldName:  f.Name,
					Regex:      strings.Trim(c[1], "'"),
				}
				r = append(r, reg)
			}
		}
	}
	return r
}
