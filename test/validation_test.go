package main

import "testing"

func TestValidationRequired(t *testing.T) {
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
			List:   []string{"hello1", "hello2", "hello3", "hello4"},
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
			List:   []string{"hello1", "hello2", "hello3", "hello4"},
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
			List:   []string{"hello1", "hello2", "hello3", "hello4"},
		}
		err := s.Validate()
		if err != nil {
			t.Fatalf("struct is supposed to be valid")
		}
	})
}

func TestValidationMin(t *testing.T) {
	s := TestStruct{
		Name:   "test",
		Age:    20,
		Height: 5.5,
		List:   []string{"hello1", "hello2", "hello3", "hello4"},
	}
	t.Run("invalid-string", func(t *testing.T) {
		name := s.Name
		defer func() {
			s.Name = name
		}()
		s.Name = "t"

		err := s.Validate()
		if err == nil {
			t.Fatalf("struct is supposed to be invalid")
		}
		t.Log(err)
	})
	t.Run("invalid-int", func(t *testing.T) {
		age := s.Age
		defer func() {
			s.Age = age
		}()
		s.Age = 5

		err := s.Validate()
		if err == nil {
			t.Fatalf("struct is supposed to be invalid")
		}
		t.Log(err)
	})
	t.Run("invalid-float", func(t *testing.T) {
		height := s.Height
		defer func() {
			s.Height = height
		}()
		s.Height = 0.5

		err := s.Validate()
		if err == nil {
			t.Fatalf("struct is supposed to be invalid")
		}
		t.Log(err)
	})
	t.Run("invalid-list", func(t *testing.T) {
		list := s.List
		defer func() {
			s.List = list
		}()
		s.List = []string{"onlyone"}
		err := s.Validate()
		if err == nil {
			t.Fatalf("struct is supposed to be invalid")
		}
		t.Log(err)
	})
}

func TestValidationMax(t *testing.T) {
	s := TestStruct{
		Name:   "test",
		Age:    20,
		Height: 5.5,
		List:   []string{"hello1", "hello2", "hello3", "hello4"},
	}
	t.Run("invalid-string", func(t *testing.T) {
		name := s.Name
		defer func() {
			s.Name = name
		}()
		s.Name = "test123467"

		err := s.Validate()
		if err == nil {
			t.Fatalf("struct is supposed to be invalid")
		}
		t.Log(err)
	})
	t.Run("invalid-int", func(t *testing.T) {
		age := s.Age
		defer func() {
			s.Age = age
		}()
		s.Age = 40

		err := s.Validate()
		if err == nil {
			t.Fatalf("struct is supposed to be invalid")
		}
		t.Log(err)
	})
	t.Run("invalid-float", func(t *testing.T) {
		height := s.Height
		defer func() {
			s.Height = height
		}()
		s.Height = 12.5

		err := s.Validate()
		if err == nil {
			t.Fatalf("struct is supposed to be invalid")
		}
		t.Log(err)
	})
	t.Run("invalid-list", func(t *testing.T) {
		list := s.List
		defer func() {
			s.List = list
		}()
		s.List = []string{"1", "2", "3", "4", "5", "6"}
		err := s.Validate()
		if err == nil {
			t.Fatalf("struct is supposed to be invalid")
		}
		t.Log(err)
	})
}
