// Code generated by gandalf(BuildDate: 2021-01-19 16:08:18, Version: 556307f (2021-01-19 14:28:29 +0530)). DO NOT EDIT.
package main

import (
	"errors"
	"regexp"
)

var (
	TestRegexAgeRegex = regexp.MustCompile("^[0-9]+$")
)

func (s *TestStruct) Validate() error {
	if s.Name == "" {
		return errors.New("name can't be blank")
	}
	if len(s.Name) < 3 {
		return errors.New("name can't be less than 3")
	}
	if len(s.Name) > 5 {
		return errors.New("name can't be greather than 5")
	}
	if s.Age == 0 {
		return errors.New("age can't be blank")
	}
	if s.Age < 10 {
		return errors.New("age can't be less than 10")
	}
	if s.Age > 30 {
		return errors.New("age can't be greather than 30")
	}
	if s.Height == 0 {
		return errors.New("height can't be blank")
	}
	if s.Height < 1.5 {
		return errors.New("height can't be less than 1.5")
	}
	if s.Height > 10.5 {
		return errors.New("height can't be greather than 10.5")
	}
	if s.List == nil {
		return errors.New("list can't be blank")
	}
	if len(s.List) < 3 {
		return errors.New("list can't be less than 3")
	}
	if len(s.List) > 5 {
		return errors.New("list can't be greather than 5")
	}
	return nil
}

func (s *TestEqField) Validate() error {
	if s.Name != s.Name2 {
		return errors.New("Name should be equal to Name2")
	}
	return nil
}

func (s *TestRegex) Validate() error {
	if !TestRegexAgeRegex.MatchString(s.Age) {
		return errors.New("Age doesn't match given regex")
	}
	return nil
}
