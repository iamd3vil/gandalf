package main

import "errors"

func (s *TestStruct) Validate() error {
	if s.Name == "" {
		return errors.New("Name can't be blank")
	}
	if s.Age == 0 {
		return errors.New("Age can't be blank")
	}
	if s.Height == 0 {
		return errors.New("Height can't be blank")
	}
	if s.List == nil {
		return errors.New("List can't be blank")
	}
	return nil
}
