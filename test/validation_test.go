package main

import "testing"

func TestValidation(t *testing.T) {
	t.Run("empty-struct", func(t *testing.T) {
		s := TestStruct{}
		err := s.Validate()
		if err == nil {
			t.Fatalf("struct is supposed to be invalid")
		}
		t.Log(err)
	})
	t.Run("missing-string", func(t *testing.T) {
		s := TestStruct{
			Age:    25,
			Height: 5.5,
			List:   []string{"hello"},
		}
		err := s.Validate()
		if err == nil {
			t.Fatalf("struct is supposed to be invalid")
		}
		t.Log(err)
	})
	t.Run("missing-int", func(t *testing.T) {
		s := TestStruct{
			Name:   "test",
			Height: 5.5,
			List:   []string{"hello"},
		}
		err := s.Validate()
		if err == nil {
			t.Fatalf("struct is supposed to be invalid")
		}
		t.Log(err)
	})
	t.Run("missing-float", func(t *testing.T) {
		s := TestStruct{
			Name: "test",
			Age:  20,
			List: []string{"hello"},
		}
		err := s.Validate()
		if err == nil {
			t.Fatalf("struct is supposed to be invalid")
		}
		t.Log(err)
	})
	t.Run("missing-slice", func(t *testing.T) {
		s := TestStruct{
			Name:   "test",
			Age:    20,
			Height: 5.5,
		}
		err := s.Validate()
		if err == nil {
			t.Fatalf("struct is supposed to be invalid")
		}
		t.Log(err)
	})
	t.Run("valid-struct", func(t *testing.T) {
		s := TestStruct{
			Name:   "test",
			Age:    20,
			Height: 5.5,
			List:   []string{"hello"},
		}
		err := s.Validate()
		if err != nil {
			t.Fatalf("struct is supposed to be valid")
		}
	})
}
