package main

import "errors"

func (s *TestStruct) Validate() error {
	if s.Name == "" {
		return errors.New("name can't be blank")
	}
	if len(s.Name) < 3 {
		return errors.New("name can't be less than 3")
	}
	if s.Age == 0 {
		return errors.New("age can't be blank")
	}
	if s.Age < 10 {
		return errors.New("age can't be less than 10")
	}
	if s.Height == 0 {
		return errors.New("height can't be blank")
	}
	if s.Height < 1.5 {
		return errors.New("height can't be less than 1.5")
	}
	if s.List == nil {
		return errors.New("list can't be blank")
	}
	if len(s.List) < 3 {
		return errors.New("list can't be less than 3")
	}
	return nil
}
