package main

type TestStruct struct {
	Name   string   `validate:"required min:3 max:5"`
	Age    int      `validate:"required min:10 max:30"`
	Height float64  `validate:"required min:1.5 max:10.5"`
	List   []string `validate:"required min:3 max:5"`
	Dummy  string
}

type TestEqField struct {
	Name  string `validate:"eqfield:Name2"`
	Name2 string
}

type TestRegex struct {
	Age string `validate:"regexp:'^[0-9]+$'"`
}
