package main

type TestStruct struct {
	Name   string   `validate:"required min:3 max:5"`
	Age    int      `validate:"required"`
	Height float64  `validate:"required"`
	List   []string `validate:"required"`
}
